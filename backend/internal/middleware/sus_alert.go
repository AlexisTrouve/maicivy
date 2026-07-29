package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// maybeAlertSusIP alerte (log + webhook Discord) qu'une IP est clairement filou (score élevé).
// Dé-dupliqué à 1 alerte par IP par heure (SETNX) : un scanner persiste des jours, on ne veut
// pas une notif par requête.
func maybeAlertSusIP(cfg SusConfig, c *fiber.Ctx, ip string, score float64) {
	ctx := context.Background()
	keyIP := susKeyIP(ip) // mêmes clés que le scoring (/64 en IPv6) — sinon paths/dédup désalignés

	// Chemins récents (max 5) lus EN PREMIER : nécessaires pour classifier le type d'attaque.
	paths, _ := cfg.Redis.LRange(ctx, fmt.Sprintf("maicivy:sus:paths:%s", keyIP), 0, 4).Result()
	attackType := classifyAttack(paths)
	muted := susTypeMuted(attackType, cfg.AlertMuteTypes)

	// Dé-dup à DEUX canaux distincts, pour que muter un type ne masque ni ne multiplie les alertes :
	//   - non muté → clé "maicivy:sus:alerted:<ip>"       = 1 NOTIF / IP / heure (comportement normal ;
	//     un scanner multi-types non-muté → 1 seule alerte, pas un flood par type).
	//   - muté     → clé "maicivy:sus:alerted:muted:<ip>" = 1 LOG / IP / heure (anti-spam log), JAMAIS
	//     de webhook. Canal séparé → ne consomme PAS la clé non-mutée : un scan secrets juste après
	//     un scan php notifie quand même.
	akey := fmt.Sprintf("maicivy:sus:alerted:%s", keyIP)
	if muted {
		akey = fmt.Sprintf("maicivy:sus:alerted:muted:%s", keyIP)
	}
	set, err := cfg.Redis.SetNX(ctx, akey, "1", time.Hour).Result()
	if err != nil || !set {
		return // erreur Redis, ou déjà alerté (ce canal) dans l'heure
	}

	ua := c.Get("User-Agent")
	country := c.Get("CF-IPCountry") // posé par Cloudflare
	version, block := ipVersionAndBlock(ip)

	// 1. Log structuré — TOUJOURS (un type muté reste visible en log : muté ≠ invisible).
	log.Warn().
		Str("ip", ip).Str("ip_version", version).Str("block", block).Str("country", country).
		Float64("sus_score", score).Str("attack_type", attackType).Bool("alert_muted", muted).
		Str("user_agent", ua).Strs("sample_paths", paths).
		Msg("sus IP flagged (filou)")

	// 2. Webhook Discord — SEULEMENT si le type n'est pas muté. Async (ne bloque pas ; fail-safe).
	if cfg.WebhookURL != "" && !muted {
		go sendDiscordSusAlert(cfg.WebhookURL, ip, version, block, country, score, ua, paths, attackType)
	}
}

// classifyAttack catégorise une vague de scan à partir de ses chemins récents.
// QUOI : renvoie un label de type d'attaque ("php", "wordpress", "env", "config", "mixed", "unknown").
// POURQUOI : permet de muter certains types d'alertes (cf. SusConfig.AlertMuteTypes) — ex un scan
// webshell PHP est du bruit sur maicivy (aucun PHP servi), on ne veut pas la notif Discord.
// COMMENT : compte les chemins par catégorie (insensible casse, 1ère catégorie qui matche ; ".php"
// testé en premier → un /wp-login.php compte comme "php"). La catégorie strictement majoritaire
// (> moitié des chemins) donne le label ; sinon "mixed".
func classifyAttack(paths []string) string {
	if len(paths) == 0 {
		return "unknown"
	}
	counts := map[string]int{}
	for _, p := range paths {
		lp := strings.ToLower(p)
		switch {
		case strings.Contains(lp, ".php"):
			counts["php"]++
		case strings.Contains(lp, "wp-") || strings.Contains(lp, "wordpress"):
			counts["wordpress"]++
		case strings.Contains(lp, ".env"):
			counts["env"]++
		case strings.Contains(lp, "config") || strings.Contains(lp, "settings") ||
			strings.HasSuffix(lp, ".yml") || strings.HasSuffix(lp, ".yaml") ||
			strings.HasSuffix(lp, ".ini"):
			counts["config"]++
		default:
			counts["other"]++
		}
	}
	top, topN := "", 0
	for k, n := range counts {
		if n > topN {
			top, topN = k, n
		}
	}
	if topN*2 > len(paths) { // strictement majoritaire
		return top
	}
	return "mixed"
}

// susTypeMuted indique si un type d'attaque est dans la liste des types mutés (casse/espaces ignorés).
func susTypeMuted(t string, muted []string) bool {
	for _, m := range muted {
		if strings.EqualFold(strings.TrimSpace(m), t) {
			return true
		}
	}
	return false
}

// susKeyIP retourne l'identité d'agrégation d'une IP pour les clés sus (score, paths, dédup alerte).
// QUOI : IPv4 → l'IP entière (équivaut /32) ; IPv6 → le préfixe /64 (sans suffixe).
// POURQUOI : en IPv6 un attaquant dispose souvent d'un /64 entier (2^64 adresses). Sans agrégation
// il créerait une clé Redis par /128 (bloat mémoire → éviction → fail-open = tout le rate-limit
// désactivé) ET échapperait au scoring en tournant d'adresse. Un /64 = typiquement un seul
// client/foyer → l'agréger est correct, pas juste défensif (cohérent avec le /64 d'ipVersionAndBlock).
func susKeyIP(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip // IP non parsable → telle quelle (pas de surprise)
	}
	if parsed.To4() != nil {
		return ip // IPv4 : clé = IP complète
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String() // IPv6 : clé = préfixe /64
}

// ipVersionAndBlock retourne ("IPv4"|"IPv6", bloc à bannir). En IPv6 on donne le /64 (un même
// user change de /128 dans son /64 → c'est le /64 qu'on bannit) ; en IPv4 le /32 (l'IP elle-même).
func ipVersionAndBlock(ip string) (string, string) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "?", ip
	}
	if parsed.To4() != nil {
		return "IPv4", ip + "/32"
	}
	masked := parsed.Mask(net.CIDRMask(64, 128))
	return "IPv6", masked.String() + "/64"
}

// sendDiscordSusAlert poste une embed Discord décrivant le filou. Best-effort (timeout 5s).
func sendDiscordSusAlert(webhookURL, ip, version, block, country string, score float64, ua string, paths []string, attackType string) {
	if country == "" {
		country = "?"
	}
	if ua == "" {
		ua = "(vide)"
	}
	pathsStr := strings.Join(paths, "\n")
	if pathsStr == "" {
		pathsStr = "(aucun)"
	}

	payload := map[string]interface{}{
		"username": "maicivy sentinel",
		"embeds": []map[string]interface{}{{
			"title": "🚨 Filou détecté",
			"color": 15158332, // rouge
			"fields": []map[string]interface{}{
				{"name": "IP", "value": fmt.Sprintf("`%s`", ip), "inline": true},
				{"name": "Type", "value": attackType, "inline": true},
				{"name": "Version", "value": version, "inline": true},
				{"name": "Pays", "value": country, "inline": true},
				{"name": "Bloc à bannir", "value": fmt.Sprintf("`%s`", block), "inline": true},
				{"name": "Score sus", "value": fmt.Sprintf("%.1f", score), "inline": true},
				{"name": "User-Agent", "value": susTruncate(ua, 1000)},
				{"name": "Chemins récents", "value": susTruncate(pathsStr, 1000)},
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("sus-alert: marshal webhook failed")
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Error().Err(err).Msg("sus-alert: webhook POST failed")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Error().Int("status", resp.StatusCode).Msg("sus-alert: webhook non-2xx")
	}
}

// susTruncate coupe une string à n runes (champs d'embed Discord limités à 1024).
func susTruncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
