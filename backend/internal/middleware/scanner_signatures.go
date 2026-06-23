package middleware

import (
	"net/url"
	"regexp"
	"strings"
)

// scanner_signatures.go — signal "signature de path" du sus-rate-limit.
//
// QUOI    : motifs de path révélateurs d'un scanner de secrets (.env, api-key, /config…),
//           minés depuis de VRAIES attaques (185.177.72.52) — chacun mesuré à 0 overlap sur le
//           corpus légitime.
// POURQUOI: le score basé sur les 4xx est aveugle au scan du frontend (qui renvoie 200 à tout).
//           Reconnaître le scanner par ses PATHS rattrape ce trou — un vrai navigateur ne
//           demande jamais ces chemins.
// COMMENT : ScannerPathMatcher compile les motifs en une regex et DÉCODE les %xx avant de matcher
//           (pour attraper les évasions type %2eenv). Le tuning (internal/susreplay) évalue
//           chaque signature contre attaque + légitime et ne garde que celles à 0 faux positif.

// ScannerSignatureDefs = signatures candidates (nom → motif, insensible à la casse). L'ordre
// n'a pas d'importance ; le tuning décide lesquelles activer selon catch-rate vs faux-positifs.
var ScannerSignatureDefs = []struct{ Name, Pattern string }{
	{"dotenv", `\.env`},
	{"git", `\.git`},
	{"aws", `\.aws`},
	{"apikey", `api-key`},
	{"credential", `credential`},
	{"mailcreds", `sendgrid|mailgun`},
	{"phpinfo", `phpinfo`},
	{"wordpress", `wp-`},
	{"bakext", `\.bak`},
	{"sqlext", `\.sql`},
	{"distext", `\.dist`},
	{"properties", `\.properties`},
	{"config", `/config`},
	{"admin", `/admin`},
	{"secret", `secret`},
	{"backup", `backup`},
	{"yaml", `\.ya?ml`},
	{"token", `token`},
	{"envlocal", `\.local`},
	// — Scanners de services d'automatisation/IA/infra exposés (n8n, Langflow, Docker API), minés du
	//   scanner 103.153.183.92 (juin 2026, Go-http-client) qui sprayait ces chemins. Préfixes COMPLETS
	//   et distinctifs → 0 overlap avec l'API maicivy (letters/cv/blog/analytics/gitstats…) ni les
	//   assets Next (un chunk `components-x.js` ne contient pas `/api/v1/components`). Validés 0 faux
	//   positif par TestTune_ScannerSignatures (corpus légitime synth + historique).
	//   NB : on n'ajoute PAS `webhook-test` (un futur slug blog "webhook-testing…" matcherait → FP) ;
	//   la densité de 404 l'attrape déjà de toute façon.
	{"n8nworkflows", `/api/v1/workflows`},
	{"n8nexecutions", `/api/v1/executions`},
	{"n8nrest", `/rest/workflows`},
	{"langflowflows", `/api/v1/flows`},
	{"langflowcomponents", `/api/v1/components`},
	{"dockerapi", `docker/api`},
}

// ScannerPathMatcher compile un matcher depuis des motifs. Décode les %xx d'abord (évasions),
// matche insensible à la casse. Retourne nil si aucun motif → signal désactivé (défaut prod).
func ScannerPathMatcher(patterns ...string) func(string) bool {
	if len(patterns) == 0 {
		return nil
	}
	re := regexp.MustCompile(`(?i)(` + strings.Join(patterns, "|") + `)`)
	return func(p string) bool {
		if d, err := url.PathUnescape(p); err == nil {
			p = d // %2eenv → .env, etc.
		}
		return re.MatchString(p)
	}
}

// AllScannerPatterns retourne tous les motifs candidats (pour bâtir le matcher combiné).
func AllScannerPatterns() []string {
	out := make([]string, len(ScannerSignatureDefs))
	for i, s := range ScannerSignatureDefs {
		out[i] = s.Pattern
	}
	return out
}
