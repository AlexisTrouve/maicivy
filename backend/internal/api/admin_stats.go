package api

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/middleware"
	"maicivy/internal/models"
)

// AdminStatsHandler — dashboard "stats privées" owner-only (GET /admin/stats). Agrège des métriques
// déjà présentes (Redis + DB) en 3 sections : coûts/usage IA, abus/sécurité, analytics détaillées.
// POURQUOI un seul endpoint : un panneau owner simple, une requête. Chaque section dégrade en vide si
// sa source est indispo (l'endpoint ne casse jamais entièrement). Auth = cookie admin (comme /admin/*).
type AdminStatsHandler struct {
	db            *gorm.DB
	redis         *redis.Client
	sessionSecret string
}

func NewAdminStatsHandler(db *gorm.DB, redis *redis.Client, sessionSecret string) *AdminStatsHandler {
	return &AdminStatsHandler{db: db, redis: redis, sessionSecret: sessionSecret}
}

func (h *AdminStatsHandler) RegisterRoutes(api fiber.Router) {
	api.Get("/admin/stats", h.GetStats)
}

// € estimés par MILLION de tokens, par modèle (blended in/out, APPROXIMATIF — generated_letters ne
// stocke que le total de tokens, pas le split). Sert un ordre de grandeur, pas une facture.
var costPerMTok = map[string]float64{
	"claude-opus-4-6":           35,
	"claude-opus-4-5":           35,
	"claude-haiku-4-5-20251001": 2,
	"claude-haiku-4-5":          2,
}

func GetStatsAuth(c *fiber.Ctx, secret string) bool {
	return middleware.VerifyAdminCookie(c.Cookies("maicivy_admin"), secret)
}

func (h *AdminStatsHandler) GetStats(c *fiber.Ctx) error {
	// Owner-only : même garde que le reste de /admin (cookie HMAC signé).
	if !GetStatsAuth(c, h.sessionSecret) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "owner only"})
	}
	ctx := context.Background()
	return c.JSON(fiber.Map{
		"ai":        h.aiStats(ctx),
		"security":  h.securityStats(ctx),
		"analytics": h.analyticsStats(),
	})
}

// --- Section 1 : coûts / usage IA ---
func (h *AdminStatsHandler) aiStats(ctx context.Context) fiber.Map {
	// Générations IA aujourd'hui (compteur global, toutes features/IP) — Redis.
	genToday, _ := h.redis.Get(ctx, "ratelimit:ai:global:daily").Int()

	// Lettres du mois courant, groupées par modèle (seule feature qui persiste tokens en DB).
	type row struct {
		AIModel string
		Cnt     int64
		Tokens  int64
	}
	var rows []row
	monthStart := time.Now().AddDate(0, 0, -(time.Now().Day() - 1))
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, monthStart.Location())
	if h.db != nil {
		h.db.Model(&models.GeneratedLetter{}).
			Select("ai_model, count(*) as cnt, coalesce(sum(tokens_used),0) as tokens").
			Where("created_at >= ?", monthStart).
			Group("ai_model").Scan(&rows)
	}

	byModel := make([]fiber.Map, 0, len(rows))
	var totalTokens int64
	var totalCost float64
	for _, r := range rows {
		cost := float64(r.Tokens) / 1_000_000 * costPerMTok[r.AIModel] // 0 si modèle inconnu
		totalTokens += r.Tokens
		totalCost += cost
		byModel = append(byModel, fiber.Map{"model": r.AIModel, "count": r.Cnt, "tokens": r.Tokens, "cost_eur": cost})
	}

	return fiber.Map{
		"generations_today":  genToday,
		"letters_this_month": byModel,
		"letters_tokens":     totalTokens,
		"letters_cost_eur":   totalCost,
		// Honnêteté : seules les lettres persistent leur coût. CV/messages/chat non trackés en DB.
		"note": "letters only — CV/messages/chat not persisted",
	}
}

// --- Section 2 : abus / sécurité ---
func (h *AdminStatsHandler) securityStats(ctx context.Context) fiber.Map {
	type susIP struct {
		IP    string   `json:"ip"`
		Score float64  `json:"score"`
		Paths []string `json:"paths"`
	}
	flagged := []susIP{}

	// Scan des clés de score sus (maicivy:sus:<ip>), en excluant les sous-clés paths/alerted.
	var cursor uint64
	for {
		keys, cur, err := h.redis.Scan(ctx, cursor, "maicivy:sus:*", 300).Result()
		if err != nil {
			break
		}
		for _, k := range keys {
			if strings.Contains(k, ":paths:") || strings.Contains(k, ":alerted:") {
				continue
			}
			ip := strings.TrimPrefix(k, "maicivy:sus:")
			scoreStr, _ := h.redis.HGet(ctx, k, "score").Result()
			score, _ := strconv.ParseFloat(scoreStr, 64)
			if score <= 0 {
				continue
			}
			paths, _ := h.redis.LRange(ctx, "maicivy:sus:paths:"+ip, 0, 4).Result()
			flagged = append(flagged, susIP{IP: ip, Score: score, Paths: paths})
		}
		cursor = cur
		if cursor == 0 {
			break
		}
	}
	// Tri par score décroissant, top 25.
	sort.Slice(flagged, func(i, j int) bool { return flagged[i].Score > flagged[j].Score })
	if len(flagged) > 25 {
		flagged = flagged[:25]
	}

	return fiber.Map{
		"flagged_ips":   flagged,
		"flagged_count": len(flagged),
	}
}

// --- Section 3 : analytics détaillées (données stockées mais non exposées publiquement) ---
func (h *AdminStatsHandler) analyticsStats() fiber.Map {
	type kv struct {
		K   string
		Cnt int64
	}
	byProfile := []fiber.Map{}
	topRef := []fiber.Map{}
	if h.db != nil {
		var prof []kv
		h.db.Model(&models.Visitor{}).
			Select("profile_detected as k, count(*) as cnt").
			Group("profile_detected").Order("cnt desc").Scan(&prof)
		for _, p := range prof {
			byProfile = append(byProfile, fiber.Map{"profile": p.K, "count": p.Cnt})
		}

		var refs []kv
		h.db.Model(&models.AnalyticsEvent{}).
			Where("referrer <> '' AND referrer IS NOT NULL").
			Select("referrer as k, count(*) as cnt").
			Group("referrer").Order("cnt desc").Limit(10).Scan(&refs)
		for _, r := range refs {
			topRef = append(topRef, fiber.Map{"referrer": r.K, "count": r.Cnt})
		}
	}
	return fiber.Map{
		"by_profile":    byProfile,
		"top_referrers": topRef,
	}
}
