package services

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : dossier_pdf_labels.json est une COPIE des messages next-intl du document (le backend n'a
// pas accès à frontend/ dans son image Docker). Ce test la relit contre frontend/messages : une
// traduction qui change côté front sans la copie fait échouer la CI au lieu d'imprimer l'ancien
// libellé en silence. DOSSIER_LABELS_WRITE=1 régénère la copie.

// dossierStatuses — useDossierPdf.ts › STATUSES : les statuts de fiche que le tableau sait nommer.
var dossierStatuses = []string{"production", "active", "wip", "R&D", "retired"}

func messagesText(t *testing.T, lang string) dossierPDFText {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "frontend", "messages", lang+".json"))
	require.NoError(t, err, "frontend/messages est la source des libellés du PDF")
	var m struct {
		DossierVivant struct {
			Dossier struct {
				Title    string `json:"title"`
				For      string `json:"for"`
				Prepared string `json:"prepared"`
				Contact  string `json:"contact"`
				Proofs   struct {
					Project, Status, Lines, Tests, Demo, Video, Site, None, Also string
				} `json:"proofs"`
				PDF struct {
					ReplayNote string `json:"replayNote"`
				} `json:"pdf"`
				Skills struct {
					Title, Lead, Skill, Level, Source, Need, Technologies string
					Levels                                                map[string]string
				} `json:"skills"`
			} `json:"dossier"`
		} `json:"dossierVivant"`
		Montagne struct {
			Status map[string]string `json:"status"`
		} `json:"montagne"`
	}
	require.NoError(t, json.Unmarshal(raw, &m))
	d := m.DossierVivant.Dossier
	statuses := map[string]string{}
	for _, s := range dossierStatuses {
		if v, ok := m.Montagne.Status[s]; ok {
			statuses[s] = v
		}
	}
	levels := map[string]string{}
	for _, l := range []string{"expert", "advanced", "intermediate", "beginner"} {
		levels[l] = d.Skills.Levels[l]
	}
	return dossierPDFText{Title: d.Title, For: d.For, Prepared: d.Prepared, Labels: DossierPDFLabels{
		Project: d.Proofs.Project, Status: d.Proofs.Status, Lines: d.Proofs.Lines, Tests: d.Proofs.Tests,
		Demo: d.Proofs.Demo, Video: d.Proofs.Video, Site: d.Proofs.Site, None: d.Proofs.None, Also: d.Proofs.Also,
		Contact: d.Contact, ReplayNote: d.PDF.ReplayNote, Statuses: statuses,
		Skills: DossierPDFSkillLabels{Title: d.Skills.Title, Lead: d.Skills.Lead, Skill: d.Skills.Skill, Level: d.Skills.Level,
			Source: d.Skills.Source, Need: d.Skills.Need, Technologies: d.Skills.Technologies, Levels: levels},
	}}
}

func TestDossierPDFLabels_CopieDesMessages(t *testing.T) {
	want := map[string]dossierPDFText{}
	for lang := range dossierLangs {
		want[lang] = messagesText(t, lang)
		// fillMessage ne connaît que les arguments simples : une apostrophe ASCII (échappement ICU)
		// ou un pluriel feraient diverger le PDF de l'écran.
		for _, msg := range []string{want[lang].For, want[lang].Prepared, want[lang].Labels.Also} {
			assert.NotContains(t, msg, "'", "%s : apostrophe ICU dans %q", lang, msg)
			assert.NotContains(t, msg, ", plural", "%s : pluriel ICU dans %q", lang, msg)
		}
	}
	if os.Getenv("DOSSIER_LABELS_WRITE") == "1" {
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		enc.SetEscapeHTML(false) // « R&D » lisible dans la copie, pas « R\u0026D »
		enc.SetIndent("", " ")
		require.NoError(t, enc.Encode(want))
		require.NoError(t, os.WriteFile("dossier_pdf_labels.json", out.Bytes(), 0o644))
	}
	assert.Equal(t, want, dossierPDFTexts, "dossier_pdf_labels.json a divergé de frontend/messages (DOSSIER_LABELS_WRITE=1 pour la régénérer)")
}

func TestFillMessage_UnePasse(t *testing.T) {
	got := fillMessage("Pour {name}, {affiliation}", map[string]string{"name": "{affiliation}", "affiliation": "CTO"})
	assert.Equal(t, "Pour {affiliation}, CTO", got)
	assert.True(t, strings.HasPrefix(formatCost(1234.5678, "fr"), "1\u202f234,568"))
	assert.Equal(t, "0.022", formatCost(0.022, "en"))
	assert.Equal(t, "0,022", formatCost(0.022, "de"))
	assert.Equal(t, "1:14", formatElapsed(74.9))
}
