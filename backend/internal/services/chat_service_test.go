package services

import (
	"encoding/json"
	"strings"
	"testing"
)

// Verrouille le contrat de validateLang : le paramètre `language` (obligatoire sur tous les tools
// du chat) doit être servable, sinon une erreur explicite est renvoyée au LLM (listant les langues
// dispo + invitant à répondre en anglais). Régression de la demande "language systématique + erreur".

func TestValidateLang_Supported(t *testing.T) {
	for _, l := range []string{"fr", "en", "de", "it", "zh"} {
		got, err := validateLang(map[string]interface{}{"language": l})
		if err != nil {
			t.Fatalf("validateLang(%q) a renvoyé une erreur: %v", l, err)
		}
		if got != l {
			t.Fatalf("validateLang(%q) = %q, attendu %q", l, got, l)
		}
	}
}

func TestValidateLang_Normalizes(t *testing.T) {
	got, err := validateLang(map[string]interface{}{"language": "  EN "})
	if err != nil || got != "en" {
		t.Fatalf("validateLang('  EN ') = (%q, %v), attendu ('en', nil)", got, err)
	}
}

func TestValidateLang_UnsupportedMentionsListAndEnglish(t *testing.T) {
	for _, l := range []string{"es", "ja", "pt", "ka", "ru"} {
		_, err := validateLang(map[string]interface{}{"language": l})
		if err == nil {
			t.Fatalf("validateLang(%q) attendait une erreur, reçu nil", l)
		}
		msg := err.Error()
		// L'erreur doit lister les langues dispo...
		if !strings.Contains(msg, chatLangList) {
			t.Errorf("erreur pour %q sans la liste %q: %s", l, chatLangList, msg)
		}
		// ...et proposer l'anglais comme repli.
		if !strings.Contains(strings.ToLower(msg), "english") && !strings.Contains(msg, "(en)") {
			t.Errorf("erreur pour %q devrait proposer l'anglais: %s", l, msg)
		}
	}
}

func TestValidateLang_MissingOrInvalid(t *testing.T) {
	cases := []map[string]interface{}{
		{},                // pas de clé language
		{"language": ""},  // vide
		{"language": 123}, // mauvais type
	}
	for _, input := range cases {
		if _, err := validateLang(input); err == nil {
			t.Errorf("validateLang(%v) attendait une erreur, reçu nil", input)
		}
	}
}

// allChatTools liste tous les tools exposés à Claude — sert à vérifier que la garde de langue
// s'applique SYSTÉMATIQUEMENT (aucun tool n'y échappe).
var allChatTools = []string{
	"get_project", "list_projects", "list_skills", "get_experience",
	"show_project", "show_projects", "show_skills", "show_experience",
	"search_projects", "show_blog_article", "show_blog_list", "add_tip",
}

// TI : pour CHAQUE tool, une langue non supportée doit court-circuiter en erreur AVANT tout accès
// aux données. On peut donc utiliser un ChatService aux dépendances nil (portfolio/blog jamais
// touchés) — si la garde laissait passer, on aurait un nil-panic, ce qui ferait échouer le test.
func TestExecuteTool_RejectsUnsupportedLanguage(t *testing.T) {
	cs := &ChatService{} // portfolio/blog nil exprès
	// JSON avec tous les paramètres possibles : peu importe, la validation langue tombe en premier.
	input := json.RawMessage(`{"name":"maicivy","query":"rust","slug":"x","text":"tip","language":"es"}`)
	for _, name := range allChatTools {
		res, err := cs.executeTool(name, input)
		if err == nil {
			t.Errorf("%s: langue 'es' non supportée → attendait une erreur, reçu nil (res=%v)", name, res)
			continue
		}
		if !strings.Contains(err.Error(), chatLangList) {
			t.Errorf("%s: l'erreur devrait lister %q, reçu: %s", name, chatLangList, err.Error())
		}
	}
}

// TI : un `language` manquant est aussi rejeté pour tous les tools (param obligatoire).
func TestExecuteTool_RejectsMissingLanguage(t *testing.T) {
	cs := &ChatService{}
	input := json.RawMessage(`{"name":"maicivy","query":"rust","slug":"x","text":"tip"}`) // pas de language
	for _, name := range allChatTools {
		if _, err := cs.executeTool(name, input); err == nil {
			t.Errorf("%s: language manquant → attendait une erreur, reçu nil", name)
		}
	}
}
