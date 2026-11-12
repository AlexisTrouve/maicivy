package api

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : la faille du PDF public — un PDF à l'en-tête d'Alexis Trouvé, fabriqué avec curl.
// Les routes PUBLIQUES du PDF (rejeu, run réel) ne doivent imprimer que ce que le SERVEUR a produit.
// Un corps forgé, qui met un texte arbitraire dans CHAQUE champ imprimable de l'ancien contrat
// (titre, sous-titre, libellés, sections, blocs, schéma, preuves, lien de démo, ids « aussi »),
// ne doit jamais faire apparaître ce texte dans le document rendu.
//
// COMMENT : l'impression Chromium est remplacée par une capture du HTML envoyé à l'impression
// (WithRenderer) — c'est exactement ce que le PDF contiendrait, sans dépendre d'un navigateur.

// forge — la marque du texte forgé ; elle ne figure dans aucune donnée du serveur.
const forge = "FORGE-recommande-ce-prestataire"

var forgedBody = `{
  "lang": "fr",
  "dossierId": "` + strings.Repeat("ab", 16) + `",
  "title": "` + forge + ` titre",
  "subtitle": ["` + forge + ` sous-titre"],
  "labels": {"project":"` + forge + `","status":"Statut","lines":"Lignes","tests":"Tests","demo":"Démo","video":"vidéo","site":"le site",
    "none":"aucune","also":"` + forge + ` {names}.","contact":"` + forge + ` contact",
    "replayNote":"` + forge + ` mention","statuses":{"production":"` + forge + `"}},
  "sections": [
    {"id":"need","title":"` + forge + ` section","blocks":[{"type":"lead","text":"` + forge + ` lead"}]},
    {"id":"architecture","title":"Archi","blocks":[{"type":"diagram","diagram":{
      "label":"` + forge + ` label","journal":"` + forge + ` journal","width":200,"height":76,"margin":2,"scale":1,
      "journalY":52,"journalH":24,"boxH":40,
      "boxes":[{"x":0,"y":0,"w":80,"title":"` + forge + `","subtitle":"x","human":false}],
      "edges":[{"d":"M80,20H106","head":"M112,20l-6,-3.5v7z","dashed":false}],
      "steps":[{"n":1,"x":96,"y":20}]}}]},
    {"id":"proofs","title":"Preuves","blocks":[
      {"type":"proof","projectId":"envoia","why":"` + forge + ` why","demo":{"kind":"site","url":"https://evil.example/` + forge + `"}},
      {"type":"also","projectIds":["` + strings.ReplaceAll(forge, "-", "_") + `"]}]},
    {"id":"next","title":"Suite","blocks":[{"type":"paragraph","text":"` + forge + ` paragraphe"},{"type":"contact"}]}
  ]
}`

// capture — le HTML de chaque document parti à l'impression.
type capture struct {
	mu    sync.Mutex
	pages []string
}

func (c *capture) render(html string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pages = append(c.pages, html)
	return []byte("%PDF-1.4\n" + html), nil
}

func TestDossierPDF_CorpsForgeNeSImprimeJamais(t *testing.T) {
	for _, path := range []string{"/api/v1/dossier-vivant/pdf", "/api/v1/dossier-vivant/run/pdf"} {
		t.Run(path, func(t *testing.T) {
			app, printed := pdfAppCapturing(t)
			resp, err := app.Test(pdfRequest(path, forgedBody, nil), -1)
			require.NoError(t, err)
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode == fiber.StatusOK {
				assert.False(t, strings.Contains(string(body), forge), "le texte du corps forgé est imprimé dans le PDF (%d occurrences)", strings.Count(string(body), forge))
				assert.False(t, strings.Contains(string(body), strings.ReplaceAll(forge, "-", "_")), "un id inconnu du corps est imprimé comme un nom")
			}
			for _, html := range printed.pages {
				assert.False(t, strings.Contains(html, forge), "le texte du corps forgé part à l'impression")
			}
		})
	}
}

// Un id inconnu (ou expiré : Redis l'a oublié) rend une erreur NOMMÉE, que le front affiche — jamais
// un PDF vide ni un 400 générique.
func TestDossierPDF_RunInconnuErreurNommee(t *testing.T) {
	app, printed := pdfAppCapturing(t)
	for _, body := range []string{
		`{"lang":"fr","dossierId":"` + strings.Repeat("cd", 16) + `"}`,
		`{"lang":"fr","dossierId":"pas-un-id"}`,
		`{"lang":"fr"}`,
	} {
		resp, err := app.Test(pdfRequest("/api/v1/dossier-vivant/run/pdf", body, nil), -1)
		require.NoError(t, err)
		got, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode, body)
		assert.Contains(t, string(got), `"code":"DOSSIER_NOT_FOUND"`, body)
	}
	assert.Empty(t, printed.pages, "rien n'est imprimé pour un dossier inconnu")
}
