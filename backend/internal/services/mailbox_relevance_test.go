package services

import (
	"encoding/json"
	"strings"
	"testing"
)

// Régression backtest réel : un bloc <style> volumineux avant le contenu utile ne doit jamais
// repousser la description de mission hors de la troncature (cf. maxRelevanceBodyChars).
func TestStripHTMLNoise_RemovesStyleBlockAndTags(t *testing.T) {
	input := `<style type="text/css">
.heading { font-size: 32px; }
.section { padding-bottom: 20px; }
</style>
<p class="heading">Bonjour,<br/>cette opportunité correspond à votre profil !</p>
<p class="section-details">Mission Go/Kubernetes, 3 mois, Paris.</p>`

	got := StripHTMLNoise(input)

	if strings.Contains(got, "font-size") || strings.Contains(got, ".heading") {
		t.Fatalf("le bloc <style> doit être retiré, got: %q", got)
	}
	if strings.Contains(got, "<p") || strings.Contains(got, "class=") {
		t.Fatalf("les balises HTML doivent être retirées, got: %q", got)
	}
	if !strings.Contains(got, "cette opportunité correspond à votre profil") {
		t.Fatalf("le texte utile doit être préservé, got: %q", got)
	}
	if !strings.Contains(got, "Mission Go/Kubernetes, 3 mois, Paris") {
		t.Fatalf("le texte utile doit être préservé, got: %q", got)
	}
}

// Régression backtest réel : après le strip des balises HTML, l'URL du lien de mission doit rester
// visible dans le texte (un strip naïf de toutes les balises l'effaçait avec le reste, Link revenait
// systématiquement vide).
func TestStripHTMLNoise_PreservesLinkURLs(t *testing.T) {
	input := `<p>Détails de la mission.</p>
<a href="https://www.malt.fr/messages/client-project-offer/abc123" title="Répondre">Répondre au client</a>`

	got := StripHTMLNoise(input)

	if !strings.Contains(got, "https://www.malt.fr/messages/client-project-offer/abc123") {
		t.Fatalf("l'URL du lien doit être préservée, got: %q", got)
	}
	if !strings.Contains(got, "Répondre au client") {
		t.Fatalf("le texte du lien doit être préservé, got: %q", got)
	}
	if strings.Contains(got, "<a ") || strings.Contains(got, "href=") {
		t.Fatalf("la balise <a> elle-même doit être retirée, got: %q", got)
	}
}

func TestNewMailboxRelevanceService_NilWhenUnconfigured(t *testing.T) {
	if svc := NewMailboxRelevanceService("", "key", nil, 50); svc != nil {
		t.Fatal("baseURL vide doit donner un service nil")
	}
	if svc := NewMailboxRelevanceService("https://proxy", "", nil, 50); svc != nil {
		t.Fatal("apiKey vide doit donner un service nil")
	}
}

func TestNewMailboxRelevanceService_ThresholdExposed(t *testing.T) {
	svc := NewMailboxRelevanceService("https://proxy", "key", NewPortfolioService(), 42)
	if svc == nil {
		t.Fatal("service ne doit pas être nil avec credentials présents")
	}
	if got := svc.Threshold(); got != 42 {
		t.Fatalf("Threshold() = %d, attendu 42", got)
	}
}

// Verrouille la présence de tous les tools de l'agent (dont le tool terminal submit_verdict) —
// buildRelevanceTools n'utilise aucun client réseau, testable directement.
func TestBuildRelevanceTools_IncludesAllTools(t *testing.T) {
	tools := buildRelevanceTools()

	names := map[string]bool{}
	for _, tool := range tools {
		if tool.OfTool != nil {
			names[tool.OfTool.Name] = true
		}
	}

	want := []string{"get_profile", "get_experience", "list_skills", "search_projects", "submit_verdict"}
	for _, w := range want {
		if !names[w] {
			t.Fatalf("tool %q manquant dans buildRelevanceTools", w)
		}
	}
}

// search_projects doit exiger `query`, submit_verdict tous ses champs — sinon l'agent pourrait
// appeler ces tools sans les infos nécessaires.
func TestBuildRelevanceTools_RequiredFields(t *testing.T) {
	tools := buildRelevanceTools()
	byName := map[string]anthropicToolSchema{}
	for _, tool := range tools {
		if tool.OfTool != nil {
			byName[tool.OfTool.Name] = anthropicToolSchema{required: tool.OfTool.InputSchema.Required}
		}
	}

	if got := byName["search_projects"].required; len(got) != 1 || got[0] != "query" {
		t.Fatalf("search_projects.required = %v, attendu [query]", got)
	}

	want := map[string]bool{"is_opportunity": true, "score": true, "reason": true, "cot": true, "link": true}
	got := byName["submit_verdict"].required
	if len(got) != len(want) {
		t.Fatalf("submit_verdict.required = %v, attendu 5 champs %v", got, want)
	}
	for _, f := range got {
		if !want[f] {
			t.Fatalf("submit_verdict.required contient un champ inattendu: %q", f)
		}
	}
}

type anthropicToolSchema struct {
	required []string
}

func rawInput(m map[string]interface{}) json.RawMessage {
	b, _ := json.Marshal(m)
	return b
}

func TestParseVerdictToolInput_Basic(t *testing.T) {
	v, err := parseVerdictToolInput(rawInput(map[string]interface{}{
		"is_opportunity": true,
		"score":          85,
		"reason":         "Strong Go/backend match",
		"cot":            "Checked get_experience, found 3 Go backend roles.",
		"link":           "https://malt.fr/mission/123",
	}))
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if !v.IsOpportunity || v.Score != 85 || v.Reason != "Strong Go/backend match" || v.Link != "https://malt.fr/mission/123" {
		t.Fatalf("verdict inattendu: %+v", v)
	}
	if v.CoT == "" {
		t.Fatal("CoT ne doit pas être vide")
	}
}

func TestParseVerdictToolInput_ClampsScore(t *testing.T) {
	v, err := parseVerdictToolInput(rawInput(map[string]interface{}{
		"is_opportunity": true, "score": 150, "reason": "x", "cot": "x", "link": "",
	}))
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if v.Score != 100 {
		t.Fatalf("score doit être clampé à 100, got %d", v.Score)
	}

	v2, err := parseVerdictToolInput(rawInput(map[string]interface{}{
		"is_opportunity": true, "score": -20, "reason": "x", "cot": "x", "link": "",
	}))
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if v2.Score != 0 {
		t.Fatalf("score doit être clampé à 0, got %d", v2.Score)
	}
}

func TestParseVerdictToolInput_NotOpportunity(t *testing.T) {
	v, err := parseVerdictToolInput(rawInput(map[string]interface{}{
		"is_opportunity": false, "score": 0, "reason": "Community newsletter, not a mission", "cot": "No client or mission mentioned.", "link": "",
	}))
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if v.IsOpportunity {
		t.Fatal("attendu is_opportunity=false")
	}
}

func TestParseVerdictToolInput_InvalidInput(t *testing.T) {
	if _, err := parseVerdictToolInput("not a valid input"); err == nil {
		t.Fatal("input invalide doit retourner une erreur")
	}
}
