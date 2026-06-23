package middleware

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	// susRetryAfter : un 429 dit "réessaie dans quelques secondes" — JAMAIS un blocage long.
	// La vraie pression vient de la PROBABILITÉ, qui persiste tant que le score reste haut (jours).
	susRetryAfter = 5

	// Courbe LOGARITHMIQUE de throttle (score → proba). Croissance log (pas linéaire/quadratique).
	susFreeScore = 5.0  // erreurs "gratuites" avant tout throttle (une page cassée ne pénalise pas)
	susFullScore = 50.0 // score où le throttle sature
	susMaxP      = 0.9  // proba max — jamais 100% (jamais de coupure dure)

	// susSuccessCredit : décrément du score par vrai 2xx (densité d'erreurs, cf. la branche succès).
	// Conservateur (0.5) : protège les vrais users sans annuler le signal d'un scanner qui pad des 200.
	susSuccessCredit = 0.5

	// friendCookieName : cookie "badge dev" exemptant le sus (cf. SusConfig.FriendSecret). Même nom
	// que celui posé au login owner (api/admin.go).
	friendCookieName = "maicivy_friend"
)

// SusConfig configure le sus-rate-limit persistant + l'alerte filou.
type SusConfig struct {
	Redis         *redis.Client
	HalflifeHours float64 // demi-vie du score (défaut 48h = 2 j ; mémoire utile ~4-5 j)
	AlertScore    float64 // score à partir duquel on alerte filou (défaut 20)
	WebhookURL    string  // webhook Discord (vide = log seul)
	// ScannerPath : signal "signature de path" optionnel (cf. scanner_signatures.go). Si défini,
	// une requête dont le path matche compte comme erreur-scanner MÊME en 200 → rattrape le scan
	// du frontend (qui renvoie 200 à tout). nil = signal désactivé (défaut prod).
	ScannerPath func(string) bool
	// Now : horloge injectable (défaut time.Now). Permet au replay de SIMULER le temps écoulé
	// entre requêtes → le déclin s'applique correctement sur des traces multi-jours. nil = time.Now.
	Now func() time.Time
	// OwnerKey : si non vide, une requête avec le header X-Owner-Key == OwnerKey est EXEMPTÉE (ni
	// score ni throttle). Indépendant de l'IP → marche malgré une IP dynamique (appels owner/S2S).
	OwnerKey string
	// Allowlist : IPs/CIDR exemptés (IPs STABLES : monitor, partenaire, box de test). Parsé au setup.
	Allowlist []string
	// FriendSecret : si non vide, une requête portant un cookie `maicivy_friend` HMAC-valide (signé
	// avec ce secret = le SESSION_SECRET serveur) est EXEMPTÉE (ni score ni throttle). Badge dev
	// "je suis un copain" : posé au login owner, valable ~90 j, IP-INDÉPENDANT (survit au changement
	// d'IP, contrairement à l'allowlist) et révocable (tourner le secret invalide tous les badges).
	FriendSecret string
	// AlertMuteTypes : types d'attaque (cf. classifyAttack) dont l'alerte filou est LOGguée mais
	// PAS notifiée sur Discord. Ex "php" : les scans webshell PHP sont du bruit (aucun PHP servi
	// sur maicivy) → on garde la trace en log mais on ne spamme pas la notif. CSV via env.
	AlertMuteTypes []string
}

// SusRateLimit : throttle PROGRESSIF et PERSISTANT par IP, basé sur un score de réputation qui
// monte avec la DENSITÉ d'erreurs et DÉCLINE exponentiellement (demi-vie ~2 j, mémoire ~4-5 j).
//
// QUOI    : score par IP = erreurs "scanner" (404/403/400/405) accumulées, en déclin exponentiel.
//
//	La proba de 429 monte LOGARITHMIQUEMENT avec le score (vite au début, plateau ensuite ;
//	cap 90%). Un 429 = "réessaie dans 5s", jamais un blocage long — mais la pression
//	PERSISTE des jours tant que le score reste haut (scanner dense). Si l'IP arrête les
//	erreurs, le score décline → le throttle se réduit tout seul.
//
// POURQUOI: une IP ≠ une personne → jamais de coupure dure. Mémoire longue pour les récidivistes
//
//	denses, oubli progressif (pas en 30s). Croissance log → reste gentil même très haut.
//
// COMMENT : hash Redis `maicivy:sus:<ip>` {score, ts}. À la lecture on applique le déclin depuis ts.
//
//	Throttle = susMaxP × clamp(ln(score/free)/ln(full/free), 0, 1). Sur requête scanner :
//	score = déclin(score) + 1, réécriture {score, now}, TTL ~3 demi-vies. Alerte filou si
//	score >= AlertScore (log + webhook, dé-dupliquée). On ne touche au score QUE sur erreur
//	(les requêtes OK ne coûtent rien). Fail-open si Redis indisponible.
func SusRateLimit(cfg SusConfig) fiber.Handler {
	halflifeHours := cfg.HalflifeHours
	if halflifeHours <= 0 {
		halflifeHours = 48
	}
	halflifeSec := halflifeHours * 3600
	memoryTTL := time.Duration(halflifeSec*3) * time.Second // après ~3 demi-vies, score négligeable
	alertScore := cfg.AlertScore
	if alertScore <= 0 {
		alertScore = 20
	}
	allowNets := parseAllowlist(cfg.Allowlist) // parsé une fois au setup

	return func(c *fiber.Ctx) error {
		ip := c.IP()
		if ip == "" {
			return c.Next()
		}
		// Exemptions (ni score ni throttle) : owner via header (IP-indépendant, résiste à l'IP
		// dynamique) ou IP allowlistée (tiers stables). L'exception ne touche jamais le score.
		if cfg.OwnerKey != "" && c.Get("X-Owner-Key") == cfg.OwnerKey {
			return c.Next()
		}
		// Badge dev "je suis un copain" : cookie maicivy_friend HMAC-valide → exempté. IP-indépendant
		// (le owner dev génère des 404 légitimes en bossant ; ce cookie le marque ami sans allowlister
		// son IP, qui peut tourner). VerifyAdminCookie vérifie signature + expiry (constant-time).
		if cfg.FriendSecret != "" && VerifyAdminCookie(c.Cookies(friendCookieName), cfg.FriendSecret) {
			return c.Next()
		}
		if ipAllowed(ip, allowNets) {
			return c.Next()
		}
		// Chemins de navigateur bénins (favicon, _next, robots…) ET assets statiques (.png/.css/.js/
		// fontes…) : JAMAIS scorés ni throttlés.
		// POURQUOI : un asset manquant → 404 ; le navigateur le redemande à CHAQUE page → le score sus
		// d'un VRAI utilisateur grimpait jusqu'au throttle (faux positif). Un favicon précis était déjà
		// couvert, mais N'IMPORTE quel asset cassé (/logo.png, un .css externe, une font) faisait le
		// même dégât. Un scanner ne cible pas /logo.png (il cible /.env, /wp-login.php… → couverts par
		// les signatures), donc exempter les extensions d'asset ne perd aucun signal.
		// EXCEPTION : on n'exempte QUE si le chemin ne matche pas aussi une signature scanner — sinon
		// un scanner suffixait .js / préfixait /_next/ pour scanner à score 0 (ex: /wp-config.js reste
		// scoré). Un vrai asset (/logo.png, /_next/static/x.js) ne matche jamais une signature.
		if (benignPath(c.Path()) || isAssetExtension(c.Path())) && !(cfg.ScannerPath != nil && cfg.ScannerPath(c.Path())) {
			return c.Next()
		}
		ctx := context.Background()
		// Identité d'agrégation des clés sus : /64 en IPv6 (anti-bloat Redis + anti-rotation
		// d'adresse), IP complète en IPv4. Mêmes clés pour le score, les paths et le dédup d'alerte.
		keyIP := susKeyIP(ip)
		key := fmt.Sprintf("maicivy:sus:%s", keyIP)
		now := time.Now()
		if cfg.Now != nil {
			now = cfg.Now() // horloge injectable (replay multi-jours avec déclin)
		}

		// Score courant (déclin appliqué).
		score := susReadScore(ctx, cfg.Redis, key, now, halflifeSec)

		// Throttle probabiliste, croissance logarithmique.
		if p := susThrottleP(score); p > 0 && rand.Float64() < p {
			c.Set("Retry-After", strconv.Itoa(susRetryAfter))
			c.Set("X-RateLimit-Type", "sus-rate")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Ralentissement temporaire",
				"code":        "SUS_RATE_LIMIT",
				"retry_after": susRetryAfter,
				"message":     "Trop d'erreurs récentes depuis votre IP. Réessayez dans un instant.",
			})
		}

		// Laisser passer, observer le statut final.
		nextErr := c.Next()
		status := c.Response().StatusCode()
		if nextErr != nil {
			if fe, ok := nextErr.(*fiber.Error); ok {
				status = fe.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		isBad := status == fiber.StatusBadRequest || status == fiber.StatusForbidden ||
			status == fiber.StatusNotFound || status == fiber.StatusMethodNotAllowed
		// Signal "signature de path" : un chemin de scanner (.env, api-key, /config…) compte même
		// en 200 — c'est ce qui attrape le scan du frontend, aveugle au seul taux de 4xx.
		isScanner := cfg.ScannerPath != nil && cfg.ScannerPath(c.Path())
		if !isBad && !isScanner {
			// Crédit de densité : un vrai 2xx (contenu réel, non-scanner) ABAISSE le score (plancher 0).
			// POURQUOI : un user légitime génère surtout des SUCCÈS ; ils doivent absorber ses 4xx
			// incidents (lien mort, asset non couvert par l'exemption) pour qu'il n'atteigne jamais le
			// throttle. Le système mesure ainsi la DENSITÉ d'erreurs, pas le compte brut. Un scanner ne
			// génère quasi aucun 2xx → non concerné (et ses hits de signature scorent ailleurs). On
			// n'écrit QUE si un score existe déjà (>0) → aucun write Redis pour les 99% d'IP propres.
			if score > 0 && status >= 200 && status < 300 {
				credited := score - susSuccessCredit
				if credited < 0 {
					credited = 0
				}
				cfg.Redis.HSet(ctx, key, "score", strconv.FormatFloat(credited, 'f', 3, 64), "ts", now.Unix())
				cfg.Redis.Expire(ctx, key, memoryTTL)
			}
			return nextErr // requête légitime → le score n'augmente jamais (et décroît si succès)
		}

		// Erreur scanner → score += 1 (sur le score déjà décliné), persiste + échantillon de chemin.
		newScore := score + 1.0
		pipe := cfg.Redis.Pipeline()
		pipe.HSet(ctx, key, "score", strconv.FormatFloat(newScore, 'f', 3, 64), "ts", now.Unix())
		pipe.Expire(ctx, key, memoryTTL)
		pkey := fmt.Sprintf("maicivy:sus:paths:%s", keyIP)
		pipe.LPush(ctx, pkey, c.Path())
		pipe.LTrim(ctx, pkey, 0, 4)
		pipe.Expire(ctx, pkey, memoryTTL)
		if _, e := pipe.Exec(ctx); e != nil {
			log.Error().Err(e).Msg("sus-ratelimit: redis pipeline failed")
			return nextErr
		}

		// Alerte filou quand le score franchit le seuil.
		if newScore >= alertScore {
			maybeAlertSusIP(cfg, c, ip, newScore)
		}
		return nextErr
	}
}

// staticAssetExts : extensions d'ASSETS statiques (visuels, styles, scripts, fontes, sourcemaps).
// Un 4xx dessus = asset manquant (favicon, logo, css/js cassé, font absente), JAMAIS un scan — un
// scanner cible des chemins sans extension ou en .php/.env/.git (couverts par les signatures), pas
// /logo.png. L'exemption est de toute façon écrasée si le chemin matche une signature (cf. l'appel).
var staticAssetExts = map[string]bool{
	".ico": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".svg": true, ".webp": true, ".avif": true, ".bmp": true,
	".css": true, ".js": true, ".mjs": true, ".map": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,
}

// isAssetExtension : le chemin se termine-t-il par une extension d'asset statique ? On isole
// l'extension après le DERNIER point, et on rejette si un '/' suit ce point (ex: /a.b/c n'a pas
// d'extension). Distinct de isStaticAsset (analytics) : EXCLUT .json — un scanner probe /config.json
// et autres secrets en .json, on ne doit donc PAS exempter cette extension du scoring sus.
func isAssetExtension(p string) bool {
	i := strings.LastIndexByte(p, '.')
	if i < 0 {
		return false
	}
	ext := p[i:]
	if strings.IndexByte(ext, '/') >= 0 {
		return false
	}
	return staticAssetExts[strings.ToLower(ext)]
}

// benignPath : chemins générés par les navigateurs/SPA, jamais malveillants → exemptés du sus
// (ni score ni throttle). `/favicon.ico` est LE cas qui faisait grimper le score des vrais users
// (chaque onglet le redemande → 404 → bump). Les scanners ne ciblent pas ces chemins.
func benignPath(p string) bool {
	switch p {
	case "/favicon.ico", "/robots.txt", "/sitemap.xml", "/manifest.json",
		"/site.webmanifest", "/apple-touch-icon.png", "/apple-touch-icon-precomposed.png":
		return true
	}
	return strings.HasPrefix(p, "/_next/") || strings.HasPrefix(p, "/.well-known/")
}

// susReadScore lit {score, ts} et applique le déclin exponentiel (demi-vie en secondes).
func susReadScore(ctx context.Context, rdb *redis.Client, key string, now time.Time, halflifeSec float64) float64 {
	vals, err := rdb.HMGet(ctx, key, "score", "ts").Result()
	if err != nil {
		return 0
	}
	score := susParseFloat(vals[0])
	ts := int64(susParseFloat(vals[1]))
	if score <= 0 || ts <= 0 {
		return 0
	}
	age := float64(now.Unix() - ts)
	if age <= 0 {
		return score
	}
	return score * math.Pow(2, -age/halflifeSec)
}

// susThrottleP : proba de throttle, croissance LOGARITHMIQUE du score (0 sous susFreeScore, cap susMaxP).
func susThrottleP(score float64) float64 {
	if score <= susFreeScore {
		return 0
	}
	ratio := math.Log(score/susFreeScore) / math.Log(susFullScore/susFreeScore)
	if ratio > 1 {
		ratio = 1
	}
	return susMaxP * ratio
}

// susParseFloat convertit une valeur HMGet (string ou nil) en float64.
func susParseFloat(v interface{}) float64 {
	s, ok := v.(string)
	if !ok {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseAllowlist convertit des entrées "ip" ou "cidr" en réseaux. Une IP nue devient /32 (v4)
// ou /128 (v6). Les entrées invalides sont ignorées (pas de fail si une ligne est mal formée).
func parseAllowlist(entries []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.Contains(e, "/") {
			if strings.Contains(e, ":") {
				e += "/128"
			} else {
				e += "/32"
			}
		}
		if _, n, err := net.ParseCIDR(e); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

// ipAllowed dit si ip appartient à l'un des réseaux allowlistés.
func ipAllowed(ip string, nets []*net.IPNet) bool {
	if len(nets) == 0 {
		return false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}
