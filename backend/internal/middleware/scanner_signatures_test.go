package middleware

import "testing"

// Verrouille les signatures de scanners de services exposés (n8n / Langflow / Docker API) ajoutées
// suite au scanner 103.153.183.92. Les chemins du scanner DOIVENT matcher ; les vrais chemins maicivy
// et les assets Next ne doivent JAMAIS matcher (garde anti-faux-positif au plus près du code).
func TestScannerSignatures_ToolScanners(t *testing.T) {
	m := ScannerPathMatcher(AllScannerPatterns()...)

	// Chemins réellement sprayés par le scanner (n8n /workflows /executions /rest, Langflow /flows
	// /components, Docker /docker/api), y compris préfixés par la locale (matching en substring).
	mustMatch := []string{
		"/api/v1/workflows",
		"/api/v1/executions",
		"/rest/workflows",
		"/fr/rest/workflows",
		"/api/v1/flows/execute",
		"/api/v1/components",
		"/docker/api",
		"/fr/docker/api",
	}
	for _, p := range mustMatch {
		if !m(p) {
			t.Errorf("devrait matcher (scanner): %q", p)
		}
	}

	// Vrais chemins maicivy + assets Next + slugs blog piégeux : AUCUN ne doit matcher. Couvre les
	// collisions qu'on a écartées par construction : "docker-api" (tiret ≠ slash), un chunk Next
	// "components-x.js", un slug "webhook-testing" (raison pour laquelle webhook-test n'est PAS signé),
	// "rest-api" (tiret ≠ /rest/), "gitstats" (≠ \.git).
	mustNotMatch := []string{
		"/api/v1/letters/generate",
		"/api/v1/cv/export",
		"/api/v1/blog/posts",
		"/api/v1/analytics/stats",
		"/api/v1/gitstats",
		"/fr/blog/docker-api-guide",
		"/_next/static/chunks/components-a1b2c3.js",
		"/fr/blog/webhook-testing-en-2026",
		"/fr/blog/rest-api-best-practices",
		"/fr",
		"/",
	}
	for _, p := range mustNotMatch {
		if m(p) {
			t.Errorf("ne devrait PAS matcher (légit): %q", p)
		}
	}
}
