// Command attackcapture convertit des lignes de log nginx ("combined") en une trace
// d'attaque JSONL rejouable, pour tester/tuner la défense anti-abus sur du comportement RÉEL.
//
// QUOI    : lit des lignes nginx sur stdin, filtre par IP, émet sur stdout une trace JSONL :
//
//	1 ligne meta + 1 ligne par requête {t, m, p, st, b, ua}.
//
// POURQUOI: capturer le "full behavior" d'un filou (ex: 185.177.72.52) pour le REJOUER en
//
//	test — la preuve qu'une défense marche = rejouer la vraie attaque, pas lire le
//	code (doctrine TDD adversarial + persistant : les fixtures restent en régression).
//
// COMMENT : 1. regex sur le format combined, 2. parse l'heure (layout nginx), 3. tri par
//
//	temps, 4. t = secondes depuis la 1re requête, 5. JSONL compact (champs courts car
//	les fixtures font plusieurs milliers de lignes). Les lignes malformées (scans TLS
//	sur port HTTP, etc.) sont ignorées proprement.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// logLine matche le format nginx "combined" :
//
//	$remote_addr - $remote_user [$time_local] "$request" $status $bytes "$referer" "$ua"
//
// On capture la requête entière (groupe 3) puis on la découpe en Go : une regex qui tente de
// séparer méthode/path/proto casse sur les requêtes malformées, ce découpage est plus robuste.
var logLine = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([^"]*)" (\d{3}) (\S+) "[^"]*" "([^"]*)"`)

// nginxTime = layout du $time_local nginx (ex: 06/Jun/2026:12:31:58 +0000).
const nginxTime = "02/Jan/2006:15:04:05 -0700"

// req = une requête capturée. Champs courts (t/m/p/st/b/ua) : fichiers de milliers de lignes.
// ts est interne (non exporté → ignoré par JSON) et sert uniquement au tri chronologique.
type req struct {
	T  float64 `json:"t"`  // secondes depuis la 1re requête de la trace
	M  string  `json:"m"`  // méthode HTTP
	P  string  `json:"p"`  // path demandé
	St int     `json:"st"` // status HTTP renvoyé
	B  int     `json:"b"`  // octets envoyés (corps)
	UA string  `json:"ua"` // user-agent
	ts time.Time
}

// meta = en-tête de la trace (1re ligne JSONL), repérable par "type":"meta".
type meta struct {
	Type  string  `json:"type"`
	IP    string  `json:"ip"`
	UA    string  `json:"ua"`
	Total int     `json:"total"`
	DurS  float64 `json:"dur_s"`
	First string  `json:"first"`
}

// parseLine parse une ligne nginx combined. ok=false si non parsable (ligne malformée).
func parseLine(line string) (ip string, r req, ok bool) {
	m := logLine.FindStringSubmatch(line)
	if m == nil {
		return "", req{}, false
	}
	ts, err := time.Parse(nginxTime, m[2])
	if err != nil {
		return "", req{}, false
	}
	// m[3] = "METHOD PATH PROTO" — on découpe ; une requête sans path (< 2 champs) est rejetée.
	fields := strings.Fields(m[3])
	if len(fields) < 2 {
		return "", req{}, false
	}
	st, _ := strconv.Atoi(m[4])
	b, _ := strconv.Atoi(m[5]) // "-" → 0 (pas d'octets)
	return m[1], req{M: fields[0], P: fields[1], St: st, B: b, UA: m[6], ts: ts}, true
}

// stripQuery retire la query string (?...) d'un path — anonymisation du corpus légitime (PII).
func stripQuery(p string) string {
	if i := strings.IndexByte(p, '?'); i >= 0 {
		return p[:i]
	}
	return p
}

func main() {
	ip := flag.String("ip", "", "ne garder que cette IP (vide = toutes)")
	anon := flag.String("anon", "", "anonymise la sortie : remplace l'IP (meta) par ce label + retire les query strings (corpus légitime)")
	flag.Parse()

	// 1. Lecture + parse de toutes les lignes (buffer large : certaines lignes sont longues).
	var reqs []req
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		lip, r, ok := parseLine(sc.Text())
		if !ok || (*ip != "" && lip != *ip) {
			continue
		}
		reqs = append(reqs, r)
	}
	if len(reqs) == 0 {
		return
	}

	// 2. Tri chronologique (les logs peuvent venir de plusieurs fichiers/jours).
	sort.Slice(reqs, func(i, j int) bool { return reqs[i].ts.Before(reqs[j].ts) })

	// 3. Émission : meta puis chaque requête avec t = secondes depuis la 1re.
	first := reqs[0].ts
	metaIP := *ip
	if *anon != "" {
		metaIP = *anon // corpus légitime : on ne committe jamais une vraie IP de visiteur
	}
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(meta{
		Type:  "meta",
		IP:    metaIP,
		UA:    reqs[0].UA,
		Total: len(reqs),
		DurS:  reqs[len(reqs)-1].ts.Sub(first).Seconds(),
		First: first.UTC().Format(time.RFC3339),
	})
	for _, r := range reqs {
		r.T = r.ts.Sub(first).Seconds()
		if *anon != "" {
			r.P = stripQuery(r.P) // retire ?... (PII potentielle) pour le corpus légitime
		}
		_ = enc.Encode(r)
	}
}
