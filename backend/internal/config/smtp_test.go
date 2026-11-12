package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Une IP tailscale de relai SMTP a été câblée en dur comme repli dans les trois émetteurs de mail
// jusqu'au 2026-08-30. Le relai s'est réenregistré dans le tailnet (nouvelle IP) et son exim s'est
// arrêté : l'adresse est devenue un trou noir, et les transferts d'offres Malt ont échoué 13 jours
// sans qu'aucun signal ne remonte.
//
// Le verrou ne nomme PAS cette adresse (le dépôt est miroité en public : on n'y publie aucune
// adresse du réseau interne). Il interdit la CLASSE entière : toute adresse de la plage tailscale
// (CGNAT 100.64.0.0/10) écrite en dur dans le source. Une plage CIDR (« 100.64.0.0/10 », utilisée
// par la protection SSRF) n'est pas une adresse et reste permise.
var ipTailnetEnDur = regexp.MustCompile(`\b100\.(6[4-9]|[7-9][0-9]|1[01][0-9]|12[0-7])\.[0-9]{1,3}\.[0-9]{1,3}(/[0-9]+)?`)

// TestSMTPRelayAddr_HoteVideEchoue — un hôte absent doit ARRÊTER l'appelant.
//
// POURQUOI ce test existe : c'est l'invariant exact qui manquait. Tant qu'un repli en dur existait,
// « variable non configurée » et « variable correcte » produisaient le même comportement observable
// au démarrage, et la panne ne pouvait se voir que dans les logs de transfert, 13 jours plus tard.
func TestSMTPRelayAddr_HoteVideEchoue(t *testing.T) {
	t.Setenv("NEWSLETTER_SMTP_HOST", "")
	t.Setenv("NEWSLETTER_SMTP_PORT", "")

	addr, err := SMTPRelayAddr()
	if err == nil {
		t.Fatalf("hôte vide : une erreur était attendue, obtenu addr=%q sans erreur", addr)
	}
	if addr != "" {
		t.Errorf("hôte vide : addr devait être vide, obtenu %q", addr)
	}
	// L'erreur doit nommer la variable à poser, sinon elle n'aide personne à 3h du matin.
	if !strings.Contains(err.Error(), "NEWSLETTER_SMTP_HOST") {
		t.Errorf("l'erreur doit nommer NEWSLETTER_SMTP_HOST, obtenu: %v", err)
	}
}

// TestSMTPRelayAddr_PortParDefaut — le port, lui, garde un défaut : se tromper de port produit un
// refus de connexion immédiat et bruyant, jamais un silence prolongé. Seul l'HÔTE est piégeux.
func TestSMTPRelayAddr_PortParDefaut(t *testing.T) {
	t.Setenv("NEWSLETTER_SMTP_HOST", "100.64.0.25")
	t.Setenv("NEWSLETTER_SMTP_PORT", "")

	addr, err := SMTPRelayAddr()
	if err != nil {
		t.Fatalf("hôte posé : aucune erreur attendue, obtenu %v", err)
	}
	if addr != "100.64.0.25:25" {
		t.Errorf("addr = %q, attendu %q", addr, "100.64.0.25:25")
	}
}

func TestSMTPRelayAddr_PortExplicite(t *testing.T) {
	t.Setenv("NEWSLETTER_SMTP_HOST", "relai.interne")
	t.Setenv("NEWSLETTER_SMTP_PORT", "2525")

	addr, err := SMTPRelayAddr()
	if err != nil {
		t.Fatalf("aucune erreur attendue, obtenu %v", err)
	}
	if addr != "relai.interne:2525" {
		t.Errorf("addr = %q, attendu %q", addr, "relai.interne:2525")
	}
}

// TestAucuneIPDeRelaiEnDur — verrou de régression sur la CLASSE de bug, pas sur un fichier.
//
// COMMENT : on balaie tout le source Go du backend (tests exclus, ce fichier compris) à la recherche
// d'une adresse tailnet en dur. L'IP morte vivait dans trois appelants distincts — cmd/main.go deux fois et
// internal/api/newsletter.go — donc un test ciblant un seul fichier aurait laissé les autres passer.
func TestAucuneIPDeRelaiEnDur(t *testing.T) {
	racine := filepath.Join("..", "..") // internal/config -> backend/

	var coupables []string
	err := filepath.Walk(racine, func(chemin string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// vendor/ n'est pas notre code : une occurrence là-dedans ne serait pas notre dette.
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return nil
		}
		contenu, err := os.ReadFile(chemin)
		if err != nil {
			return err
		}
		// Une correspondance suivie de « /N » est une plage CIDR, pas une adresse : permise.
		for _, m := range ipTailnetEnDur.FindAllString(string(contenu), -1) {
			if !strings.Contains(m, "/") {
				coupables = append(coupables, chemin+" ("+m+")")
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du source: %v", err)
	}

	if len(coupables) > 0 {
		t.Errorf("adresse tailnet en dur trouvée dans %v — le relai se lit dans "+
			"l'environnement via SMTPRelayAddr(), jamais dans le code : un défaut mort est "+
			"indiscernable d'une configuration valide", coupables)
	}
}
