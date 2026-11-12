// Package frontdoor construit le reverse-proxy "front-door" : le checkpoint anti-abus placé
// DEVANT tout le trafic.
//
// QUOI    : une app Fiber qui (1) applique middleware.SusRateLimit (signature de path incluse)
//
//	sur CHAQUE requête, puis (2) proxy /api/* → backend, tout le reste → frontend.
//
// POURQUOI: le sus-rate-limit vivant dans le backend ne voyait que /api (5% du scan). Ici il est
//
//	au point de convergence → il voit TOUT (le scan du frontend aussi). C'est le
//	"choke-point" identifié par le tuning ③ comme le levier #1.
//
// COMMENT : New(cfg) assemble l'app (testable en isolation avec de faux upstreams). Le proxy
//
//	réémet la requête telle quelle (proxy.Do) et recopie la réponse — le middleware
//	observe le vrai status renvoyé par backend/frontend pour scorer. Fail-open hérité du
//	middleware (Redis down → on ne bloque pas). /ws/ n'arrive jamais ici (bypass nginx).
package frontdoor

import (
	"bufio"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/valyala/fasthttp"

	"maicivy/internal/middleware"
)

// Config paramètre le front-door.
type Config struct {
	Sus         middleware.SusConfig // checkpoint anti-abus (avec ScannerPath activé en prod)
	BackendURL  string               // ex: http://maicivy-backend:8080
	FrontendURL string               // ex: http://maicivy-frontend:3000
	BodyLimit   int                  // octets max du corps (0 = défaut Fiber) ; CV/offres volumineux
	Timeout     time.Duration        // timeout du proxy (0 = défaut 120s) ; génération CV synchrone longue
}

// New assemble l'app front-door (sans la démarrer — testable).
func New(cfg Config) *fiber.App {
	fcfg := fiber.Config{
		ProxyHeader:           "X-Real-IP", // l'IP réelle vient de nginx (cf. cloudflare-realip)
		DisableStartupMessage: true,
	}
	if cfg.BodyLimit > 0 {
		fcfg.BodyLimit = cfg.BodyLimit
	}
	app := fiber.New(fcfg)

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second // couvre la génération CV/IA synchrone
	}

	// Healthcheck local du front-door (non proxifié) — enregistré avant le checkpoint.
	app.Get("/__frontdoor/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// 1. Checkpoint anti-abus sur TOUT le trafic.
	app.Use(middleware.SusRateLimit(cfg.Sus))

	// 2. Routage : /api/* → backend, tout le reste → frontend. proxy.Do réémet la requête
	//    (méthode, headers dont X-Real-IP, corps) et recopie la réponse + son status.
	//    /api passe par proxyAPI, qui sait relayer un SSE au fil de l'eau (cf. son commentaire).
	api := newStreamingClient(timeout)
	front := newFrontendClient()
	app.All("/*", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api") {
			return proxyAPI(c, api, cfg.BackendURL+c.OriginalURL())
		}
		return proxy.DoTimeout(c, cfg.FrontendURL+c.OriginalURL(), timeout, front)
	})

	return app
}

// upstreamMaxIdle — durée au-delà de laquelle le front-door abandonne une connexion keep-alive
// inactive vers un upstream.
//
// POURQUOI : elle doit rester SOUS le délai d'inactivité du serveur d'en face. Node (Next
// standalone) ferme une connexion inactive après 5 s ; fasthttp la garde 10 s par défaut. Dans
// l'intervalle, le front-door réutilisait une connexion déjà fermée : fasthttp rejoue un GET au
// plus 5 fois, et une rafale d'images après une pause en tirait davantage du pool → 500
// « the server closed connection before returning the first response byte » (journal nginx du
// 11/11 : /favicon.ico, /_next/image). 2 s laisse de la marge au nettoyeur de fasthttp, qui ne
// passe que périodiquement. Verrouillé par TestFrontdoor_FrontKeepAliveFermeParLeServeur.
const upstreamMaxIdle = 2 * time.Second

// newFrontendClient — client fasthttp du proxy vers Next. Celui par défaut du middleware proxy
// de Fiber garde ses connexions 10 s : voir upstreamMaxIdle.
func newFrontendClient() *fasthttp.Client {
	return &fasthttp.Client{
		MaxIdleConnDuration:      upstreamMaxIdle,
		NoDefaultUserAgentHeader: true, // mêmes réglages que le client par défaut du middleware
		DisablePathNormalizing:   true,
	}
}

// newStreamingClient — client fasthttp qui rend la main dès les EN-TÊTES de la réponse
// (StreamResponseBody) au lieu d'attendre le corps entier.
//
// ReadTimeout : fasthttp pose l'échéance de lecture UNE fois avant la réponse ; elle reste en
// vigueur pendant la lecture du flux, donc elle borne la réponse entière — comme le DoTimeout
// d'avant (120 s, aligné sur le WriteTimeout du backend).
func newStreamingClient(timeout time.Duration) *fasthttp.Client {
	return &fasthttp.Client{
		MaxIdleConnDuration:      upstreamMaxIdle,
		StreamResponseBody:       true,
		ReadTimeout:              timeout,
		WriteTimeout:             timeout,
		NoDefaultUserAgentHeader: true,
		DisablePathNormalizing:   true,
	}
}

// proxyAPI — relaie une requête /api vers le backend.
//
// POURQUOI pas proxy.DoTimeout : il lit la réponse ENTIÈRE avant de la recopier. Mesuré en prod
// le 07/11/2026 : le backend diffusait le run du dossier vivant dès 0,9 ms, le visiteur ne
// recevait rien avant la fin (≈ 45 s) — le chat SSE était tamponné de la même façon.
//
// COMMENT : 1. recopier la requête (méthode, en-têtes dont X-Real-IP, corps) vers l'URL backend ;
// 2. une fois les en-têtes reçus, les recopier (statut compris) ; 3. réponse `text/event-stream`
// → relayer le flux morceau par morceau, flush à chaque lecture, la réponse fasthttp n'est rendue
// au pool qu'à la fin du flux ; toute autre réponse → corps lu en entier, comportement d'avant
// (Content-Length conservé). Le statut est posé AVANT le retour : le checkpoint anti-abus, qui le
// lit après c.Next(), voit le vrai statut.
func proxyAPI(c *fiber.Ctx, client *fasthttp.Client, url string) error {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	c.Request().CopyTo(req)
	req.SetRequestURI(url)
	req.Header.Del(fiber.HeaderConnection)

	err := client.Do(req, resp)
	fasthttp.ReleaseRequest(req)
	if err != nil {
		fasthttp.ReleaseResponse(resp)
		return err
	}

	resp.Header.Del(fiber.HeaderConnection)
	stream := resp.BodyStream()
	if stream == nil || !strings.HasPrefix(string(resp.Header.ContentType()), "text/event-stream") {
		// Réponse ordinaire : Body() draine le flux éventuel ; CopyTo duplique en-têtes + corps.
		resp.Body()
		resp.CopyTo(c.Response())
		_ = resp.CloseBodyStream()
		fasthttp.ReleaseResponse(resp)
		return nil
	}

	resp.Header.CopyTo(&c.Response().Header)
	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer func() {
			_ = resp.CloseBodyStream()
			fasthttp.ReleaseResponse(resp)
		}()
		buf := make([]byte, 4096)
		for {
			n, rerr := stream.Read(buf)
			if n > 0 {
				if _, werr := w.Write(buf[:n]); werr != nil {
					return // visiteur parti : on ferme le flux amont, le backend annule son run
				}
				if werr := w.Flush(); werr != nil {
					return
				}
			}
			if rerr != nil {
				return // io.EOF = fin normale du flux
			}
		}
	})
	return nil
}
