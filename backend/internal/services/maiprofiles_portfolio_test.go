package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLangueDuLivre : les éditions curées du book ne sont PAS les langues du reste de
// maiProFiles. Le contenu général existe en fr/en/ka ; le livre, lui, est curé en en/fr/zh.
// Utiliser la table générale replierait le chinois sur l'anglais et ferait disparaître une
// édition qui existe pourtant — d'où une table dédiée.
func TestLangueDuLivre(t *testing.T) {
	for demandee, attendue := range map[string]string{
		"en": "en", "fr": "fr", "zh": "zh",
		"de": "en", "it": "en", "ka": "en", "": "en",
	} {
		require.Equal(t, attendue, langueDuLivre(demandee), "langue demandée %q", demandee)
	}
}

// serveurLivre monte un faux maiProFiles qui note les chemins appelés.
func serveurLivre(t *testing.T, appels *[]string) *MaiProFilesClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*appels = append(*appels, r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/book":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"lang": r.URL.Query().Get("lang"), "title": "Portfolio",
				"projects": []map[string]interface{}{{
					"num": 1, "anchor": "p1", "id": "coco_seo", "name": "CoconSEO",
					"category": "SaaS", "index_domain": "SaaS", "index_stack": "Rust",
					"image": map[string]interface{}{"src": "../portfolio_images/a.jpg", "alt": "A", "url": "/portfolio-images/a.jpg"},
					"images": []map[string]interface{}{
						{"src": "../portfolio_images/a.jpg", "alt": "A", "url": "/portfolio-images/a.jpg", "caption": "hero"},
						{"src": "/images/img_b", "alt": "B", "url": "/images/img_b", "caption": "architecture"},
					},
					"paragraphs": []string{"p1", "p2", "p3"},
					"badges":     []map[string]string{{"label": "Rust", "class": "rust"}},
					"stats":      []map[string]string{{"value": "33 000+", "label": " lignes Rust"}},
				}},
			})
		case "/portfolio/coco_seo":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "coco_seo", "name": "CoconSEO", "paragraphs": []string{"p1", "p2", "p3"},
				"description_portfolio": "# CoconSEO\n\n## The Problem",
				"links":                 map[string]string{"repo": "git.etheryale.com/x"},
				"status":                "production",
				"image":                 map[string]interface{}{"url": "/portfolio-images/a.jpg"},
				"video": map[string]string{
					"src": "/images/img_v", "poster": "/images/img_p", "caption": "quatre agents",
					"url": "/images/img_v", "poster_url": "/images/img_p",
				},
				"gallery": []map[string]interface{}{
					{"id": "img_1", "alt": "écran principal", "linked_to": map[string]string{"section": "screenshot"}},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detail":"introuvable"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return &MaiProFilesClient{http: srv.Client(), baseURL: srv.URL}
}

// TestGetPortfolioBook_DemandeLesChampsDuWeb : sans `web=true`, la réponse ne porte ni `id` ni
// `image.url` — le site ne saurait donc ni router vers une fiche ni afficher une capture.
// Ce paramètre est délibérément absent par défaut côté API : il casserait la régénération du PDF.
func TestGetPortfolioBook_DemandeLesChampsDuWeb(t *testing.T) {
	var appels []string
	livre, err := serveurLivre(t, &appels).GetPortfolioBook(context.Background(), "en")

	require.NoError(t, err)
	require.Contains(t, appels[0], "web=true")
	require.Len(t, livre.Projects, 1)
	require.Equal(t, "coco_seo", livre.Projects[0].ID)
	require.Equal(t, "/portfolio-images/a.jpg", livre.Projects[0].Image.URL)
	require.Len(t, livre.Projects[0].Paragraphs, 3)
	// La galerie : plusieurs visuels par projet, chacun avec sa légende. Elle n'existe que sur
	// la vue web — le modèle du PDF ne la porte pas.
	require.Len(t, livre.Projects[0].Images, 2)
	require.Equal(t, "architecture", livre.Projects[0].Images[1].Caption)
	require.Equal(t, "/images/img_b", livre.Projects[0].Images[1].URL)
}

// TestGetPortfolioBook_ChinoisNonReplie : le piège. L'édition chinoise EXISTE ; la replier sur
// l'anglais servirait un livre que le lecteur ne peut pas lire alors qu'il est traduit.
func TestGetPortfolioBook_ChinoisNonReplie(t *testing.T) {
	var appels []string
	_, err := serveurLivre(t, &appels).GetPortfolioBook(context.Background(), "zh")

	require.NoError(t, err)
	require.Contains(t, appels[0], "lang=zh")
}

// TestGetPortfolioBook_AllemandReplieSurAnglais : de/it ne sont pas curés — repli anglais, et
// jamais le français par défaut (règle établie pour tout le contenu maiProFiles).
func TestGetPortfolioBook_AllemandReplieSurAnglais(t *testing.T) {
	var appels []string
	_, err := serveurLivre(t, &appels).GetPortfolioBook(context.Background(), "de")

	require.NoError(t, err)
	require.Contains(t, appels[0], "lang=en")
	require.NotContains(t, appels[0], "lang=de")
}

func TestGetPortfolioPage_JointLesDeuxCouches(t *testing.T) {
	var appels []string
	page, err := serveurLivre(t, &appels).GetPortfolioPage(context.Background(), "coco_seo", "fr")

	require.NoError(t, err)
	require.Equal(t, "coco_seo", page.ID)
	require.Len(t, page.Paragraphs, 3)                     // le récit curé
	require.Contains(t, page.DescriptionPortfolio, "# Co") // la fiche technique, en markdown
	// Rendue en HTML côté serveur : le front n'a pas de moteur markdown, et goldmark est déjà
	// là pour le blog — même contrat que `content_html` des articles.
	require.Contains(t, page.DescriptionPortfolioHTML, "<h1")
	require.Contains(t, page.DescriptionPortfolioHTML, "<h2")
	require.Equal(t, "production", page.Status)
	require.Equal(t, "git.etheryale.com/x", page.Links["repo"])
	require.Len(t, page.Gallery, 1)
	// La section sert de légende dans la galerie : sans elle, une capture d'architecture et une
	// capture d'interface se ressemblent trop pour qu'un lecteur sache ce qu'il regarde.
	require.Equal(t, "screenshot", page.Gallery[0].LinkedTo.Section)
	require.Contains(t, appels[0], "lang=fr")
	// La démo vidéo traverse jusqu'au front : fichier, image d'attente et légende localisée.
	require.NotNil(t, page.Video)
	require.Equal(t, "/images/img_v", page.Video.URL)
	require.Equal(t, "/images/img_p", page.Video.PosterURL)
	require.Equal(t, "quatre agents", page.Video.Caption)
}

// TestGetPortfolioBook_SansVideo : un projet qui ne déclare pas de démo n'a pas de lecteur vide.
// `Video` reste nil, et le front n'affiche alors aucune section.
func TestGetPortfolioBook_SansVideo(t *testing.T) {
	var appels []string
	livre, err := serveurLivre(t, &appels).GetPortfolioBook(context.Background(), "en")

	require.NoError(t, err)
	require.Nil(t, livre.Projects[0].Video)
}

// TestGetPortfolioPage_ProjetHorsCuration : le portfolio est curé — une fiche qui n'y est pas
// doit remonter une erreur, pas une page vide qu'on afficherait quand même.
func TestGetPortfolioPage_ProjetHorsCuration(t *testing.T) {
	var appels []string
	_, err := serveurLivre(t, &appels).GetPortfolioPage(context.Background(), "veracite", "en")

	require.ErrorIs(t, err, ErrPortfolioAbsent,
		"le handler doit pouvoir distinguer 'hors curation' (404) de 'maiProFiles est tombé' (502)")
}
