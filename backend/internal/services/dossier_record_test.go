package services

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : la copie serveur d'un dossier imprime EXACTEMENT ce que le front imprimait.
//   - mise en page du schéma : le port Go rend la géométrie de pipeline.ts au nombre près, sur la
//     golden écrite par le test jest serverCopies.test.ts ;
//   - rejeu Tessera : pour chaque langue, la requête construite par le serveur est égale, champ pour
//     champ, à celle que construisait le front de 6f78f6c (testdata/dossier_pdf_avant/, produites par
//     pdfExport.ts › buildDossierPdfRequest et les vrais messages next-intl) — et le document HTML
//     imprimé est identique octet pour octet.
// Le run réel est couvert du côté de l'émetteur (dossiervivant/record_test.go), sur le flux brut
// d'un vrai run.

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, v))
}

func TestPrintDiagram_GoldenDuFront(t *testing.T) {
	var cases []struct {
		Name    string                `json:"name"`
		Nodes   []DossierPipelineNode `json:"nodes"`
		Edges   []DossierPipelineEdge `json:"edges"`
		Journal string                `json:"journal"`
		Label   string                `json:"label"`
		Diagram DossierPDFDiagram     `json:"diagram"`
	}
	readJSON(t, filepath.Join("testdata", "dossier_print_diagrams.json"), &cases)
	require.GreaterOrEqual(t, len(cases), 5)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			assert.Equal(t, &c.Diagram, printDiagram(c.Nodes, c.Edges, c.Journal, c.Label))
		})
	}
}

// avantFacts — les faits d'une machine de test : la montagne porte la vidéo de MelodiCode que les
// requêtes « avant » ont reçue par le front (data.ts › replayContext).
type avantFacts struct{ countingFactsStub }

type countingFactsStub struct{}

func (countingFactsStub) Montagne(context.Context, string) (*Montagne, error) {
	video := MontagnePreuve{MontagneVisuel: MontagneVisuel{Kind: "video", URL: "/portfolio-images/melodicode-demo.mp4"}}
	return &Montagne{
		Nodes:  []MontagneNoeud{{ID: "melodicode", Name: "MelodiCode", Monogram: "MC"}, {ID: "envoia", Name: "EnvoIA", Monogram: "EI"}},
		Summit: []MontagneCarte{{ID: "melodicode", Proofs: []MontagnePreuve{video}}},
	}, nil
}

func (countingFactsStub) ListProjects(context.Context, string) ([]MPFProject, error) {
	return []MPFProject{{ID: "envoia", Status: "production", Stats: MPFStats{LOC: 8971, Tests: 246}, Tags: []string{"Automatisation"}}}, nil
}

func (countingFactsStub) Skills(context.Context, string) ([]MPFSkill, error) {
	return []MPFSkill{{Name: "Automatisation", Level: "advanced"}}, nil
}

func TestDossierRecord_MemeRenduQuAvant(t *testing.T) {
	svc := NewDossierPDFService(avantFacts{})
	for lang := range dossierLangs {
		t.Run("tessera-"+lang, func(t *testing.T) {
			var before DossierPDFRequest
			readJSON(t, filepath.Join("testdata", "dossier_pdf_avant", "tessera-"+lang+".json"), &before)
			got, err := svc.ReplayRequest(context.Background(), lang)
			require.NoError(t, err)
			assert.Equal(t, &before, got, "la requête du serveur doit être celle que le front envoyait")

			wantHTML, err := svc.RenderHTML(context.Background(), &before, true)
			require.NoError(t, err)
			gotHTML, err := svc.RenderHTML(context.Background(), got, true)
			require.NoError(t, err)
			assert.True(t, wantHTML == gotHTML, "document imprimé différent de celui d'avant")
			assert.Contains(t, gotHTML, dossierPDFTexts[lang].Labels.ReplayNote, "le rejeu porte sa mention")
		})
	}
}

// Sans vidéo de MelodiCode sur la montagne, sa preuve n'a pas de lien (comme à l'écran) ; le lien du
// site suit la langue.
func TestTesseraRecord_JetonsDuRendu(t *testing.T) {
	rec := tesseraRecord("de", &Montagne{})
	var demos []string
	for _, s := range rec.Sections {
		for _, b := range s.Blocks {
			if b.Demo != nil {
				demos = append(demos, b.Demo.URL)
			}
			assert.False(t, b.Type == "proof" && b.ProjectID == "melodicode" && b.Demo != nil)
		}
	}
	assert.Equal(t, []string{"/de"}, demos)
	assert.False(t, strings.Contains(string(dossierTesseraSrc), "/de\""), "la copie embarquée n'est pas modifiée en place")
}

func TestDossierRecord_LangueHorsListe(t *testing.T) {
	_, err := tesseraRecord("fr", &Montagne{}).PDFRequest("xx")
	assert.ErrorIs(t, err, ErrDossierPDFInvalid)
	_, err = NewDossierPDFService(avantFacts{}).ReplayRequest(context.Background(), "")
	assert.ErrorIs(t, err, ErrDossierPDFInvalid)
}
