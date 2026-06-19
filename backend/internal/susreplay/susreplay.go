// Package susreplay rejoue une trace d'attaque (capturée par tools/attackcapture) contre le
// vrai middleware anti-abus, et mesure ce que la défense fait.
//
// QUOI    : charge une trace JSONL, envoie chaque requête (filtrée par `scope`) à travers
//
//	middleware.SusRateLimit sur une stack jetable (Fiber + miniredis), et compte ce
//	qui est throttlé / passé + le score final + si l'alerte a sauté.
//
// POURQUOI: ② du pipeline capture→replay→tune. La preuve qu'une défense marche = rejouer la
//
//	VRAIE attaque, pas lire le code. Réutilisable par le tuner ③ (même fonction Replay,
//	configs différentes). Hors image Docker (cmd/ uniquement importe le serveur).
//
// COMMENT : `scope(req)` décide quelles requêtes atteignent le checkpoint (modélise le routage
//
//	nginx : en prod seul /api non-bloqué arrivait au backend). `statusOf(req)` donne le
//	status que le handler renvoie (ex: 404 pour un path /api inexistant) → c'est lui que
//	le middleware "voit" pour scorer. Le throttle de notre middleware est repéré via le
//	header X-RateLimit-Type=sus-rate qu'il pose sur ses 429.
package susreplay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"maicivy/internal/middleware"
)

// Req = une requête de la trace (mêmes champs courts que la fixture).
type Req struct {
	T  float64 `json:"t"`
	M  string  `json:"m"`
	P  string  `json:"p"`
	St int     `json:"st"`
	B  int     `json:"b"`
	UA string  `json:"ua"`
}

// Meta = en-tête de la trace.
type Meta struct {
	IP    string  `json:"ip"`
	UA    string  `json:"ua"`
	Total int     `json:"total"`
	DurS  float64 `json:"dur_s"`
	First string  `json:"first"` // timestamp RFC3339 de la 1re requête (pour l'horloge simulée)
}

// Trace = une attaque complète chargée depuis une fixture.
type Trace struct {
	Meta Meta
	Reqs []Req
}

// LoadTrace lit une fixture JSONL (1 ligne meta + N lignes requête).
func LoadTrace(path string) (*Trace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tr := &Trace{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		b := sc.Bytes()
		if bytes.Contains(b, []byte(`"type":"meta"`)) {
			_ = json.Unmarshal(b, &tr.Meta)
			continue
		}
		var r Req
		if json.Unmarshal(b, &r) == nil && r.P != "" {
			tr.Reqs = append(tr.Reqs, r)
		}
	}
	return tr, sc.Err()
}

// Metrics = ce que la défense a fait sur la trace rejouée.
type Metrics struct {
	Fed        int     // requêtes envoyées au checkpoint (ayant passé `scope`)
	Throttled  int     // throttlées par NOTRE middleware (429 sus-rate)
	Reached    int     // passées le throttle (ont atteint le handler)
	FinalScore float64 // score Redis final de l'IP
	Alerted    bool    // l'alerte filou a-t-elle sauté ?
}

// ThrottleRate = fraction throttlée (0 si rien envoyé).
func (m Metrics) ThrottleRate() float64 {
	if m.Fed == 0 {
		return 0
	}
	return float64(m.Throttled) / float64(m.Fed)
}

// Replay rejoue les requêtes de tr passant `scope` à travers le middleware configuré par cfg.
// statusOf donne le status que le handler renvoie pour chaque requête. cfg.Redis est écrasé
// par un miniredis jetable propre à ce replay.
func Replay(tr *Trace, cfg middleware.SusConfig, scope func(Req) bool, statusOf func(Req) int) (Metrics, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return Metrics{}, err
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfg.Redis = rdb

	// Horloge simulée : on avance le temps selon r.T (secondes depuis la 1re requête) pour que
	// le déclin du score s'applique sur les traces longues (sessions multi-jours réelles).
	base, _ := time.Parse(time.RFC3339, tr.Meta.First)
	var simNow time.Time
	if !base.IsZero() {
		cfg.Now = func() time.Time { return simNow }
	}

	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(middleware.SusRateLimit(cfg))
	// Handler universel : renvoie le status passé en header (le "vrai" status du backend).
	app.All("/*", func(c *fiber.Ctx) error {
		st, _ := strconv.Atoi(c.Get("X-Replay-Status"))
		if st == 0 {
			st = 200
		}
		return c.SendStatus(st)
	})

	var m Metrics
	for _, r := range tr.Reqs {
		if scope != nil && !scope(r) {
			continue
		}
		simNow = base.Add(time.Duration(r.T * float64(time.Second))) // avance l'horloge simulée
		method := r.M
		if method == "" {
			method = "GET"
		}
		// http.NewRequest renvoie une erreur (pas de panic) sur un path tordu → on skippe.
		req, e := http.NewRequest(method, "http://replay"+r.P, nil)
		if e != nil {
			continue
		}
		req.Header.Set("X-Real-IP", tr.Meta.IP)
		req.Header.Set("User-Agent", r.UA)
		req.Header.Set("X-Replay-Status", strconv.Itoa(statusOf(r)))

		resp, e := app.Test(req, -1)
		if e != nil {
			continue
		}
		m.Fed++
		if resp.Header.Get("X-RateLimit-Type") == "sus-rate" {
			m.Throttled++
		} else {
			m.Reached++
		}
	}

	// Score final + alerte, lus depuis Redis.
	ctx := context.Background()
	if v, e := rdb.HGet(ctx, "maicivy:sus:"+tr.Meta.IP, "score").Result(); e == nil {
		m.FinalScore, _ = strconv.ParseFloat(v, 64)
	}
	if n, _ := rdb.Exists(ctx, "maicivy:sus:alerted:"+tr.Meta.IP).Result(); n > 0 {
		m.Alerted = true
	}
	return m, nil
}
