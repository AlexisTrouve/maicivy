package services

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Tests des icônes de projets, contre un FAUX Gitea (httptest) — jamais la production.

// pngDe — un PNG uni de w×h px.
func pngDe(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// fauxGitea — dépôts → (chemin → contenu). Sert contents (racine et fichier) et media, exige le
// jeton, et compte les lectures par dépôt.
type fauxGitea struct {
	mu       sync.Mutex
	depots   map[string]map[string][]byte
	lectures map[string]int
	jeton    string
}

func (f *fauxGitea) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token "+f.jeton {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		reste, ok := strings.CutPrefix(r.URL.Path, "/api/v1/repos/StillHammer/")
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		repo, route, _ := strings.Cut(reste, "/")
		f.mu.Lock()
		f.lectures[repo]++
		fichiers, existe := f.depots[repo]
		f.mu.Unlock()
		if !existe {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch {
		case route == "contents":
			vus := map[string]bool{}
			var racine []giteaContenu
			for chemin := range fichiers {
				tete, _, dossier := strings.Cut(chemin, "/")
				if vus[tete] {
					continue
				}
				vus[tete] = true
				typ := "file"
				if dossier {
					typ = "dir"
				}
				racine = append(racine, giteaContenu{Name: tete, Type: typ, SHA: "sha-" + repo + "-" + tete})
			}
			_ = json.NewEncoder(w).Encode(racine)
		case strings.HasPrefix(route, "contents/"):
			chemin := strings.TrimPrefix(route, "contents/")
			if _, ok := fichiers[chemin]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(giteaContenu{Name: chemin, Type: "file", SHA: "sha-" + repo + "-" + strings.ReplaceAll(chemin, "/", "_")})
		case strings.HasPrefix(route, "media/"):
			data, ok := fichiers[strings.TrimPrefix(route, "media/")]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write(data)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

type fauxProjets []MPFProject

func (f fauxProjets) ListProjects(context.Context, string) ([]MPFProject, error) { return f, nil }

func projetRelie(id, repo string) MPFProject {
	return MPFProject{ID: id, Links: map[string]string{"repo": "git.etheryale.com/StillHammer/" + repo}}
}

// banc — un faux Gitea à cinq dépôts, les projets qui les citent, et le service.
func banc(t *testing.T, rdb *redis.Client) (*ProjectIconService, *fauxGitea) {
	t.Helper()
	g := &fauxGitea{
		jeton:    "jeton-test",
		lectures: map[string]int{},
		depots: map[string]map[string][]byte{
			"Avec":    {"icon.png": pngDe(t, 512, 256, color.NRGBA{200, 40, 40, 255}), "README.md": []byte("# lisez-moi")},
			"Sans":    {"README.md": []byte("# rien")},
			"Secret":  {"icon.png": pngDe(t, 64, 64, color.NRGBA{0, 0, 255, 255})},
			"NextApp": {"frontend/app/icon.png": pngDe(t, 48, 48, color.NRGBA{0, 200, 0, 255})},
			"Casse":   {"icon.png": []byte("version https://git-lfs.github.com/spec/v1\noid sha256:00")},
		},
	}
	srv := httptest.NewServer(g.handler(t))
	t.Cleanup(srv.Close)
	projets := fauxProjets{
		projetRelie("p_avec", "Avec"), projetRelie("p_sans", "Sans"), projetRelie("p_secret", "Secret"),
		projetRelie("p_next", "NextApp"), projetRelie("p_casse", "Casse"), {ID: "p_sans_depot"},
	}
	// « secret » en minuscules : la comparaison est sans casse, comme pour les stats.
	return newProjectIconService(projets, rdb, srv.URL, g.jeton, "StillHammer", "autre, secret"), g
}

func TestProjectIcons_IndexEtImages(t *testing.T) {
	s, g := banc(t, nil)
	ctx := context.Background()

	v := s.Versions(ctx)
	require.ElementsMatch(t, []string{"p_avec", "p_next"}, keys(v), "seuls les projets à icône lisible, hors dépôt exclu")

	data, version, ok := s.Icon(ctx, "p_avec")
	require.True(t, ok)
	require.Equal(t, v["p_avec"], version)
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err, "un PNG ré-encodé par le backend")
	require.Equal(t, image.Pt(iconeCote, iconeCote/2), img.Bounds().Size(), "réduit à 128 px, proportions gardées")
	r, _, _, a := img.At(10, 10).RGBA()
	require.InDelta(t, 200, r>>8, 1)
	require.Equal(t, uint32(0xffff), a)

	// Icône Next.js sous frontend/app/ : trouvée, et pas agrandie (48 px restent 48 px).
	data, _, ok = s.Icon(ctx, "p_next")
	require.True(t, ok)
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, 48, cfg.Width)

	// Sans icône, pointeur LFS non résolu (pas un PNG), sans dépôt relié, inconnu : rien.
	for _, id := range []string{"p_sans", "p_casse", "p_sans_depot", "inconnu"} {
		_, _, ok := s.Icon(ctx, id)
		require.False(t, ok, id)
	}

	// Le dépôt exclu n'est JAMAIS lu, pas même sa racine.
	require.Zero(t, g.lectures["Secret"])
}

func TestProjectIcons_ServiceNilSansIcone(t *testing.T) {
	var s *ProjectIconService = newProjectIconService(fauxProjets{}, nil, "", "", "", "")
	require.Nil(t, s, "sans jeton, pas de service")
	require.Empty(t, s.Versions(context.Background()))
	_, _, ok := s.Icon(context.Background(), "p_avec")
	require.False(t, ok)
}

// Premier visiteur : l'index n'existe pas encore → réponse IMMÉDIATE et vide (monogrammes), la
// construction part en fond ; ensuite les icônes sont servies depuis Redis sans relire Gitea.
func TestProjectIcons_JamaisDAttenteNiDeRelecture(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	s, g := banc(t, rdb)
	ctx := context.Background()

	require.Empty(t, s.Versions(ctx), "rien en cache : on n'attend pas Gitea")
	require.Eventually(t, func() bool { return len(s.Versions(ctx)) == 2 }, 5*time.Second, 20*time.Millisecond)

	g.mu.Lock()
	avant := g.lectures["Avec"]
	g.mu.Unlock()
	for i := 0; i < 5; i++ {
		_, _, ok := s.Icon(ctx, "p_avec")
		require.True(t, ok)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	require.Equal(t, avant, g.lectures["Avec"], "icône servie depuis le cache, aucune lecture Gitea de plus")
}

// Une icône évincée du cache mais toujours indexée est relue pour ce seul fichier.
func TestProjectIcons_IconeEvinceeRelue(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	s, _ := banc(t, rdb)
	ctx := context.Background()
	s.Versions(ctx)
	require.Eventually(t, func() bool { return len(s.Versions(ctx)) == 2 }, 5*time.Second, 20*time.Millisecond)
	for _, k := range mr.Keys() {
		if strings.HasPrefix(k, iconesPrefixePNG) {
			mr.Del(k)
		}
	}
	_, _, ok := s.Icon(ctx, "p_avec")
	require.True(t, ok)
}

func TestReduireIcone_Bornes(t *testing.T) {
	_, err := reduireIcone([]byte("<svg/>"))
	require.ErrorIs(t, err, errIconeInvalide, "seul le PNG est accepté")
	_, err = reduireIcone(pngDe(t, iconePixelsMax+1, 1, color.NRGBA{A: 255}))
	require.ErrorIs(t, err, errIconeInvalide, "dimensions hors bornes : pas décodé")

	// Bord transparent : la réduction ne l'assombrit pas (alpha prémultiplié).
	img := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			if x < 128 {
				img.Set(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	out, err := reduireIcone(buf.Bytes())
	require.NoError(t, err)
	dec, err := png.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	c := color.NRGBAModel.Convert(dec.At(63, 10)).(color.NRGBA)
	require.Equal(t, uint8(255), c.R, "blanc reste blanc au bord de la transparence")
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
