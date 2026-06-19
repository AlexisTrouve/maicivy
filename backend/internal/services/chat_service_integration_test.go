//go:build integration

// Tests d'intégration du chat : exécution réelle des tools contre l'API maiProFiles live
// (https://maiprofiles.etheryale.com). Réseau requis → exclus du `go test` par défaut.
//
// Lancer : go test -tags=integration ./internal/services/ -run Integration -v
//
// POURQUOI : la garde de langue (validateLang) est déjà couverte en unitaire ; ici on vérifie la
// CHAÎNE COMPLÈTE — la langue validée est bien propagée jusqu'à maiProFiles et le contenu revient
// effectivement dans cette langue (corrige le bug "fiches toujours en FR"), et la langue non
// supportée échoue end-to-end.
package services

import (
	"encoding/json"
	"strings"
	"testing"
)

func raw(s string) json.RawMessage { return json.RawMessage(s) }

// La langue passée au tool doit changer le contenu renvoyé par maiProFiles (en ≠ fr).
func TestExecuteTool_Integration_LangThreading(t *testing.T) {
	cs := &ChatService{portfolio: NewPortfolioService()}

	resEn, err := cs.executeTool("show_project", raw(`{"name":"maicivy","language":"en"}`))
	if err != nil {
		t.Fatalf("show_project en: erreur inattendue: %v", err)
	}
	resFr, err := cs.executeTool("show_project", raw(`{"name":"maicivy","language":"fr"}`))
	if err != nil {
		t.Fatalf("show_project fr: erreur inattendue: %v", err)
	}

	en, ok1 := resEn.(PortfolioEntry)
	fr, ok2 := resFr.(PortfolioEntry)
	if !ok1 || !ok2 {
		t.Fatalf("résultat inattendu (types: %T / %T)", resEn, resFr)
	}
	if en.ShortDesc == "" || fr.ShortDesc == "" {
		t.Fatalf("ShortDesc vide (en=%q fr=%q) — projet 'maicivy' introuvable ?", en.ShortDesc, fr.ShortDesc)
	}
	// La preuve que la langue est bien threadée : le contenu diffère entre en et fr.
	if en.ShortDesc == fr.ShortDesc {
		t.Errorf("ShortDesc identique en/fr → la langue n'est pas propagée: %q", en.ShortDesc)
	}
}

// get_experience doit aussi être localisé (profil/bio dans la bonne langue).
func TestExecuteTool_Integration_ExperienceLocalized(t *testing.T) {
	cs := &ChatService{portfolio: NewPortfolioService()}
	resEn, err := cs.executeTool("get_experience", raw(`{"language":"en"}`))
	if err != nil {
		t.Fatalf("get_experience en: %v", err)
	}
	if _, ok := resEn.(ExperienceData); !ok {
		t.Fatalf("get_experience: type inattendu %T", resEn)
	}
}

// Langue non supportée → erreur end-to-end, sans appel à maiProFiles.
func TestExecuteTool_Integration_UnsupportedLanguage(t *testing.T) {
	cs := &ChatService{portfolio: NewPortfolioService()}
	_, err := cs.executeTool("show_project", raw(`{"name":"maicivy","language":"es"}`))
	if err == nil {
		t.Fatal("langue 'es' → attendait une erreur")
	}
	if !strings.Contains(err.Error(), chatLangList) || !strings.Contains(strings.ToLower(err.Error()), "english") {
		t.Errorf("erreur devrait lister les langues + proposer l'anglais: %s", err.Error())
	}
}
