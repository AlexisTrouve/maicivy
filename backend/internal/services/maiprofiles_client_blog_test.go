// Test white-box (package services) : accès à baseURL/http non exportés pour pointer le client
// vers un httptest et capturer le payload réellement envoyé à maiProFiles.
//
// POURQUOI : verrouille l'incident 2026-06-29 — un PUT PARTIEL (cover seule) envoyait title/summary/
// content vides, que le merge superficiel de maiProFiles transformait en effacement du post.
package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"maicivy/internal/models"
)

// captureServer renvoie un serveur httptest qui décode le body de la requête dans dst et répond 200.
func captureServer(t *testing.T, dst *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, dst)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":34,"slug":"s","title":"t"}`)) // réponse minimale décodable
	}))
}

// Un update PARTIEL (cover seule) ne doit PAS envoyer title/summary/content : sinon "" écrase le post.
func TestUpdateBlogPost_PartialDoesNotWipeFields(t *testing.T) {
	var got map[string]any
	srv := captureServer(t, &got)
	defer srv.Close()

	c := &MaiProFilesClient{http: srv.Client(), baseURL: srv.URL}
	if _, err := c.UpdateBlogPost(context.Background(), 34, &models.BlogPost{
		CoverImageURL: "https://x/img.png",
	}); err != nil {
		t.Fatalf("UpdateBlogPost: %v", err)
	}

	for _, k := range []string{"title", "summary", "content", "content_html"} {
		if _, present := got[k]; present {
			t.Errorf("payload contient %q sur un update partiel — risque d'écrasement du post", k)
		}
	}
	if got["cover_image_url"] != "https://x/img.png" {
		t.Errorf("cover_image_url manquante/incorrecte: %v", got["cover_image_url"])
	}
}

// Quand le content EST fourni, on l'envoie AVEC content_html régénéré, sans toucher les autres champs.
func TestUpdateBlogPost_ContentAlsoSendsHTML(t *testing.T) {
	var got map[string]any
	srv := captureServer(t, &got)
	defer srv.Close()

	c := &MaiProFilesClient{http: srv.Client(), baseURL: srv.URL}
	if _, err := c.UpdateBlogPost(context.Background(), 1, &models.BlogPost{Content: "# Hello"}); err != nil {
		t.Fatalf("UpdateBlogPost: %v", err)
	}

	if got["content"] != "# Hello" {
		t.Errorf("content non envoyé: %v", got["content"])
	}
	if _, ok := got["content_html"]; !ok {
		t.Errorf("content_html non régénéré alors que content est fourni")
	}
	if _, ok := got["title"]; ok {
		t.Errorf("title envoyé alors qu'il était absent du update")
	}
}
