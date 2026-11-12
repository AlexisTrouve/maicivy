package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ============================================================================
// MontagneService — les projets d'Alexis en montagne : strates, liens, positions, visuels.
// ----------------------------------------------------------------------------
// QUOI : assemble, depuis maiProFiles, tout ce que la page /[locale]/montagne affiche — un nœud
// par projet visible (nom court, mini, statut, strate, position 3D, monogramme, visuels, curé ou
// non) et les liens « sert à construire ». Servi par GET /api/v1/montagne?lang=xx.
//
// POURQUOI un endpoint dédié : la page rend côté serveur la liste des projets (SEO, sans-JS) ET
// confie la même donnée au canvas 3D. Une seule lecture, déjà calculée : le front ne fait que
// dessiner.
//
// COMMENT :
//   - STRUCTURE (strates, liens, positions) calculée sur l'édition ANGLAISE, quelle que soit la
//     langue demandée : les tournures de la règle 2 des liens sont anglaises (montagne_liens.go),
//     et la montagne doit être la même pour tous les visiteurs.
//   - TEXTES (nom, mini) superposés depuis l'index de la langue demandée. Un mini identique au
//     mini français dans une autre langue est un mini NON traduit (maiProFiles renvoie le
//     français tant que la traduction n'est pas faite) : il est vidé, jamais affiché.
//   - CURATION et VISUELS : le livre (`/book?web=true`) dit quels projets ont une page
//     /portfolio/[id] et porte leurs captures et vidéo ; le store d'images (`/images`) ajoute les
//     captures rattachées aux projets non curés.
//   - CACHE : swrStore (Redis, stale-while-revalidate), une clé par langue, aucune expiration,
//     fraîcheur 10 min, rafraîchissements en fond plafonnés par limiteFondMaiProFiles. Aucun
//     préchauffage au démarrage : le premier visiteur d'une langue paie une lecture, les suivants
//     jamais (exigence : aucun appel à maiProFiles au démarrage du backend).
//   - Toute lecture en échec fait échouer l'assemblage (502 côté handler) : une montagne sans
//     curation montrerait des projets sans leur bouton, sans le dire. Le périmé en cache, lui,
//     continue d'être servi.
// ============================================================================

const (
	// montagneCachePrefix — versionné : un changement de forme du blob se fait par bump de clé.
	montagneCachePrefix = "montagne:v4" // v4 : `summit` porte les cartes et leurs preuves
	montagneFraicheur   = 10 * time.Minute
	// visuelsMax — pétales d'images au plus autour de la fleur (la vidéo s'y ajoute) : au-delà,
	// les pétales se recouvrent sur un écran de téléphone.
	visuelsMax = 6
	// langueStructure — édition sur laquelle la structure est calculée (cf. en-tête).
	langueStructure = "en"
)

// langsMontagne — langues où maiProFiles sert un contenu natif exposé par le front. `ka` existe
// côté maiProFiles mais pas côté front ; de/it retombent sur l'anglais (jamais le français).
var langsMontagne = map[string]bool{"fr": true, "en": true, "zh": true}

func langueMontagne(lang string) string {
	if langsMontagne[lang] {
		return lang
	}
	return fallbackLang
}

// MontagneVisuel — un pétale de la fleur : une image ou une vidéo, chemin servi par maiProFiles
// (le front le préfixe, comme pour le portfolio).
type MontagneVisuel struct {
	Kind      string `json:"kind"` // "image" | "video"
	URL       string `json:"url"`
	PosterURL string `json:"poster_url,omitempty"`
	// Vector — image SVG. Le front la sert telle quelle : l'optimiseur d'images de Next refuse le
	// SVG, et l'URL du store (`/images/<id>`) ne porte pas d'extension pour le deviner.
	Vector bool   `json:"vector,omitempty"`
	Alt    string `json:"alt"`
}

// MontagneNoeud — un projet sur la montagne.
type MontagneNoeud struct {
	ID       string `json:"id"`
	Name     string `json:"name"` // nom court (identité), dans la langue demandée
	Mini     string `json:"mini"` // vide si non traduit dans cette langue
	Status   string `json:"status"`
	Stratum  Strate `json:"stratum"`
	Monogram string `json:"monogram"`
	// IconURL — l'icône définitive (produite par Eikon, à venir). Vide = le front dessine le
	// monogramme. Le champ existe dès maintenant pour que le front n'ait rien à changer le jour
	// où il sera renseigné.
	IconURL string           `json:"icon_url"`
	Curated bool             `json:"curated"` // a une page /portfolio/[id]
	Visuals []MontagneVisuel `json:"visuals"`
	PositionNoeud
}

// Montagne — la réponse complète.
type Montagne struct {
	Lang   string           `json:"lang"`
	Strata []MontagneStrate `json:"strata"`
	// Shape — le cône portant les projets (silhouette et courbes de niveau côté front).
	Shape MontagneForme   `json:"shape"`
	Nodes []MontagneNoeud `json:"nodes"`
	Links []MontagneLien  `json:"links"`
	// Summit — les cartes du sommet, dans leur ordre d'affichage, avec leurs preuves
	// (montagne_sommet.go).
	Summit []MontagneCarte `json:"summit"`
}

// montagneSource — les lectures maiProFiles nécessaires. Interface pour tester l'assemblage
// contre une source en mémoire.
type montagneSource interface {
	ListProjects(ctx context.Context, lang string) ([]MPFProject, error)
	GetProjectsIndex(ctx context.Context, lang string) ([]MPFProjectIndexEntry, error)
	GetPortfolioBook(ctx context.Context, lang string) (*MPFBook, error)
	ListImages(ctx context.Context) ([]MPFStoreImage, error)
}

// MontagneService — voir l'en-tête du fichier.
type MontagneService struct {
	source montagneSource
	store  *swrStore
}

// NewMontagneService — rdb peut être nil (tests, Redis absent) : le cache devient transparent.
func NewMontagneService(source *MaiProFilesClient, rdb *redis.Client) *MontagneService {
	return newMontagneService(source, rdb)
}

func newMontagneService(source montagneSource, rdb *redis.Client) *MontagneService {
	return &MontagneService{
		source: source,
		store: &swrStore{
			redis:     rdb,
			nom:       "montagne",
			fraicheur: montagneFraicheur,
			ttl:       0, // trois langues au plus : clés en nombre fini
			limite:    limiteFondMaiProFiles,
		},
	}
}

// Get — la montagne dans la langue demandée (normalisée avant la clé : de/it partagent l'anglais).
func (s *MontagneService) Get(ctx context.Context, lang string) (*Montagne, error) {
	lang = langueMontagne(lang)
	var m Montagne
	err := s.store.lire(ctx, fmt.Sprintf("%s:%s", montagneCachePrefix, lang), &m, func(c context.Context) (interface{}, error) {
		return construireMontagne(c, s.source, lang)
	})
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// construireMontagne — l'assemblage, sans cache. Voir l'en-tête pour les étapes.
func construireMontagne(ctx context.Context, src montagneSource, lang string) (*Montagne, error) {
	// 1. Structure, sur l'édition anglaise.
	projets, err := src.ListProjects(ctx, langueStructure)
	if err != nil {
		return nil, fmt.Errorf("montagne: lecture des projets : %w", err)
	}
	structure := make([]projetStructure, 0, len(projets))
	strates := make(map[string]Strate, len(projets))
	categories := make(map[string]string, len(projets))
	for _, p := range projets {
		categories[p.ID] = p.Category
		structure = append(structure, projetStructure{
			ID: p.ID, Name: p.Name, Category: p.Category, Tags: p.Tags, Stack: p.Stack,
			Description: p.Description.Short,
		})
		strates[p.ID] = strateDuProjet(p.Category, p.Tags)
	}
	liens := deduireLiens(structure)
	positions, bandes, forme := calculerPositions(strates, liens)

	// 2. Textes de la langue demandée.
	textes, err := textesLocalises(ctx, src, lang)
	if err != nil {
		return nil, err
	}

	// 3. Curation et visuels.
	livre, err := src.GetPortfolioBook(ctx, lang)
	if err != nil {
		return nil, fmt.Errorf("montagne: lecture du livre : %w", err)
	}
	images, err := src.ListImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("montagne: lecture du store d'images : %w", err)
	}
	visuels, curated := visuelsParProjet(livre, images)

	noeuds := make([]MontagneNoeud, 0, len(projets))
	for _, p := range projets {
		t, ok := textes[p.ID]
		if !ok {
			// Projet absent de l'index de cette langue : son nom anglais, sans mini — jamais un
			// texte d'une autre langue présenté comme traduit.
			t = texteNoeud{nom: p.Name}
		}
		// Tableau vide et non null : le front lit `visuals.length` sans garde.
		petales := visuels[p.ID]
		if petales == nil {
			petales = []MontagneVisuel{}
		}
		noeuds = append(noeuds, MontagneNoeud{
			ID:            p.ID,
			Name:          nomIdentite(t.nom),
			Mini:          t.mini,
			Status:        p.Status,
			Stratum:       strates[p.ID],
			Monogram:      monogramme(p.Name), // sur le nom anglais : même monogramme dans toutes les langues
			IconURL:       "",
			Curated:       curated[p.ID],
			Visuals:       petales,
			PositionNoeud: positions[p.ID],
		})
	}
	sort.Slice(noeuds, func(i, j int) bool { return noeuds[i].ID < noeuds[j].ID })
	return &Montagne{
		Lang: lang, Strata: bandes, Shape: forme, Nodes: noeuds, Links: liens,
		Summit: construireSommet(noeuds, liens, categories),
	}, nil
}

type texteNoeud struct{ nom, mini string }

// textesLocalises — nom et mini par id dans `lang`, minis non traduits vidés (cf. en-tête).
func textesLocalises(ctx context.Context, src montagneSource, lang string) (map[string]texteNoeud, error) {
	index, err := src.GetProjectsIndex(ctx, lang)
	if err != nil {
		return nil, fmt.Errorf("montagne: lecture de l'index (%s) : %w", lang, err)
	}
	minisFr := map[string]string{}
	if lang != "fr" {
		fr, err := src.GetProjectsIndex(ctx, "fr")
		if err != nil {
			return nil, fmt.Errorf("montagne: lecture de l'index (fr) : %w", err)
		}
		for _, e := range fr {
			minisFr[e.ID] = strings.TrimSpace(e.Mini)
		}
	}
	out := make(map[string]texteNoeud, len(index))
	for _, e := range index {
		mini := strings.TrimSpace(e.Mini)
		if lang != "fr" && mini == minisFr[e.ID] {
			mini = "" // non traduit : du français sur une montagne anglaise serait un mensonge de langue
		}
		out[e.ID] = texteNoeud{nom: e.Name, mini: mini}
	}
	return out, nil
}

// visuelsParProjet — les pétales de chaque projet et l'ensemble des projets curés.
//
// Ordre des pétales : image principale du livre, galerie du livre, captures du store rattachées
// au projet, puis UNE vidéo (celle du livre, sinon la première du store). Plafonnés à visuelsMax
// images.
// Dédupliqués par NOM DE FICHIER (cleVisuel), pas par URL : mesuré sur grove_engine, le store
// contient les mêmes captures que le livre, téléversées une seconde fois (`/images/img_grove_3ee9`
// a pour nom d'origine `Grove.jpg`, déjà servi en `/portfolio-images/Grove.jpg`). Par URL, la
// fleur montrait deux fois la même image.
func visuelsParProjet(livre *MPFBook, images []MPFStoreImage) (map[string][]MontagneVisuel, map[string]bool) {
	curated := map[string]bool{}
	visuels := map[string][]MontagneVisuel{}
	videos := map[string]*MontagneVisuel{}
	vus := map[string]map[string]bool{}

	ajouterImage := func(id, url, nomFichier, alt string, vector bool) {
		if url == "" {
			return
		}
		if vus[id] == nil {
			vus[id] = map[string]bool{}
		}
		cle := cleVisuel(nomFichier)
		if vus[id][cle] || len(visuels[id]) >= visuelsMax {
			return
		}
		vus[id][cle] = true
		visuels[id] = append(visuels[id], MontagneVisuel{Kind: "image", URL: url, Alt: alt, Vector: vector})
	}

	for _, p := range livre.Projects {
		if p.ID == "" {
			continue
		}
		curated[p.ID] = true
		ajouterImage(p.ID, p.Image.URL, p.Image.URL, p.Image.Alt, cheminSVG(p.Image.URL))
		for _, g := range p.Images {
			alt := g.Caption
			if alt == "" {
				alt = g.Alt
			}
			ajouterImage(p.ID, g.URL, g.URL, alt, cheminSVG(g.URL))
		}
		if p.Video != nil && p.Video.URL != "" {
			videos[p.ID] = &MontagneVisuel{Kind: "video", URL: p.Video.URL, PosterURL: p.Video.PosterURL, Alt: p.Video.Caption}
		}
	}

	for _, im := range images {
		if im.LinkedTo.Type != "project" || im.LinkedTo.ID == "" || im.ID == "" {
			continue
		}
		id, url := im.LinkedTo.ID, "/images/"+im.ID
		if strings.HasPrefix(im.ContentType, "video/") {
			if videos[id] == nil {
				videos[id] = &MontagneVisuel{Kind: "video", URL: url, Alt: im.Alt}
			}
			continue
		}
		if strings.HasPrefix(im.ContentType, "image/") {
			nom := im.OriginalName
			if nom == "" {
				nom = url // sans nom d'origine, l'image est unique par son id
			}
			ajouterImage(id, url, nom, im.Alt, im.ContentType == "image/svg+xml")
		}
	}

	for id, v := range videos {
		visuels[id] = append(visuels[id], *v)
	}
	return visuels, curated
}

// cheminSVG — vrai si le chemin (query string exclue) désigne un fichier .svg. Les images du livre
// portent leur extension (« /portfolio-images/x.svg?v=… ») ; celles du store passent par leur
// content_type.
func cheminSVG(url string) bool {
	if i := strings.IndexByte(url, '?'); i >= 0 {
		url = url[:i]
	}
	return strings.HasSuffix(strings.ToLower(url), ".svg")
}

// cleVisuel — identité d'une image pour la déduplication : nom de fichier sans dossier, sans
// query string, sans extension, en minuscules (« /portfolio-images/Grove.jpg?v=1 » et « Grove.png »
// donnent « grove »).
func cleVisuel(chemin string) string {
	if i := strings.IndexByte(chemin, '?'); i >= 0 {
		chemin = chemin[:i]
	}
	if i := strings.LastIndexByte(chemin, '/'); i >= 0 {
		chemin = chemin[i+1:]
	}
	if i := strings.LastIndexByte(chemin, '.'); i > 0 {
		chemin = chemin[:i]
	}
	return strings.ToLower(chemin)
}
