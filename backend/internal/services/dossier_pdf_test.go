package services

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests du PDF du dossier vivant : contrat de la requête (bornes, tracés, langues), faits relus
// par le serveur (jamais pris dans la requête), mise en page (regroupements, contact, mention du
// rejeu), et l'exclusion mutuelle du moteur. Le rendu Chromium lui-même est couvert par le test de
// l'endpoint (api/dossier_vivant_pdf_test.go), la pagination mesurée par dossier_pdf_layout_test.go.

type fakeDossierFacts struct{}

func (fakeDossierFacts) Montagne(context.Context, string) (*Montagne, error) {
	return &Montagne{Nodes: []MontagneNoeud{
		{ID: "envoia", Name: "EnvoIA", Monogram: "EI"},
		{ID: "aria", Name: "Aria", Monogram: "Ar"},
		{ID: "momos", Name: "Momos", Monogram: "Mo"},
	}}, nil
}

func (fakeDossierFacts) ListProjects(context.Context, string) ([]MPFProject, error) {
	return []MPFProject{
		{ID: "envoia", Status: "production", Stats: MPFStats{LOC: 8971, Tests: 246}, Stack: []string{"python", "sqlite"}, Tags: []string{"AI / LLM Integration"}},
		{ID: "aria", Status: "wip", Stats: MPFStats{LOC: 28346, Tests: 0}, Stack: []string{"flutter", "sqlite"}, Tags: []string{"AI / LLM Integration"}},
	}, nil
}

// Skills — comme maiProFiles : ce nom-là est le même en français (canonique) et ailleurs.
func (fakeDossierFacts) Skills(context.Context, string) ([]MPFSkill, error) {
	return []MPFSkill{{Name: "AI / LLM Integration", Level: "expert", Years: 4}, {Name: "Rust", Level: "intermediate", Years: 1}}, nil
}

// dossierFixture — la forme d'un vrai dossier remis (run réel du 2026-10, raccourci).
func dossierFixture() *DossierPDFRequest {
	return &DossierPDFRequest{
		Lang:     "fr",
		Title:    "Dossier de conception",
		Subtitle: []string{"Pour Claire Dubois, associée chez Cabinet Lumen.", "Préparé en 1:20 par cinq agents, pour 0,009 $."},
		Labels: DossierPDFLabels{
			Project: "Projet", Status: "Statut", Lines: "Lignes", Tests: "Tests", Demo: "Démo",
			Video: "vidéo", Site: "le site", None: "aucune publique", Also: "Aussi retenus : {names}.",
			Contact: "Écrire à Alexis", ReplayNote: "Exemple rejoué : un scénario enregistré, sans appel IA.",
			Statuses: map[string]string{"production": "En production", "wip": "En cours"},
			Skills: DossierPDFSkillLabels{
				Title: "Compétences mobilisées", Lead: "Associées automatiquement aux preuves.", Skill: "Compétence",
				Level: "Niveau", Source: "Vue dans", Need: "citée dans le besoin", Technologies: "Technologies communes",
				Levels: map[string]string{"expert": "expert", "advanced": "avancé"},
			},
		},
		Sections: []DossierPDFSection{
			{ID: "need", Title: "Le besoin, reformulé", Blocks: []DossierPDFBlock{
				{Type: "lead", Text: "Vos agents liront les pièces <b>sous validation</b> humaine."},
				{Type: "item", Text: "Aucune écriture sans validation."},
				{Type: "item", Text: "Chaque pièce rangée dans le bon dossier."},
			}},
			{ID: "architecture", Title: "L’architecture proposée", Blocks: []DossierPDFBlock{
				{Type: "diagram", Diagram: &DossierPDFDiagram{
					Label: "Pièce reçue, lue, contrôlée, validée.", Journal: "Lecture, contrôle, validation humaine",
					Width: 742, Height: 112, Margin: 2, Scale: 0.9, JournalY: 88, JournalH: 24, BoxH: 40,
					Boxes: []DossierPDFDiagramBox{
						{X: 0, Y: 0, W: 105, Title: "Réception", Subtitle: "Pièce déposée"},
						{X: 137, Y: 0, W: 137, Title: "Extraction LLM", Subtitle: "LLM lit la pièce"},
						{X: 306, Y: 52, W: 124, Title: "Anomalie", Subtitle: "Traitement manuel", Human: true},
					},
					Edges: []DossierPDFDiagramArc{
						{D: "M105,20H131", Head: "M137,20l-6,-3.5v7z"},
						{D: "M274,26C290,26 290,72 300,72", Head: "M306,72l-6,-3.5v7z", Dashed: true},
					},
					Steps: []DossierPDFDiagramDot{{N: 1, X: 121, Y: 20}},
				}},
			}},
			{ID: "proofs", Title: "Les projets qui le prouvent", Blocks: []DossierPDFBlock{
				{Type: "proof", ProjectID: "envoia", Why: "Pièces traitées comme contenu non fiable.", Demo: &DossierPDFDemo{Kind: "site", URL: "https://example.org/envoia"}},
				{Type: "proof", ProjectID: "aria", Why: "Lecture de PDF et d'images."},
				{Type: "also", ProjectIDs: []string{"momos", "inconnu"}},
			}},
			{ID: "contribution", Title: "Ce qu’Alexis apporte", Blocks: nil},
			{ID: "next", Title: "Prochaine étape", Blocks: []DossierPDFBlock{{Type: "paragraph", Text: "Un court échange."}}},
		},
	}
}

func renderFixture(t *testing.T, mutate func(*DossierPDFRequest), replay bool) (string, error) {
	t.Helper()
	req := dossierFixture()
	if mutate != nil {
		mutate(req)
	}
	return NewDossierPDFService(fakeDossierFacts{}).RenderHTML(context.Background(), req, replay)
}

func TestDossierPDF_ChiffresRelusSurLesFichesJamaisDansLaRequete(t *testing.T) {
	html, err := renderFixture(t, nil, false)
	require.NoError(t, err)
	// Séparateur français (espace fine insécable), statut traduit, zéro = non mesuré.
	assert.Contains(t, html, "8\u202f971")
	assert.Contains(t, html, "246")
	assert.Contains(t, html, "28\u202f346")
	assert.Contains(t, html, "En production")
	assert.Contains(t, html, "En cours")
	// Noms et monogrammes : la montagne ; un id inconnu reste lisible tel quel.
	assert.Contains(t, html, ">EI<")
	assert.Contains(t, html, "Aussi retenus : Momos, inconnu.")
}

func TestDossierPDF_TexteEchappeEtContactDuServeur(t *testing.T) {
	html, err := renderFixture(t, nil, false)
	require.NoError(t, err)
	assert.NotContains(t, html, "<b>sous validation</b>", "le texte des agents est échappé")
	assert.Contains(t, html, "&lt;b&gt;sous validation&lt;/b&gt;")
	// Le contact est celui du serveur, et figure même si les agents n'ont pas posé de bloc contact.
	assert.Contains(t, html, `href="mailto:`+dossierEmail+`"`)
	assert.Equal(t, 1, strings.Count(html, `class="u contact"`))
}

func TestDossierPDF_RegroupementsEtSectionsVides(t *testing.T) {
	html, err := renderFixture(t, nil, false)
	require.NoError(t, err)
	// Une unité de pagination par puce et par ligne ; un seul en-tête par tableau (répété par le
	// script de pagination quand le tableau change de page).
	assert.Equal(t, 2, strings.Count(html, `class="u li"`), "une unité par puce")
	assert.Equal(t, 1, strings.Count(html, `data-thead="proofs"`), "deux preuves consécutives = un tableau, un en-tête")
	assert.Equal(t, 2, strings.Count(html, `data-row="proofs"`), "une ligne par preuve")
	assert.NotContains(t, html, `data-section="contribution"`, "section sans bloc : pas de titre orphelin")
	assert.Contains(t, html, `viewBox="-2 -2 746 116"`)
	assert.Contains(t, html, `width="671"`)
	assert.Contains(t, html, `class="dg-edge dashed"`)
}

func TestDossierPDF_SectionDesCompetencesApresLesPreuves(t *testing.T) {
	html, err := renderFixture(t, nil, false)
	require.NoError(t, err)
	proofs := strings.Index(html, `data-section="proofs"`)
	skills := strings.Index(html, `data-section="skills"`)
	next := strings.Index(html, `data-section="next"`)
	require.Positive(t, skills)
	assert.True(t, proofs < skills && skills < next, "les compétences suivent les preuves dont elles viennent")
	// Nom LOCALISÉ (fr ici = canonique), niveau traduit, preuves nommées par la montagne.
	assert.Contains(t, html, "<b>AI / LLM Integration</b>")
	assert.Contains(t, html, "EnvoIA, Aria")
	// sqlite est dans la pile des deux preuves : technologie commune ; python n'en a qu'une.
	assert.Contains(t, html, "<b>sqlite</b>")
	assert.NotContains(t, html, "<b>python</b>")

	// Sans libellés de section (ancien client) : pas de section, pas de titre vide.
	html, err = renderFixture(t, func(r *DossierPDFRequest) { r.Labels.Skills = DossierPDFSkillLabels{} }, false)
	require.NoError(t, err)
	assert.NotContains(t, html, `data-section="skills"`)
}

func TestDossierPDF_MentionDuRejeu(t *testing.T) {
	owner, err := renderFixture(t, nil, false)
	require.NoError(t, err)
	assert.NotContains(t, owner, "Exemple rejoué")

	replay, err := renderFixture(t, nil, true)
	require.NoError(t, err)
	assert.Contains(t, replay, "Exemple rejoué")

	_, err = renderFixture(t, func(r *DossierPDFRequest) { r.Labels.ReplayNote = " " }, true)
	assert.ErrorIs(t, err, ErrDossierPDFInvalid, "la route publique n'imprime pas sans sa mention")
}

func TestDossierPDF_RefuseLesRequetesHorsContrat(t *testing.T) {
	cases := map[string]func(*DossierPDFRequest){
		"langue inconnue":       func(r *DossierPDFRequest) { r.Lang = "xx" },
		"titre vide":            func(r *DossierPDFRequest) { r.Title = " " },
		"section inconnue":      func(r *DossierPDFRequest) { r.Sections[0].ID = "evil" },
		"section en double":     func(r *DossierPDFRequest) { r.Sections[1].ID = "need" },
		"bloc inconnu":          func(r *DossierPDFRequest) { r.Sections[0].Blocks[0].Type = "script" },
		"texte trop long":       func(r *DossierPDFRequest) { r.Sections[0].Blocks[0].Text = strings.Repeat("é", dossierMaxText+1) },
		"tracé hors grammaire":  func(r *DossierPDFRequest) { r.Sections[1].Blocks[0].Diagram.Edges[0].D = `M0,0"/><script>` },
		"coordonnée démesurée":  func(r *DossierPDFRequest) { r.Sections[1].Blocks[0].Diagram.Boxes[0].X = 1e9 },
		"échelle > 1":           func(r *DossierPDFRequest) { r.Sections[1].Blocks[0].Diagram.Scale = 3 },
		"lien javascript":       func(r *DossierPDFRequest) { r.Sections[2].Blocks[0].Demo.URL = "javascript:alert(1)" },
		"id de projet libre":    func(r *DossierPDFRequest) { r.Sections[2].Blocks[0].ProjectID = "../../etc" },
		"also sans {names}":     func(r *DossierPDFRequest) { r.Labels.Also = "Aussi retenus." },
		"niveau hors grammaire": func(r *DossierPDFRequest) { r.Labels.Skills.Levels["<x>"] = "x" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := renderFixture(t, mutate, false)
			assert.ErrorIs(t, err, ErrDossierPDFInvalid)
		})
	}
}

func TestDossierPDF_ChinoisCompteEnCaracteres(t *testing.T) {
	// 2 000 caractères chinois = 6 000 octets : la borne porte sur les caractères.
	_, err := renderFixture(t, func(r *DossierPDFRequest) {
		r.Lang = "zh"
		r.Sections[0].Blocks[0].Text = strings.Repeat("档", dossierMaxText)
	}, false)
	assert.NoError(t, err)
}

func TestDossierPDF_GroupDigits(t *testing.T) {
	assert.Equal(t, "303\u202f979", groupDigits(303979, "fr"))
	assert.Equal(t, "1,415", groupDigits(1415, "en"))
	assert.Equal(t, "93.586", groupDigits(93586, "de"))
	assert.Equal(t, "600", groupDigits(600, "zh"))
	assert.Equal(t, "1,234,567", groupDigits(1234567, "zh"))
}

func TestDossierPDF_UnSeulChromiumALaFois(t *testing.T) {
	svc := NewDossierPDFService(fakeDossierFacts{})
	running := make(chan struct{})
	release := make(chan struct{})
	svc.render = func(string) ([]byte, error) {
		running <- struct{}{}
		<-release
		return []byte("%PDF-1.4"), nil
	}
	done := make(chan error, 1)
	go func() {
		_, err := svc.Render(context.Background(), dossierFixture(), false)
		done <- err
	}()
	<-running
	// Le second attend son tour ; son contexte expire avant : « occupé », pas un second Chromium.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.Render(ctx, dossierFixture(), false)
	assert.ErrorIs(t, err, ErrDossierPDFBusy)
	close(release)
	require.NoError(t, <-done)
}

// fakeIcons — l'icône de dépôt de quelques projets (cf. ProjectIconReader).
type fakeIcons map[string][]byte

func (f fakeIcons) Icon(_ context.Context, id string) ([]byte, string, bool) {
	d, ok := f[id]
	return d, "v1", ok
}

// Un projet dont le dépôt a une icône la porte dans le PDF, EMBARQUÉE (data: URI : le Chromium qui
// imprime ne lit ni l'API ni Gitea) et dans la même boîte que le monogramme ; les autres gardent
// leur monogramme ; sans service d'icônes, rien ne change.
func TestDossierPDF_IconeDuDepotSinonMonogramme(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\nfaux")
	svc := NewDossierPDFService(fakeDossierFacts{}).WithIcons(fakeIcons{"envoia": png})
	html, err := svc.RenderHTML(context.Background(), dossierFixture(), false)
	require.NoError(t, err)
	assert.Contains(t, html, `<img class="icon" src="data:image/png;base64,iVBORw0KGgpmYXV4" alt="">`)
	assert.NotContains(t, html, `<span class="mono serif">EI</span>`, "l'icône remplace le monogramme")
	assert.Contains(t, html, `<span class="mono serif">Ar</span>`, "sans icône : le monogramme")

	sans, err := renderFixture(t, nil, false)
	require.NoError(t, err)
	assert.Contains(t, sans, `<span class="mono serif">EI</span>`)
	assert.NotContains(t, sans, `class="icon"`)
}
