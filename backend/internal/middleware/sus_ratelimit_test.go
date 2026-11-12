package middleware

import (
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// --- Courbe LOGARITHMIQUE de throttle (sans aléa) ---
func TestSusThrottleP(t *testing.T) {
	assert.Equal(t, 0.0, susThrottleP(2.5))                       // sous le score "gratuit" → 0
	assert.Equal(t, 0.0, susThrottleP(susFreeScore))              // exactement free → 0
	assert.InDelta(t, susMaxP, susThrottleP(susFullScore), 0.001) // full → cap
	assert.InDelta(t, susMaxP, susThrottleP(500), 0.001)          // au-delà → reste capé
	// Point milieu (ratio log 0.5) : score = free * (full/free)^0.5 = 5*10^0.5 ≈ 15.81 → 0.45
	mid := susFreeScore * math.Sqrt(susFullScore/susFreeScore)
	assert.InDelta(t, susMaxP*0.5, susThrottleP(mid), 0.01)
	// Monotone croissant
	assert.Greater(t, susThrottleP(30), susThrottleP(10))
}

// --- Déclin exponentiel : après 1 demi-vie, le score est divisé par 2 ---
func TestSusReadScore_Decays(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	now := time.Now()
	halflifeSec := 3600.0 // 1h
	rdb.HSet(ctx, "maicivy:sus:x", "score", "40", "ts", now.Add(-time.Hour).Unix())

	got := susReadScore(ctx, rdb, "maicivy:sus:x", now, halflifeSec)
	assert.InDelta(t, 20.0, got, 0.1) // 40 → 20 après 1 demi-vie
}

func newSusApp(rdb *redis.Client) *fiber.App {
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(SusRateLimit(SusConfig{Redis: rdb}))
	app.Get("/miss", func(c *fiber.Ctx) error { return c.Status(404).SendString("nope") })
	app.Get("/ok", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func susHit(app *fiber.App, path, ip string) int {
	r := httptest.NewRequest("GET", path, nil)
	r.Header.Set("X-Real-IP", ip)
	resp, _ := app.Test(r)
	return resp.StatusCode
}

// Score bas (peu d'erreurs) → jamais throttlé.
func TestSusRateLimit_NoThrottleLowScore(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := newSusApp(rdb)
	for i := 0; i < 4; i++ { // 4 erreurs < free(5) → score reste sous le seuil → P=0
		assert.Equal(t, 404, susHit(app, "/miss", "1.1.1.1"))
	}
	assert.Equal(t, 200, susHit(app, "/ok", "1.1.1.1"))
}

// Chaque erreur scanner augmente le score (+1).
func TestSusRateLimit_BumpsScoreOnError(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := newSusApp(rdb)
	for i := 0; i < 3; i++ {
		susHit(app, "/miss", "2.2.2.2")
	}
	s, _ := rdb.HGet(context.Background(), "maicivy:sus:2.2.2.2", "score").Result()
	score, _ := strconv.ParseFloat(s, 64)
	assert.InDelta(t, 3.0, score, 0.01) // 3 erreurs → score ≈ 3
}

// benignPath : chemins navigateur (favicon, _next, robots…) exemptés ; le reste non.
func TestBenignPath(t *testing.T) {
	assert.True(t, benignPath("/favicon.ico"))
	assert.True(t, benignPath("/robots.txt"))
	assert.True(t, benignPath("/_next/static/chunks/x.js"))
	assert.True(t, benignPath("/.well-known/acme-challenge/abc"))
	assert.False(t, benignPath("/fr/cv"))
	assert.False(t, benignPath("/.env"))
	assert.False(t, benignPath("/api/v1/visitors/heartbeat"))
}

// FAUX POSITIF corrigé : 10 favicon 404 (navigateur normal) ne doivent PAS scorer ; un chemin
// scanner, lui, score toujours.
func TestSusRateLimit_BenignPathNoScore(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(SusRateLimit(SusConfig{Redis: rdb, ScannerPath: ScannerPathMatcher(AllScannerPatterns()...)}))
	app.Get("/*", func(c *fiber.Ctx) error { return c.SendStatus(404) }) // tout en 404

	for i := 0; i < 10; i++ { // 10 favicon 404 d'affilée
		assert.Equal(t, 404, susHit(app, "/favicon.ico", "9.9.9.9"))
	}
	s, _ := rdb.HGet(context.Background(), "maicivy:sus:9.9.9.9", "score").Result()
	assert.Empty(t, s, "favicon 404 ne doit JAMAIS scorer (clé sus inexistante)")

	// contrôle : un vrai chemin scanner (/.env) doit toujours scorer
	susHit(app, "/.env", "8.8.8.8")
	s2, _ := rdb.HGet(context.Background(), "maicivy:sus:8.8.8.8", "score").Result()
	assert.NotEmpty(t, s2, "un chemin scanner doit scorer")
}

// benignPath ne doit PAS exempter un chemin qui matche AUSSI une signature scanner (ex:
// /_next/credentials.json) — sinon un scanner préfixe /_next/ ou /.well-known/ pour scanner à
// score 0. On sert 200 partout : le SEUL moyen de scorer ici est le signal signature.
func TestSusRateLimit_BenignPrefixScannerStillScores(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(SusRateLimit(SusConfig{Redis: rdb, ScannerPath: ScannerPathMatcher(AllScannerPatterns()...)}))
	app.Get("/*", func(c *fiber.Ctx) error { return c.SendStatus(200) }) // 200 partout (aucun 4xx)

	// Préfixe bénin /_next/ MAIS signature scanner ("credential") → doit scorer.
	for i := 0; i < 3; i++ {
		susHit(app, "/_next/credentials.json", "4.4.4.4")
	}
	s, _ := rdb.HGet(context.Background(), "maicivy:sus:4.4.4.4", "score").Result()
	assert.NotEmpty(t, s, "/_next/<scanner> doit scorer (bypass benignPath corrigé)")

	// Non-régression : un VRAI asset _next bénin (non-scanner) ne doit JAMAIS scorer.
	susHit(app, "/_next/static/chunks/main-abc.js", "3.3.3.3")
	n, _ := rdb.Exists(context.Background(), "maicivy:sus:3.3.3.3").Result()
	assert.Equal(t, int64(0), n, "un vrai asset _next ne doit pas scorer (pas de régression favicon)")
}

// isAssetExtension : reconnaît les extensions d'asset (hors .json), rejette le reste.
func TestIsAssetExtension(t *testing.T) {
	assert.True(t, isAssetExtension("/logo.png"))
	assert.True(t, isAssetExtension("/assets/app.css"))
	assert.True(t, isAssetExtension("/fonts/Inter.woff2"))
	assert.True(t, isAssetExtension("/x.JS"))         // casse insensible
	assert.False(t, isAssetExtension("/config.json")) // .json EXCLU (probe scanner)
	assert.False(t, isAssetExtension("/.env"))
	assert.False(t, isAssetExtension("/api/v1/letters"))
	assert.False(t, isAssetExtension("/wp-login.php"))
	assert.False(t, isAssetExtension("/a.b/c")) // point dans un segment, pas une extension finale
	assert.False(t, isAssetExtension("/admin"))
}

// FOOTGUN "asset manquant = throttle" : un asset (favicon/logo/css/font) qui 404 EN RAFALE ne doit
// JAMAIS scorer un vrai user. Un chemin scanner score toujours (contrôle).
func TestSusRateLimit_StaticAssetNoScore(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(SusRateLimit(SusConfig{Redis: rdb, ScannerPath: ScannerPathMatcher(AllScannerPatterns()...)}))
	app.Get("/*", func(c *fiber.Ctx) error { return c.SendStatus(404) }) // tout en 404

	for _, asset := range []string{"/logo.png", "/style.css", "/fonts/Inter.woff2", "/app.js"} {
		for i := 0; i < 10; i++ {
			susHit(app, asset, "1.2.3.4")
		}
	}
	n, _ := rdb.Exists(context.Background(), "maicivy:sus:1.2.3.4").Result()
	assert.Equal(t, int64(0), n, "40 assets 404 ne doivent JAMAIS scorer (footgun favicon/asset)")

	// Contrôle : un vrai chemin scanner doit toujours scorer.
	susHit(app, "/.env", "5.6.7.8")
	s, _ := rdb.HGet(context.Background(), "maicivy:sus:5.6.7.8", "score").Result()
	assert.NotEmpty(t, s, "un chemin scanner doit toujours scorer")
}

// Override : un chemin d'extension asset qui matche AUSSI une signature scanner DOIT scorer (sinon
// un scanner suffixe .js pour passer à score 0).
func TestSusRateLimit_AssetExtScannerStillScores(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(SusRateLimit(SusConfig{Redis: rdb, ScannerPath: ScannerPathMatcher(`\.env`)}))
	app.Get("/*", func(c *fiber.Ctx) error { return c.SendStatus(200) }) // 200 partout

	for i := 0; i < 3; i++ {
		susHit(app, "/config/.env.js", "9.8.7.6") // extension .js MAIS signature .env dans le path
	}
	s, _ := rdb.HGet(context.Background(), "maicivy:sus:9.8.7.6", "score").Result()
	assert.NotEmpty(t, s, "asset-ext + signature scanner doit scorer (override)")
}

// Crédit de densité : un vrai 2xx abaisse un score existant (plancher 0) ; une IP propre ne crée
// aucune clé sur un 200 (pas de write inutile).
func TestSusRateLimit_SuccessCredit(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	rdb.HSet(ctx, "maicivy:sus:3.3.3.3", "score", "4.0", "ts", time.Now().Unix())

	app := newSusApp(rdb) // /ok → 200, /miss → 404
	for i := 0; i < 4; i++ {
		assert.Equal(t, 200, susHit(app, "/ok", "3.3.3.3")) // 4 succès × 0.5 = -2.0
	}
	s, _ := rdb.HGet(ctx, "maicivy:sus:3.3.3.3", "score").Result()
	score, _ := strconv.ParseFloat(s, 64)
	assert.InDelta(t, 2.0, score, 0.05, "4 succès (×0.5) doivent retrancher 2.0")

	for i := 0; i < 20; i++ { // plancher 0 : jamais négatif
		susHit(app, "/ok", "3.3.3.3")
	}
	s2, _ := rdb.HGet(ctx, "maicivy:sus:3.3.3.3", "score").Result()
	score2, _ := strconv.ParseFloat(s2, 64)
	assert.InDelta(t, 0.0, score2, 0.0001, "le score doit être plancher à 0")

	// IP propre (score absent) → un 200 ne crée AUCUNE clé.
	susHit(app, "/ok", "7.7.7.7")
	n, _ := rdb.Exists(ctx, "maicivy:sus:7.7.7.7").Result()
	assert.Equal(t, int64(0), n, "un 200 sur IP propre ne doit créer aucune clé (pas de write)")
}

// Score élevé pré-chargé → la plupart des requêtes throttlées (P≈0.9, test statistique).
func TestSusRateLimit_HighScoreThrottles(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	rdb.HSet(ctx, "maicivy:sus:9.9.9.9", "score", strconv.FormatFloat(susFullScore, 'f', 1, 64), "ts", time.Now().Unix())

	app := newSusApp(rdb)
	throttled := 0
	for i := 0; i < 100; i++ {
		if susHit(app, "/ok", "9.9.9.9") == 429 {
			throttled++
		}
	}
	assert.Greater(t, throttled, 50) // P≈0.9 → bien plus de 50/100 (jamais 100% : cap 0.9)
	assert.Less(t, throttled, 100)   // jamais tout (cap 0.9)
}

// Signal "signature de path" : un chemin scanner en 200 bumpe le score (malgré le 200) ; un
// path légitime en 200 ne bumpe pas.
func TestSusRateLimit_ScannerPathSignal(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(SusRateLimit(SusConfig{Redis: rdb, ScannerPath: ScannerPathMatcher(`\.env`, `api-key`)}))
	app.Get("/*", func(c *fiber.Ctx) error { return c.SendStatus(200) }) // 200 partout

	// Path scanner en 200 → bumpe via le signal signature.
	for i := 0; i < 3; i++ {
		susHit(app, "/app/.env", "5.5.5.5")
	}
	s, err := rdb.HGet(context.Background(), "maicivy:sus:5.5.5.5", "score").Result()
	if err != nil || s == "" {
		t.Fatal("le path scanner aurait dû construire un score")
	}
	score, _ := strconv.ParseFloat(s, 64)
	assert.Greater(t, score, 0.0)

	// Path légitime en 200 → aucun score (clé absente).
	susHit(app, "/fr/cv", "6.6.6.6")
	n, _ := rdb.Exists(context.Background(), "maicivy:sus:6.6.6.6").Result()
	assert.Equal(t, int64(0), n, "un path légitime ne doit pas construire de score")
}

// Le helper allowlist : IP nue → /32 ou /128, CIDR, entrées invalides ignorées.
func TestIPAllowed(t *testing.T) {
	nets := parseAllowlist([]string{"10.0.0.0/8", "1.2.3.4", "2001:db8::/32", "garbage", ""})
	assert.True(t, ipAllowed("10.5.6.7", nets))
	assert.True(t, ipAllowed("1.2.3.4", nets))
	assert.False(t, ipAllowed("11.0.0.1", nets))
	assert.True(t, ipAllowed("2001:db8::1", nets))
	assert.False(t, ipAllowed("8.8.8.8", nets))
	assert.False(t, ipAllowed("1.2.3.4", parseAllowlist(nil))) // allowlist vide → rien d'exempté
}

// Exemptions : owner (header) et allowlist IP bypassent le sus (ni score ni throttle).
func TestSusRateLimit_Bypass(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(SusRateLimit(SusConfig{Redis: rdb, OwnerKey: "secret-owner", Allowlist: []string{"10.0.0.0/8", "1.2.3.4"}}))
	app.Get("/*", func(c *fiber.Ctx) error { return c.Status(404).SendString("x") }) // 404 → bumperait

	// Owner-key → exempté (header), même en floodant des 404.
	for i := 0; i < 20; i++ {
		r := httptest.NewRequest("GET", "/miss", nil)
		r.Header.Set("X-Real-IP", "55.55.55.55")
		r.Header.Set("X-Owner-Key", "secret-owner")
		resp, _ := app.Test(r)
		assert.Equal(t, 404, resp.StatusCode) // jamais 429
	}
	n, _ := rdb.Exists(context.Background(), "maicivy:sus:55.55.55.55").Result()
	assert.Equal(t, int64(0), n, "owner-key ne doit construire aucun score")

	// IP allowlistée (∈ 10/8) → exemptée même en floodant.
	for i := 0; i < 20; i++ {
		assert.Equal(t, 404, susHit(app, "/miss", "10.1.2.3"))
	}
	n, _ = rdb.Exists(context.Background(), "maicivy:sus:10.1.2.3").Result()
	assert.Equal(t, int64(0), n, "IP allowlistée ne doit construire aucun score")

	// Contrôle : IP non allowlistée, sans key → bien scorée.
	for i := 0; i < 8; i++ {
		susHit(app, "/miss", "77.77.77.77")
	}
	s, _ := rdb.HGet(context.Background(), "maicivy:sus:77.77.77.77", "score").Result()
	assert.NotEmpty(t, s, "une IP normale doit être scorée (contrôle)")
}

// Badge dev "je suis un copain" : un cookie maicivy_friend HMAC-valide exempte du sus (ni score ni
// throttle), même en floodant des 404 ; un cookie forgé/absent ne change rien (scoré normalement).
// IP-indépendant → résout le throttle du dev sans allowlist d'IP (l'IP peut tourner).
func TestSusRateLimit_FriendCookieBypass(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	secret := "test-session-secret"
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(SusRateLimit(SusConfig{Redis: rdb, FriendSecret: secret}))
	app.Get("/*", func(c *fiber.Ctx) error { return c.Status(404).SendString("x") }) // 404 → bumperait

	validBadge := SignAdminCookie(secret, time.Hour) // jeton signé par le serveur

	// Cookie ami VALIDE → exempté même en floodant des 404.
	for i := 0; i < 20; i++ {
		r := httptest.NewRequest("GET", "/miss", nil)
		r.Header.Set("X-Real-IP", "55.55.55.55")
		r.AddCookie(&http.Cookie{Name: "maicivy_friend", Value: validBadge})
		resp, _ := app.Test(r)
		assert.Equal(t, 404, resp.StatusCode) // jamais 429
	}
	n, _ := rdb.Exists(context.Background(), "maicivy:sus:55.55.55.55").Result()
	assert.Equal(t, int64(0), n, "badge ami valide ne doit construire aucun score")

	// Cookie FORGÉ (mauvaise signature) → ignoré, scoré normalement.
	for i := 0; i < 8; i++ {
		r := httptest.NewRequest("GET", "/miss", nil)
		r.Header.Set("X-Real-IP", "66.66.66.66")
		r.AddCookie(&http.Cookie{Name: "maicivy_friend", Value: "admin:9999999999.forged"})
		app.Test(r)
	}
	s, _ := rdb.HGet(context.Background(), "maicivy:sus:66.66.66.66", "score").Result()
	assert.NotEmpty(t, s, "cookie forgé ne doit PAS exempter (scoré normalement)")

	// Aucun cookie + FriendSecret vide → pas d'exemption (contrôle de non-régression).
	appNoSecret := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	appNoSecret.Use(SusRateLimit(SusConfig{Redis: rdb})) // FriendSecret == ""
	appNoSecret.Get("/*", func(c *fiber.Ctx) error { return c.Status(404).SendString("x") })
	for i := 0; i < 8; i++ {
		r := httptest.NewRequest("GET", "/miss", nil)
		r.Header.Set("X-Real-IP", "44.44.44.44")
		r.AddCookie(&http.Cookie{Name: "maicivy_friend", Value: validBadge})
		appNoSecret.Test(r)
	}
	s2, _ := rdb.HGet(context.Background(), "maicivy:sus:44.44.44.44", "score").Result()
	assert.NotEmpty(t, s2, "sans FriendSecret configuré, le cookie ne doit pas exempter")
}

// Helper IP version + bloc à bannir.
func TestIPVersionAndBlock(t *testing.T) {
	v4, b4 := ipVersionAndBlock("203.0.113.10")
	assert.Equal(t, "IPv4", v4)
	assert.Equal(t, "203.0.113.10/32", b4)

	v6, b6 := ipVersionAndBlock("2001:db8:2005:100::bef")
	assert.Equal(t, "IPv6", v6)
	assert.Equal(t, "2001:db8:2005:100::/64", b6)
}

// susKeyIP : IPv4 → IP complète ; IPv6 → préfixe /64 ; non parsable → tel quel.
func TestSusKeyIP(t *testing.T) {
	assert.Equal(t, "203.0.113.10", susKeyIP("203.0.113.10"))                        // IPv4 = complète
	assert.Equal(t, "2001:db8:abcd:1::", susKeyIP("2001:db8:abcd:1::dead"))          // IPv6 → /64
	assert.Equal(t, "2001:db8:abcd:1::", susKeyIP("2001:db8:abcd:1:ffff:ffff:ff:9")) // autre /128, même /64
	assert.Equal(t, "garbage", susKeyIP("garbage"))                                  // non parsable
}

// #3 durcissement : plusieurs /128 d'un MÊME /64 IPv6 doivent cumuler leur score sur UNE clé /64
// (anti-bloat Redis + anti-évasion par rotation d'adresse), pas créer une clé par /128.
func TestSusRateLimit_IPv6AggregatedBy64(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := newSusApp(rdb)

	// 4 adresses IPv6 différentes dans le même /64 → chacune un 404 (+1).
	ips := []string{
		"2001:db8:abcd:1::1", "2001:db8:abcd:1::2",
		"2001:db8:abcd:1:ffff::9", "2001:db8:abcd:1::dead",
	}
	for _, ip := range ips {
		assert.Equal(t, 404, susHit(app, "/miss", ip))
	}

	// La clé agrégée /64 porte le score cumulé.
	s, _ := rdb.HGet(context.Background(), "maicivy:sus:2001:db8:abcd:1::", "score").Result()
	score, _ := strconv.ParseFloat(s, 64)
	assert.InDelta(t, 4.0, score, 0.01, "les /128 d'un même /64 doivent cumuler sur une clé /64")

	// Aucune clé par /128 (vérif anti-bloat).
	n, _ := rdb.Exists(context.Background(), "maicivy:sus:2001:db8:abcd:1::1").Result()
	assert.Equal(t, int64(0), n, "pas de clé sus par /128 (sinon bloat Redis)")
}

// Alerte filou : webhook reçu (1 fois) + de-dup, avec version IP.
func TestSusAlert_WebhookAndDedup(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	received := make(chan []byte, 4)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		w.WriteHeader(204)
	}))
	defer ts.Close()

	cfg := SusConfig{Redis: rdb, AlertScore: 20, WebhookURL: ts.URL}
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Get("/trigger", func(c *fiber.Ctx) error {
		maybeAlertSusIP(cfg, c, "6.6.6.6", 25.0)
		return c.SendString("ok")
	})

	app.Test(httptest.NewRequest("GET", "/trigger", nil)) // 1er → webhook
	app.Test(httptest.NewRequest("GET", "/trigger", nil)) // 2e → dédupliqué

	select {
	case body := <-received:
		assert.Contains(t, string(body), "Filou")
		assert.Contains(t, string(body), "6.6.6.6")
		assert.Contains(t, string(body), "IPv4")
	case <-time.After(2 * time.Second):
		t.Fatal("webhook jamais reçu")
	}
	select {
	case <-received:
		t.Fatal("webhook envoyé 2x — de-dup cassée")
	case <-time.After(300 * time.Millisecond):
	}
}

// classifyAttack : catégorie strictement majoritaire des chemins ; ".php" prime sur "wp-".
func TestClassifyAttack(t *testing.T) {
	assert.Equal(t, "php", classifyAttack([]string{"/wefile.php", "/ops.php", "/201.php"}))
	assert.Equal(t, "php", classifyAttack([]string{"/wp-login.php", "/x.php"}))
	assert.Equal(t, "wordpress", classifyAttack([]string{"/wp-admin/", "/wp-content/x"}))
	assert.Equal(t, "env", classifyAttack([]string{"/.env", "/app/.env.backup", "/.env.prod"}))
	assert.Equal(t, "config", classifyAttack([]string{"/config/app.rb", "/settings.ini", "/mailer.yaml"}))
	assert.Equal(t, "mixed", classifyAttack([]string{"/a.php", "/.env", "/config/x", "/random"}))
	assert.Equal(t, "unknown", classifyAttack(nil))
}

// susTypeMuted : appartenance à la liste mutée, insensible casse/espaces.
func TestSusTypeMuted(t *testing.T) {
	assert.True(t, susTypeMuted("php", []string{"php"}))
	assert.True(t, susTypeMuted("php", []string{" PHP "}))
	assert.False(t, susTypeMuted("config", []string{"php"}))
	assert.False(t, susTypeMuted("php", nil))
}

// Mute par type : un scan php (muté) ne notifie PAS ; un scan secrets (non muté) de la MÊME IP
// notifie quand même → le dédup PAR IP+TYPE ne masque pas un vrai scan derrière un php.
func TestSusAlert_MuteByType(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	received := make(chan []byte, 4)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		w.WriteHeader(204)
	}))
	defer ts.Close()

	cfg := SusConfig{Redis: rdb, AlertScore: 20, WebhookURL: ts.URL, AlertMuteTypes: []string{"php"}}
	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Get("/trigger", func(c *fiber.Ctx) error {
		maybeAlertSusIP(cfg, c, "7.7.7.7", 25.0)
		return c.SendString("ok")
	})
	ctx := context.Background()

	// 1. Scan PHP (muté) → AUCUN webhook.
	rdb.RPush(ctx, "maicivy:sus:paths:7.7.7.7", "/wefile.php", "/ops.php", "/201.php")
	app.Test(httptest.NewRequest("GET", "/trigger", nil))
	select {
	case <-received:
		t.Fatal("webhook envoyé pour un type muté (php)")
	case <-time.After(300 * time.Millisecond):
	}

	// 2. Même IP, scan secrets/config (non muté) → webhook attendu (pas masqué par le dédup php).
	rdb.Del(ctx, "maicivy:sus:paths:7.7.7.7")
	rdb.RPush(ctx, "maicivy:sus:paths:7.7.7.7", "/config/application.rb", "/app/config/email.yaml", "/settings.ini")
	app.Test(httptest.NewRequest("GET", "/trigger", nil))
	select {
	case body := <-received:
		assert.Contains(t, string(body), "7.7.7.7")
		assert.Contains(t, string(body), "config")
	case <-time.After(2 * time.Second):
		t.Fatal("webhook secrets jamais reçu — type non-muté doit notifier")
	}
}
