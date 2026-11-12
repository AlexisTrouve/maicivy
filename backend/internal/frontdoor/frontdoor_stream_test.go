package frontdoor

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"maicivy/internal/middleware"
)

// QUOI : verrouille que le front-door relaie un SSE AU FIL DE L'EAU (chat, dossier vivant).
//
// POURQUOI : panne mesurée en prod le 07/11/2026 — proxy.DoTimeout de Fiber lit la réponse
// ENTIÈRE du backend avant de la recopier. Le backend diffusait (premier octet à 0,9 ms) mais le
// visiteur ne recevait rien avant la fin du run (≈ 45 s), et Cloudflare pouvait couper en 504.
//
// COMMENT : faux backend qui émet une trame, attend 1,5 s, puis une seconde ; front-door sur un
// vrai port ; le client doit lire la première trame bien avant la fin, puis le flux complet.
func TestFrontdoor_StreamsSSEWithoutBuffering(t *testing.T) {
	const pause = 1500 * time.Millisecond

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Upstream", "ok")
		_, _ = io.WriteString(w, "data: un\n\n")
		w.(http.Flusher).Flush()
		time.Sleep(pause)
		_, _ = io.WriteString(w, "data: deux\n\n")
	}))
	defer backend.Close()

	mr, _ := miniredis.Run()
	defer mr.Close()
	app := New(Config{
		Sus:         middleware.SusConfig{Redis: redis.NewClient(&redis.Options{Addr: mr.Addr()})},
		BackendURL:  backend.URL,
		FrontendURL: backend.URL,
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	// Réponse ORDINAIRE volumineuse (un PDF) : relayée intacte, corps de requête transmis, statut
	// et Content-Length conservés — le chemin non-SSE de proxyAPI ne doit rien changer.
	t.Run("réponse ordinaire", func(t *testing.T) {
		big := strings.Repeat("%PDF-0123456789", 30000) // ≈ 450 ko
		pdf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			in, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("X-Body-In", string(in))
			w.WriteHeader(201)
			_, _ = io.WriteString(w, big)
		}))
		defer pdf.Close()
		app2 := New(Config{
			Sus:        middleware.SusConfig{Redis: redis.NewClient(&redis.Options{Addr: mr.Addr()})},
			BackendURL: pdf.URL, FrontendURL: pdf.URL,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/dossier-vivant/run/pdf", strings.NewReader(`{"a":1}`))
		req.Header.Set("X-Real-IP", "1.2.3.5")
		resp, err := app2.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 201 || string(b) != big || resp.Header.Get("X-Body-In") != `{"a":1}` || resp.ContentLength != int64(len(big)) {
			t.Fatalf("réponse ordinaire altérée: status=%d len=%d/%d in=%q cl=%d", resp.StatusCode, len(b), len(big), resp.Header.Get("X-Body-In"), resp.ContentLength)
		}
	})

	for _, path := range []string{"/api/v1/dossier-vivant/run", "/api/v1/chat/stream"} {
		t.Run(path, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+path, strings.NewReader("{}"))
			req.Header.Set("X-Real-IP", "1.2.3.4")
			client := &http.Client{Transport: &http.Transport{DisableCompression: true}, Timeout: 10 * time.Second}

			start := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			buf := make([]byte, 1)
			if _, err := resp.Body.Read(buf); err != nil {
				t.Fatalf("premier octet: %v", err)
			}
			if first := time.Since(start); first > pause/2 {
				t.Fatalf("premier octet après %v (pause du flux %v) : le front-door tamponne le SSE", first, pause)
			}
			rest, _ := io.ReadAll(resp.Body)
			if got := string(buf) + string(rest); got != "data: un\n\ndata: deux\n\n" {
				t.Fatalf("flux relayé incomplet: %q", got)
			}
			if resp.StatusCode != 200 || resp.Header.Get("X-Upstream") != "ok" || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
				t.Fatalf("statut/en-têtes non relayés: %d %v", resp.StatusCode, resp.Header)
			}
		})
	}
}
