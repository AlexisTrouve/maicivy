package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// masqueCouchesCachees retire la PEINTURE des trois couches cachées sans toucher à la mise en
// page (visibility, pas display) : le PDF « sans » ne diffère du PDF « avec » que par ce que ces
// couches changent à l'œil.
const masqueCouchesCachees = `<style>.stealth,.header-stealth,.ats-anchor{visibility:hidden !important}</style>`

// TestRecettePDF_CheminExport produit les PDF de recette du gabarit CV par le MÊME chemin que
// `GET /cv/export` : renderCVHTML (sélection des projets, strip borné, labels) +
// BuildUniversalStealthText + htmlToPDF. Pour chaque langue : <nom>-<lang>.html, .pdf, et
// -sans-stealth.pdf (couches cachées masquées). tools/cv-recette/mesures.py les mesure ensuite.
//
// POURQUOI un test : renderCVHTML et htmlToPDF sont non exportés, et une recette n'a de valeur
// que si elle rend exactement comme la prod — un rendu parallèle s'écarterait en silence.
// La recette se lance DANS l'image Docker du backend (binaire `go test -c` monté dans le
// conteneur) : headless-shell 123 et les polices de l'image, rien d'autre.
//
// Ignoré par défaut : il ne tourne que si CV_RECETTE_DATA (dossier contenant
// cv_fullstack_{fr,zh}.json, réponses de `GET /api/v1/cv?theme=fullstack&lang=…`) et
// CV_RECETTE_OUT sont posés. CV_RECETTE_TEMPLATES choisit le dossier du gabarit (défaut : celui
// du dépôt) ; CV_RECETTE_NOM préfixe les fichiers (défaut « prod »).
func TestRecettePDF_CheminExport(t *testing.T) {
	dataDir, outDir := os.Getenv("CV_RECETTE_DATA"), os.Getenv("CV_RECETTE_OUT")
	if dataDir == "" || outDir == "" {
		t.Skip("CV_RECETTE_DATA / CV_RECETTE_OUT non posés : recette PDF ignorée")
	}
	gabarit := os.Getenv("CV_RECETTE_TEMPLATES")
	if gabarit == "" {
		gabarit = "../../templates/cv"
	}
	nom := os.Getenv("CV_RECETTE_NOM")
	if nom == "" {
		nom = "prod"
	}
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	svc := newPDFServiceFrom(gabarit)
	require.True(t, svc.hasTemplates, "gabarit %s introuvable", gabarit)
	require.NotEmpty(t, svc.profileImg, "photo de profil absente : le PDF ne serait pas celui de prod")

	for _, lang := range []string{"fr", "zh"} {
		raw, err := os.ReadFile(filepath.Join(dataDir, "cv_fullstack_"+lang+".json"))
		require.NoError(t, err)
		var cv AdaptiveCVResponse
		require.NoError(t, json.Unmarshal(raw, &cv))

		html, err := svc.renderCVHTML(&cv, lang, BuildUniversalStealthText(lang))
		require.NoError(t, err)
		base := filepath.Join(outDir, nom+"-"+lang)
		require.NoError(t, os.WriteFile(base+".html", []byte(html), 0o644))

		pdf, err := htmlToPDF(html)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(base+".pdf", pdf, 0o644))

		sans, err := htmlToPDF(strings.Replace(html, "<head>", "<head>"+masqueCouchesCachees, 1))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(base+"-sans-stealth.pdf", sans, 0o644))
	}
}
