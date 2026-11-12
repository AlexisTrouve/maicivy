package middleware

import (
	"bufio"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
)

// QUOI : verrouille qu'une réponse en FLUX (SSE : chat, dossier vivant) traverse la chaîne de
// middlewares globale de cmd/main.go SANS être retenue jusqu'à la fin.
//
// POURQUOI : panne mesurée en prod le 07/11/2026 — le Logger lisait len(c.Response().Body()).
// Sur une réponse à SetBodyStreamWriter, fasthttp DRAINE alors tout le flux en mémoire pour
// rendre ce corps : le premier octet du run du dossier vivant arrivait après 56 s (fin du run),
// et Cloudflare répondait 504. Indice dans le journal : duration_ms=0.3 pour un run de 64 s.
//
// COMMENT : vrai serveur Fiber sur un port local, même ordre que main.go (RequestID → Logger →
// compress). Le handler émet une trame, attend 1,5 s, puis une seconde. Le client doit lire la
// première trame bien avant la fin. Deux cas : sans compression, et avec Accept-Encoding gzip
// (ce qu'envoie un navigateur).
func TestStreamingNotBufferedByGlobalMiddlewares(t *testing.T) {
	const pause = 1500 * time.Millisecond

	app := fiber.New()
	app.Use(RequestID())
	app.Use(Logger())
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
	app.Get("/sse", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
			_, _ = w.WriteString("data: un\n\n")
			_ = w.Flush()
			time.Sleep(pause)
			_, _ = w.WriteString("data: deux\n\n")
			_ = w.Flush()
		})
		return nil
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	defer func() { _ = app.Shutdown() }()

	for _, enc := range []string{"identity", "gzip"} {
		t.Run(enc, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "http://"+ln.Addr().String()+"/sse", nil)
			req.Header.Set("Accept-Encoding", enc)
			// Transport brut : pas de décompression transparente qui pourrait elle-même tamponner.
			client := &http.Client{Transport: &http.Transport{DisableCompression: true}, Timeout: 10 * time.Second}

			start := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			buf := make([]byte, 1)
			if _, err := resp.Body.Read(buf); err != nil {
				t.Fatalf("lecture du premier octet: %v", err)
			}
			if first := time.Since(start); first > pause/2 {
				t.Fatalf("premier octet après %v (pause du flux %v) : la réponse est tamponnée jusqu'à la fin", first, pause)
			}
		})
	}
}
