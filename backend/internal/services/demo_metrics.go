package services

import (
	"context"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// demo_metrics.go — générateur procédural d'analytics « vivantes », seedé par l'activité git (commits)
// et BORNÉ par le trafic réel. cf. CLAUDE.md §Analytics. Activé par DEMO_METRICS.
//
// MODÈLE (par métrique) :
//
//	affiché = réel + synthétique
//	synthétique = floor + (potentiel_commits − floor) · gate(R)
//	gate(R) = bump(R / convergence)      avec  bump(x) = x · e^(1−x)
//
// où R = niveau de VRAIS users (visiteurs uniques). Conséquences voulues par Alexi :
//   - R petit (peu de vrais users) → gate ≈ 0 → synthétique ≈ floor, MÊME si commits élevés.
//   - R monte → gate monte → le synthétique exprime le potentiel des commits.
//   - R ≫ convergence → gate → 0 → le réel domine (« laisse la place aux vrais users »).
//
// Les commits fixent le POTENTIEL (hauteur du pic) + la personnalité ; la liveness minute/minute vient
// d'une courbe horaire + jitter déterministe. Les TOTAUX cumulés (lettres, lectures blog) passent par un
// ratchet Redis → monotones (un total qui baisse = grillé).
//
// Tout est DÉTERMINISTE (fonction du temps + commits) : cohérent pour tous les visiteurs au même instant,
// survit au restart, zéro pollution de la vraie base. Toggle off → 100 % réel.

// --- Constantes de calibrage (tunables) ----------------------------------------------------------

const (
	// demoConvergence — niveau de vrais visiteurs/jour où le synthétique culmine (« point commun »),
	// puis s'efface. C'est LE bouton principal.
	demoConvergence = 120.0

	// Planchers : vie minimale quand le site est calme (R≈0). Volontairement bas.
	demoFloorOnline   = 1.0
	demoFloorVisitors = 12.0
	demoFloorLetters  = 30.0
	demoFloorBlog     = 25.0

	// Clé Redis du ratchet du total cumulé (lectures blog) : offset synthétique monotone, séparé du réel.
	demoKeyBlogOffset = "demo:blog:offset"
)

// BlogReadsTotalKey — compteur Redis du nombre RÉEL de lectures d'articles (INCR par le handler blog
// quand le backend sert un article). Lu par l'analytics pour le blend.
const BlogReadsTotalKey = "blog:reads:total"

// --- Type principal ------------------------------------------------------------------------------

// DemoMetrics produit les valeurs synthétiques. gitea = source de commits (cachée, cf. SWR) ; nil
// toléré (seed neutre). now injectable pour les tests déterministes.
type DemoMetrics struct {
	gitea   *GiteaStatsService
	redis   *redis.Client
	enabled bool
	now     func() time.Time
}

// NewDemoMetrics crée le générateur. enabled=false → tous les Blend* renvoient la valeur réelle telle
// quelle (passthrough), ce qui rend le toggle DEMO_METRICS sans effet de bord.
func NewDemoMetrics(gitea *GiteaStatsService, rdb *redis.Client, enabled bool) *DemoMetrics {
	return &DemoMetrics{
		gitea:   gitea,
		redis:   rdb,
		enabled: enabled,
		now:     time.Now,
	}
}

func (d *DemoMetrics) Enabled() bool { return d != nil && d.enabled }

// --- Math pure (testable sans réseau) ------------------------------------------------------------

// bump : 0 en x=0, =1 en x=1 (convergence), retombe ensuite. C'est la forme qui fait « monter avec les
// vrais users jusqu'au point commun puis s'effacer ».
func bump(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return x * math.Exp(1-x)
}

// gate : facteur d'échelle [0,1+] piloté par le réel R, partagé par toutes les métriques.
func gate(realUsers float64) float64 {
	return bump(realUsers / demoConvergence)
}

// hourCurve : rythme jour/nuit ∈ [0.35, 1.0], pic vers 15h, creux vers 3h du matin. Donne le « quand ».
func hourCurve(t time.Time) float64 {
	h := float64(t.Hour()) + float64(t.Minute())/60.0
	// cosinus centré sur 15h (max) / 3h (min).
	c := (1 + math.Cos((h-15)/24*2*math.Pi)) / 2 // ∈ [0,1]
	return 0.35 + 0.65*c
}

// procUnit : pseudo-aléatoire déterministe ∈ [0,1) à partir d'un seed (splitmix64). Sert au jitter et à
// la personnalité — même seed → même valeur (cohérence + reproductibilité des tests).
func procUnit(seed uint64) float64 {
	z := seed + 0x9E3779B97F4A7C15
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	z = z ^ (z >> 31)
	return float64(z>>11) / float64(1<<53)
}

// blendFlow : modèle de base pour une métrique de FLUX (non cumulée). real + plancher/potentiel scalés
// par gate(R), modulés par la courbe horaire. Pur.
func blendFlow(real, realUsers, floor, potential float64, t time.Time) float64 {
	synth := (floor + (potential-floor)*gate(realUsers)) * hourCurve(t)
	if synth < 0 {
		synth = 0
	}
	return real + synth
}

// --- Seed commits --------------------------------------------------------------------------------

// commitSeed résume l'activité git pour dériver les potentiels + une graine PRNG.
type commitSeed struct {
	total   int    // commits sur 6 mois (ampleur)
	recent7 int    // commits des 7 derniers jours (intensité récente)
	seed    uint64 // graine déterministe pour le jitter
}

// loadSeed lit les stats git cachées (SWR → quasi gratuit) et calcule le seed. gitea nil / erreur →
// seed neutre (le synthétique retombe sur ses planchers, jamais de crash).
func (d *DemoMetrics) loadSeed(ctx context.Context) commitSeed {
	if d.gitea == nil {
		return commitSeed{seed: 0x1234}
	}
	stats, err := d.gitea.GetStats(ctx, false)
	if err != nil || stats == nil {
		return commitSeed{seed: 0x1234}
	}
	cutoff := d.now().AddDate(0, 0, -7).Format("2006-01-02")
	recent := 0
	for _, day := range stats.Daily {
		if day.Date >= cutoff {
			recent += day.Commits
		}
	}
	return commitSeed{
		total:   stats.TotalCommits,
		recent7: recent,
		seed:    uint64(stats.TotalCommits)*2654435761 + uint64(recent)*40503,
	}
}

// Potentiels = plafond du synthétique, dérivés des commits (caps pour rester crédible). Tunables.
func (s commitSeed) onlinePotential() float64 {
	return demoFloorOnline + math.Min(9, float64(s.recent7)/5.0)
}
func (s commitSeed) visitorsPotential() float64 {
	return demoFloorVisitors + math.Min(290, float64(s.recent7)*3+float64(s.total)/40.0)
}
func (s commitSeed) lettersPotential() float64 {
	// Volontairement modeste (< potentiel visiteurs) pour une conversion lettres/visiteurs crédible.
	return demoFloorLetters + math.Min(180, float64(s.total)/30.0)
}
func (s commitSeed) blogPotential() float64 {
	return demoFloorBlog + math.Min(370, float64(s.total)/15.0)
}

// --- Blend des métriques de flux -----------------------------------------------------------------

// Online : compteur « en ligne maintenant ». Réel + synthétique gaté + jitter ~30s pour respirer.
func (d *DemoMetrics) Online(ctx context.Context, real int64, realUsers float64) int64 {
	if !d.Enabled() {
		return real
	}
	s := d.loadSeed(ctx)
	t := d.now()
	base := blendFlow(float64(real), realUsers, demoFloorOnline, s.onlinePotential(), t)
	// Jitter ±1 qui change toutes les ~30s (bucket temps) — la « respiration » du live.
	bucket := uint64(t.Unix() / 30)
	jitter := math.Round(procUnit(s.seed^bucket)*2) - 1 // ∈ {-1,0,1}
	v := math.Round(base) + jitter
	if v < float64(real) {
		v = float64(real) // jamais en dessous du réel
	}
	if v < 1 {
		v = 1 // jamais 0 quand activé : il y a "toujours quelqu'un" (un 0 sur le gros compteur = effet mort)
	}
	return int64(v)
}

// VisitorsToday : visiteurs uniques du jour. realUsers EST le réel ici (le gate s'auto-applique).
func (d *DemoMetrics) VisitorsToday(ctx context.Context, real int64, realUsers float64) int64 {
	if !d.Enabled() {
		return real
	}
	s := d.loadSeed(ctx)
	return int64(math.Round(blendFlow(float64(real), realUsers, demoFloorVisitors, s.visitorsPotential(), d.now())))
}

// PageViews : vues de page. Dérivé ≈ 2.4× le potentiel visiteurs (un visiteur lit plusieurs pages).
func (d *DemoMetrics) PageViews(ctx context.Context, real int64, realUsers float64) int64 {
	if !d.Enabled() {
		return real
	}
	s := d.loadSeed(ctx)
	return int64(math.Round(blendFlow(float64(real), realUsers, demoFloorVisitors*2.4, s.visitorsPotential()*2.4, d.now())))
}

// --- Blend des TOTAUX (ratchet monotone) ---------------------------------------------------------

// ratchetTotal : offset synthétique MONOTONE en Redis. À chaque lecture, on vise target = potentiel·gate
// et on ne descend jamais (max) → le total affiché ne recule jamais (un total qui baisse = grillé). Le
// réel s'ajoute par-dessus. enabled=false → 0. redis nil (tests) → target direct (toujours monotone car
// target croît avec le temps de présence des vrais users).
func (d *DemoMetrics) ratchetTotal(ctx context.Context, key string, target float64) float64 {
	if target < 0 {
		target = 0
	}
	if d.redis == nil {
		return target
	}
	stored, err := d.redis.Get(ctx, key).Float64()
	if err != nil {
		stored = 0
	}
	offset := math.Max(stored, target)
	if offset > stored {
		// Pas de TTL : l'offset persiste et ne fait que monter (ratchet).
		d.redis.Set(ctx, key, offset, 0)
	}
	return offset
}

// Letters : lettres générées sur la période — métrique de FLUX (sert aussi à la conversion). Pas de
// ratchet : un flux peut varier sans paraître cassé, contrairement à un total cumulé.
func (d *DemoMetrics) Letters(ctx context.Context, real int64, realUsers float64) int64 {
	if !d.Enabled() {
		return real
	}
	s := d.loadSeed(ctx)
	return int64(math.Round(blendFlow(float64(real), realUsers, demoFloorLetters, s.lettersPotential(), d.now())))
}

// BlogReadsTotal : total de lectures blog affiché = réel + offset synthétique ratcheté.
func (d *DemoMetrics) BlogReadsTotal(ctx context.Context, real int64, realUsers float64) int64 {
	if !d.Enabled() {
		return real
	}
	s := d.loadSeed(ctx)
	target := demoFloorBlog + (s.blogPotential()-demoFloorBlog)*gate(realUsers)
	return real + int64(math.Round(d.ratchetTotal(ctx, demoKeyBlogOffset, target)))
}

// --- Heatmap synthétique (zones chaudes labellisées) ---------------------------------------------

// heatZone : zone chaude synthétique FIXE — position en % du viewport + libellé d'interaction + poids
// relatif. POURQUOI fixe : la heatmap réelle (clics) est trop éparse pour être lisible ; on superpose
// des zones plausibles labellisées pour qu'elle "raconte" quelque chose (où l'on clique typiquement).
type heatZone struct {
	x, y   float64
	label  string
	weight float64
}

// Zones plausibles d'un portfolio CV (CTA, nav, pastilles, footer). Libellés en FR (langue primaire ;
// les clics RÉELS portent déjà le libellé dans la langue du visiteur → données mixtes, assumé).
var demoHeatZones = []heatZone{
	{25, 52, "CTA — Voir le CV", 1.0},
	{58, 52, "CTA — Générer une lettre", 0.85},
	{42, 60, "Pastille compétence", 0.7},
	{88, 6, "Nav — Chat", 0.55},
	{50, 30, "Sélecteur de thème CV", 0.5},
	{50, 78, "Lien — Lire l'article", 0.5},
	{15, 6, "Logo / Accueil", 0.4},
	{72, 90, "Footer — GitHub", 0.35},
}

// demoHeatFloor : clics synthétiques mini par zone (× poids) → heatmap toujours lisible, même calme.
const demoHeatFloor = 4.0

func (s commitSeed) heatmapPotential() float64 {
	return demoHeatFloor + math.Min(60, float64(s.total)/40.0)
}

// HeatmapPoints génère les points de heatmap synthétiques (zones chaudes labellisées), gatés par les
// vrais users comme le reste. Déterministe (respire lentement, bucket 10 min). nil si désactivé.
func (d *DemoMetrics) HeatmapPoints(ctx context.Context, realUsers float64) []map[string]interface{} {
	if !d.Enabled() {
		return nil
	}
	s := d.loadSeed(ctx)
	base := demoHeatFloor + (s.heatmapPotential()-demoHeatFloor)*gate(realUsers)
	bucket := uint64(d.now().Unix() / 600) // la heatmap évolue toutes les ~10 min
	out := make([]map[string]interface{}, 0, len(demoHeatZones))
	for i, z := range demoHeatZones {
		// Jitter déterministe ±20% par zone (casse l'uniformité sans tout faire bouger).
		j := 0.8 + 0.4*procUnit(s.seed^bucket^uint64(i)*2654435761)
		c := int(math.Round(z.weight * base * j))
		if c < 1 {
			c = 1
		}
		out = append(out, map[string]interface{}{
			"x": z.x, "y": z.y, "count": c, "intensity": c, "element": z.label,
		})
	}
	return out
}
