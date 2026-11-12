package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// ============================================================================
// ProjectIconService — l'icône d'un projet, prise dans SON dépôt Gitea (`icon.png`).
// ----------------------------------------------------------------------------
// QUOI : pour chaque projet de maiProFiles relié à un dépôt Gitea (links.repo), cherche un
// `icon.png` à quelques emplacements CONNUS du dépôt, le télécharge, le réduit à iconeCote px et le
// garde en cache. Servi par GET /api/v1/project-icons (quels projets en ont une) et
// GET /api/v1/project-icons/:id (l'image), et lu par le PDF du dossier vivant.
//
// POURQUOI côté serveur : les dépôts sont PRIVÉS — seul le jeton Gitea du backend
// (GITEA_STATS_TOKEN) peut les lire, et il ne doit jamais atteindre un navigateur. Relevé du
// 2026-11-05 : 19 dépôts sur 159 ont un icon.png (16 à la racine, sinon `assets/`, `frontend/app/`,
// `site/src/app/`), 13 d'entre eux reliés à un projet ; les icônes pèsent 0,9 à 2,4 Mo (d'où la
// réduction), et deux sont des pointeurs Git LFS (d'où la route `/media/`, qui les résout).
//
// COMMENT — ce qu'une route publique ne doit PAS permettre :
//   - lire un fichier arbitraire : le client ne donne qu'un ID DE PROJET ; le chemin lu dans le
//     dépôt vient de la liste fixe `iconeChemins`, jamais de la requête ;
//   - nommer un dépôt non affiché : seuls les projets listés par maiProFiles (ceux de la montagne)
//     ont une entrée, et aucune réponse ne porte un nom de dépôt ;
//   - contourner GITEA_STATS_EXCLUDE : un dépôt exclu n'est jamais LU (pas même sa racine) ;
//   - servir autre chose qu'une image : l'icône est DÉCODÉE puis ré-encodée en PNG — un fichier qui
//     ne décode pas n'entre pas dans l'index, et rien d'autre que nos propres pixels ne sort.
//
// CACHE (aucun appel Gitea en avant-plan à chaque visite) :
//   - l'INDEX (id → dépôt, chemin, sha du blob) : swrStore, clé `projicons:v1:index`, fraîcheur 6 h,
//     servi périmé et reconstruit en fond. Absent → construit EN FOND et l'appelant reçoit un index
//     vide (lireSansAttendre) : le premier visiteur voit les monogrammes, jamais une page qui attend ;
//   - les IMAGES réduites : une clé par sha de blob (`projicons:v1:png:<sha>`) — adressées par leur
//     contenu, donc jamais périmées ; téléchargées pendant la construction de l'index, en fond.
// ============================================================================

const (
	iconesCleIndex   = "projicons:v1:index"
	iconesPrefixePNG = "projicons:v1:png:"
	iconesFraicheur  = 6 * time.Hour
	// iconesTTLPNG — une icône remplacée dans son dépôt change de sha : l'ancienne clé n'est plus lue
	// et disparaît d'elle-même.
	iconesTTLPNG = 30 * 24 * time.Hour
	// iconeCote — côté maximal de l'icône servie (px). Le plus grand affichage est la carte de preuve
	// du dossier (34 px CSS) sur un écran 3× : 102 px. Au-delà, on paierait des octets invisibles.
	iconeCote = 128
	// iconeOctetsMax / iconePixelsMax — bornes d'une entrée externe : au-delà, on ne décode pas
	// (un PNG de 30 000 × 30 000 px tient en quelques Ko compressés et en 3,6 Go décodés).
	iconeOctetsMax = 8 << 20
	iconePixelsMax = 4096
	// iconesParallele — lectures Gitea simultanées pendant la construction. Gitea partage sa machine
	// avec maiProFiles : panne du 2026-10-09, 30 lectures parallèles l'ont effondré.
	iconesParallele = 4
)

// iconeChemins — les SEULS emplacements lus, dans l'ordre de préférence : la racine d'abord (16 des
// 19 dépôts du relevé), puis les conventions rencontrées (dossier d'assets, icône d'app Next.js).
var iconeChemins = []string{
	"icon.png",
	"assets/icon.png",
	"app/icon.png",
	"src/app/icon.png",
	"public/icon.png",
	"frontend/app/icon.png",
	"site/src/app/icon.png",
}

// projectIconProjects — d'où viennent les projets affichés (maiProFiles, cache mémoire du client).
type projectIconProjects interface {
	ListProjects(ctx context.Context, lang string) ([]MPFProject, error)
}

// iconeEntree — un projet qui a une icône. Ne sort JAMAIS du backend (il nomme le dépôt).
type iconeEntree struct {
	Repo string `json:"r"`
	Path string `json:"p"`
	SHA  string `json:"s"`
}

// version — courte empreinte du blob, posée dans l'URL de l'icône : une icône remplacée change
// d'URL, le navigateur peut donc garder l'ancienne en cache sans risque.
func (e iconeEntree) version() string {
	if len(e.SHA) > 12 {
		return e.SHA[:12]
	}
	return e.SHA
}

// ProjectIconService — voir l'en-tête du fichier. Un pointeur nil est un service valide qui n'a
// aucune icône (Gitea non configuré) : les appelants n'ont pas à le tester.
type ProjectIconService struct {
	projets projectIconProjects
	baseURL string
	token   string
	owner   string
	client  *http.Client
	exclus  map[string]bool
	store   *swrStore
	redis   *redis.Client
}

// NewProjectIconService — nil si le jeton ou l'adresse Gitea manque (même convention que
// GiteaStatsService). `exclude` = GITEA_STATS_EXCLUDE.
func NewProjectIconService(projets *MaiProFilesClient, rdb *redis.Client, baseURL, token, owner, exclude string) *ProjectIconService {
	return newProjectIconService(projets, rdb, baseURL, token, owner, exclude)
}

func newProjectIconService(projets projectIconProjects, rdb *redis.Client, baseURL, token, owner, exclude string) *ProjectIconService {
	if token == "" || baseURL == "" {
		return nil
	}
	if owner == "" {
		owner = "StillHammer"
	}
	return &ProjectIconService{
		projets: projets,
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		owner:   owner,
		client:  &http.Client{Timeout: 30 * time.Second},
		exclus:  depotsExclus(exclude),
		store:   &swrStore{redis: rdb, nom: "project-icons", fraicheur: iconesFraicheur},
		redis:   rdb,
	}
}

// Versions — id de projet → version de son icône, pour les seuls projets qui en ont une. Vide tant
// que l'index n'est pas construit (jamais d'attente : cf. en-tête).
func (s *ProjectIconService) Versions(ctx context.Context) map[string]string {
	out := map[string]string{}
	for id, e := range s.index(ctx) {
		out[id] = e.version()
	}
	return out
}

// Icon — l'icône réduite (PNG) du projet `id` et sa version ; ok=false si le projet n'en a pas
// (ou n'est pas un projet affiché, ou vit dans un dépôt exclu : indiscernables, à dessein).
func (s *ProjectIconService) Icon(ctx context.Context, id string) (data []byte, version string, ok bool) {
	e, ok := s.index(ctx)[id]
	if !ok {
		return nil, "", false
	}
	if data := s.pngEnCache(ctx, e.SHA); data != nil {
		return data, e.version(), true
	}
	// Indexée mais absente du cache (clé évincée) : relue pour CE fichier seulement.
	data, err := s.telecharger(ctx, e)
	if err != nil {
		log.Warn().Err(err).Str("projet", id).Msg("project-icons: icône indexée illisible")
		return nil, "", false
	}
	s.stockerPNG(ctx, e.SHA, data)
	return data, e.version(), true
}

func (s *ProjectIconService) index(ctx context.Context) map[string]iconeEntree {
	if s == nil {
		return nil
	}
	var idx map[string]iconeEntree
	trouve, err := s.store.lireSansAttendre(ctx, iconesCleIndex, &idx, s.construireIndex)
	if err != nil {
		log.Warn().Err(err).Msg("project-icons: index indisponible, repli sur les monogrammes")
		return nil
	}
	if !trouve {
		return nil
	}
	return idx
}

// construireIndex — un passage sur tous les projets reliés à un dépôt non exclu. Une erreur de
// TRANSPORT (Gitea injoignable, 5xx) fait échouer la construction : l'index précédent reste servi.
// Une icône qui ne se décode pas n'est qu'écartée.
func (s *ProjectIconService) construireIndex(ctx context.Context) (interface{}, error) {
	projets, err := s.projets.ListProjects(ctx, langueStructure)
	if err != nil {
		return nil, fmt.Errorf("project-icons: projets : %w", err)
	}
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		index    = map[string]iconeEntree{}
		premiere error
		places   = make(chan struct{}, iconesParallele)
	)
	for _, p := range projets {
		repo := repoFromLinks(p.Links)
		if p.ID == "" || repo == "" || s.exclus[strings.ToLower(repo)] {
			continue
		}
		wg.Add(1)
		go func(id, repo string) {
			defer wg.Done()
			places <- struct{}{}
			defer func() { <-places }()
			e, ok, err := s.preparerIcone(ctx, repo)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil && premiere == nil:
				premiere = err
			case ok:
				index[id] = e
			}
		}(p.ID, repo)
	}
	wg.Wait()
	if premiere != nil {
		return nil, premiere
	}
	log.Info().Int("icones", len(index)).Msg("project-icons: index construit")
	return index, nil
}

// preparerIcone — cherche l'icône du dépôt, et s'assure que sa version réduite est en cache.
func (s *ProjectIconService) preparerIcone(ctx context.Context, repo string) (iconeEntree, bool, error) {
	e, ok, err := s.trouver(ctx, repo)
	if err != nil || !ok {
		return e, ok, err
	}
	if s.pngEnCache(ctx, e.SHA) != nil {
		return e, true, nil
	}
	data, err := s.telecharger(ctx, e)
	if errors.Is(err, errIconeInvalide) {
		log.Warn().Err(err).Str("repo", repo).Str("chemin", e.Path).Msg("project-icons: icon.png écarté")
		return e, false, nil
	}
	if err != nil {
		return e, false, err
	}
	s.stockerPNG(ctx, e.SHA, data)
	return e, true, nil
}

// giteaContenu — une entrée de GET /repos/{owner}/{repo}/contents[/{chemin}].
type giteaContenu struct {
	Name string `json:"name"`
	Type string `json:"type"` // file | dir | symlink | submodule
	SHA  string `json:"sha"`
}

// trouver — le premier de `iconeChemins` présent dans le dépôt. Une lecture de la racine, plus une
// par emplacement candidat dont le premier dossier existe : un à trois appels par dépôt.
func (s *ProjectIconService) trouver(ctx context.Context, repo string) (iconeEntree, bool, error) {
	var racine []giteaContenu
	trouve, err := s.lireJSON(ctx, s.cheminContenu(repo, ""), &racine)
	if err != nil || !trouve {
		return iconeEntree{}, false, err // dépôt vide ou disparu : pas d'icône, pas d'erreur
	}
	presents := map[string]giteaContenu{}
	for _, c := range racine {
		presents[c.Name] = c
	}
	for _, chemin := range iconeChemins {
		tete, reste, sousDossier := strings.Cut(chemin, "/")
		if !sousDossier {
			if c, ok := presents[tete]; ok && c.Type == "file" {
				return iconeEntree{Repo: repo, Path: chemin, SHA: c.SHA}, true, nil
			}
			continue
		}
		if presents[tete].Type != "dir" || reste == "" {
			continue
		}
		var c giteaContenu
		trouve, err := s.lireJSON(ctx, s.cheminContenu(repo, chemin), &c)
		if err != nil {
			return iconeEntree{}, false, err
		}
		if trouve && c.Type == "file" {
			return iconeEntree{Repo: repo, Path: chemin, SHA: c.SHA}, true, nil
		}
	}
	return iconeEntree{}, false, nil
}

// errIconeInvalide — le fichier existe mais n'est pas une icône utilisable (trop lourd, pas un PNG,
// dimensions hors bornes). Écarté, sans faire échouer l'index.
var errIconeInvalide = errors.New("icône invalide")

// telecharger — le fichier par la route `/media/` (qui résout les pointeurs Git LFS, contrairement
// à `/raw/`), borné en taille, puis réduit.
func (s *ProjectIconService) telecharger(ctx context.Context, e iconeEntree) ([]byte, error) {
	resp, err := s.get(ctx, fmt.Sprintf("/repos/%s/%s/media/%s", url.PathEscape(s.owner), url.PathEscape(e.Repo), echapperChemin(e.Path)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w : %s absent", errIconeInvalide, e.Path)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitea media %s/%s : %d", e.Repo, e.Path, resp.StatusCode)
	}
	brut, err := io.ReadAll(io.LimitReader(resp.Body, iconeOctetsMax+1))
	if err != nil {
		return nil, err
	}
	if len(brut) > iconeOctetsMax {
		return nil, fmt.Errorf("%w : plus de %d octets", errIconeInvalide, iconeOctetsMax)
	}
	return reduireIcone(brut)
}

// lireJSON — GET JSON ; trouve=false sur 404 (ressource absente, pas une panne).
func (s *ProjectIconService) lireJSON(ctx context.Context, chemin string, out interface{}) (bool, error) {
	resp, err := s.get(ctx, chemin)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("gitea %s : %d", chemin, resp.StatusCode)
	}
	return true, json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
}

func (s *ProjectIconService) get(ctx context.Context, chemin string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/api/v1"+chemin, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+s.token)
	return s.client.Do(req)
}

func (s *ProjectIconService) cheminContenu(repo, chemin string) string {
	base := fmt.Sprintf("/repos/%s/%s/contents", url.PathEscape(s.owner), url.PathEscape(repo))
	if chemin == "" {
		return base
	}
	return base + "/" + echapperChemin(chemin)
}

// echapperChemin — échappe chaque segment d'un chemin de la liste fixe (les « / » restent).
func echapperChemin(chemin string) string {
	segs := strings.Split(chemin, "/")
	for i, seg := range segs {
		segs[i] = url.PathEscape(seg)
	}
	return strings.Join(segs, "/")
}

func (s *ProjectIconService) pngEnCache(ctx context.Context, sha string) []byte {
	if s.redis == nil {
		return nil
	}
	data, err := s.redis.Get(ctx, iconesPrefixePNG+sha).Bytes()
	if err != nil {
		return nil
	}
	return data
}

func (s *ProjectIconService) stockerPNG(ctx context.Context, sha string, data []byte) {
	if s.redis == nil {
		return
	}
	if err := s.redis.Set(ctx, iconesPrefixePNG+sha, data, iconesTTLPNG).Err(); err != nil {
		log.Warn().Err(err).Msg("project-icons: écriture du cache échouée")
	}
}

// reduireIcone — décode un PNG (et RIEN d'autre), le réduit pour tenir dans iconeCote × iconeCote
// (jamais agrandi), le ré-encode.
// COMMENT : moyenne par zone (box filter) en alpha prémultiplié — la réduction d'une icône à fond
// transparent ne fait pas de halo sombre sur ses bords. Pas de dépendance d'image en plus pour ça.
func reduireIcone(brut []byte) ([]byte, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(brut))
	if err != nil {
		return nil, fmt.Errorf("%w : %v", errIconeInvalide, err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > iconePixelsMax || cfg.Height > iconePixelsMax {
		return nil, fmt.Errorf("%w : %d×%d px", errIconeInvalide, cfg.Width, cfg.Height)
	}
	src, err := png.Decode(bytes.NewReader(brut))
	if err != nil {
		return nil, fmt.Errorf("%w : %v", errIconeInvalide, err)
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	dw, dh := sw, sh
	if sw > iconeCote || sh > iconeCote {
		if sw >= sh {
			dw, dh = iconeCote, max(1, sh*iconeCote/sw)
		} else {
			dw, dh = max(1, sw*iconeCote/sh), iconeCote
		}
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		y0, y1 := b.Min.Y+y*sh/dh, b.Min.Y+max((y+1)*sh/dh, y*sh/dh+1)
		for x := 0; x < dw; x++ {
			x0, x1 := b.Min.X+x*sw/dw, b.Min.X+max((x+1)*sw/dw, x*sw/dw+1)
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA() // prémultiplié, 16 bits
					r, g, bl, a, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca), n+1
				}
			}
			i := dst.PixOffset(x, y)
			if a == 0 {
				continue // transparent : déjà à zéro
			}
			dst.Pix[i+0] = uint8(r * 0xff / a)
			dst.Pix[i+1] = uint8(g * 0xff / a)
			dst.Pix[i+2] = uint8(bl * 0xff / a)
			dst.Pix[i+3] = uint8(a / n >> 8)
		}
	}
	var out bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&out, dst); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
