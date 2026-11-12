package services

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/models"
)

// QUOI : vérifie l'étape de VÉRIFICATION d'un draft — traduction pour l'opérateur + checklist.
//
// POURQUOI cette étape est séparée de l'agent : l'opérateur ne lit pas le français. Les deux garanties
// qui lui permettent de faire confiance au message sont (1) une traduction fidèle au TEXTE et (2) une
// checklist qui n'a pas été remplie par l'auteur du message. Si la traduction ou le jugement étaient
// produits dans le contexte de la conversation, le modèle traduirait et jugerait son INTENTION, pas ce
// qu'il a réellement écrit — une faute de rédaction deviendrait invisible exactement là où elle compte.
//
// COMMENT : un faux générateur enregistre chaque prompt reçu, ce qui permet d'affirmer que ni la
// conversation ni le raisonnement de l'agent n'y figurent.

// fakeGenerator capture les prompts et rejoue des réponses scriptées.
type fakeGenerator struct {
	prompts  []string
	models   []string
	response func(prompt string) string
}

func (f *fakeGenerator) GenerateTextWithModel(_ context.Context, prompt, model string) (string, *models.AIMetrics, error) {
	f.prompts = append(f.prompts, prompt)
	f.models = append(f.models, model)
	if f.response != nil {
		return f.response(prompt), &models.AIMetrics{}, nil
	}
	return "", &models.AIMetrics{}, nil
}

const draftÀVérifier = `Bonjour,

Je travaille exclusivement en remote. Disponible pour un call cette semaine ?

Alexis`

// scriptedResponse distingue les appels par un marqueur présent dans chaque prompt.
func scriptedResponse(prompt string) string {
	if strings.Contains(prompt, "TRADUCTION") {
		return "Hello,\n\nI work exclusively remote. Available for a call this week?\n\nAlexis"
	}
	if strings.Contains(prompt, "TITRE") {
		return "Galadrim — AI agents"
	}
	return `[{"id":"terraform","label":"Mentions Terraform experience","status":"fail"}]`
}

func newComposeServiceUnderTest() (*ComposeService, *fakeGenerator) {
	gen := &fakeGenerator{response: scriptedResponse}
	return NewComposeService(gen), gen
}

// La checklist mécanique doit être présente et calculée en local, sans dépendre d'un appel modèle.
func TestComposeVerify_InclutLesChecksMécaniques(t *testing.T) {
	svc, _ := newComposeServiceUnderTest()

	res, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft: draftÀVérifier, Signature: "Alexis", OperatorLang: "en",
	})

	require.NoError(t, err)
	ids := map[string]CheckStatus{}
	for _, c := range res.Mechanical {
		ids[c.ID] = c.Status
	}
	assert.Equal(t, CheckPass, ids[CheckRemote])
	assert.Equal(t, CheckPass, ids[CheckNoMarkdown])
	assert.Equal(t, CheckPass, ids[CheckSignature])
}

// LE point critique : le prompt de traduction ne doit contenir QUE le draft — ni l'historique de
// conversation, ni l'annonce, ni la moindre consigne de rédaction.
func TestComposeVerify_TraductionEnContexteVierge(t *testing.T) {
	svc, gen := newComposeServiceUnderTest()

	_, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft:        draftÀVérifier,
		Signature:    "Alexis",
		OperatorLang: "en",
		Announce:     "SECRET_ANNONCE_NE_DOIT_PAS_FUIR",
		History:      "SECRET_HISTORIQUE_NE_DOIT_PAS_FUIR",
	})
	require.NoError(t, err)

	var translationPrompt string
	for _, p := range gen.prompts {
		if strings.Contains(p, "TRADUCTION") {
			translationPrompt = p
		}
	}
	require.NotEmpty(t, translationPrompt, "aucun appel de traduction émis")

	assert.NotContains(t, translationPrompt, "SECRET_HISTORIQUE_NE_DOIT_PAS_FUIR",
		"l'historique fuit dans la traduction : le modèle traduirait son intention, pas son texte")
	assert.NotContains(t, translationPrompt, "SECRET_ANNONCE_NE_DOIT_PAS_FUIR",
		"l'annonce fuit dans la traduction : elle oriente le sens rendu à l'opérateur")
	assert.Contains(t, translationPrompt, draftÀVérifier)
}

func TestComposeVerify_RenvoieLaTraduction(t *testing.T) {
	svc, _ := newComposeServiceUnderTest()

	res, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft: draftÀVérifier, Signature: "Alexis", OperatorLang: "en",
	})

	require.NoError(t, err)
	assert.Contains(t, res.Translation, "I work exclusively remote")
}

// Le juge sémantique voit l'annonce et le draft — c'est son travail — mais PAS la conversation : sinon
// il note l'intention de l'agent plutôt que le texte produit.
func TestComposeVerify_JugeVoitAnnonceMaisPasHistorique(t *testing.T) {
	svc, gen := newComposeServiceUnderTest()

	res, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft:        draftÀVérifier,
		Signature:    "Alexis",
		OperatorLang: "en",
		Announce:     "Recherche expert Terraform",
		History:      "SECRET_HISTORIQUE_NE_DOIT_PAS_FUIR",
	})
	require.NoError(t, err)

	var judgePrompt string
	for _, p := range gen.prompts {
		if !strings.Contains(p, "TRADUCTION") {
			judgePrompt = p
		}
	}
	require.NotEmpty(t, judgePrompt, "aucun appel de jugement émis")

	assert.Contains(t, judgePrompt, "Recherche expert Terraform")
	assert.NotContains(t, judgePrompt, "SECRET_HISTORIQUE_NE_DOIT_PAS_FUIR")
	require.Len(t, res.Semantic, 1)
	assert.Equal(t, "terraform", res.Semantic[0].ID)
	assert.Equal(t, CheckFail, res.Semantic[0].Status)
}

// Sans annonce il n'y a rien à juger : on n'appelle pas le modèle pour rien, et la section sémantique
// reste vide plutôt que d'inventer des critères.
func TestComposeVerify_SansAnnoncePasDeJuge(t *testing.T) {
	svc, gen := newComposeServiceUnderTest()

	res, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft: draftÀVérifier, Signature: "Alexis", OperatorLang: "en",
	})

	require.NoError(t, err)
	assert.Empty(t, res.Semantic)
	for _, p := range gen.prompts {
		assert.Contains(t, p, "TRADUCTION", "un appel de jugement a été émis sans annonce")
	}
}

// Une traduction ratée ne doit pas faire échouer toute la vérification : la checklist mécanique reste
// la garantie la plus solide, il serait absurde de la perdre parce qu'un appel réseau a échoué.
func TestComposeVerify_TraductionRatéeNeCasseRienDAutre(t *testing.T) {
	gen := &fakeGenerator{response: func(prompt string) string { return "" }}
	svc := NewComposeService(gen)

	res, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft: draftÀVérifier, Signature: "Alexis", OperatorLang: "en",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, res.Mechanical)
	assert.Empty(t, res.Translation)
}

// --- Nommage automatique de l'atelier ---------------------------------------
//
// POURQUOI l'atelier a besoin d'un nom : l'opératrice garde son historique pour retrouver et reprendre
// des conversations. Une liste de lignes datées ne lui dit rien ; il lui faut « de qui ça parle ».
//
// POURQUOI le nom est GÉNÉRÉ et non découpé dans l'annonce : l'annonce est parfois une CAPTURE
// D'ÉCRAN — il n'y a alors aucun texte à découper. Et un titre en français ne l'aiderait pas à
// retrouver quoi que ce soit.

// Le titre n'est produit QUE s'il est demandé : le générer à chaque tour coûterait un appel modèle
// pour rien, et surtout ferait CHANGER le nom sous ses yeux à chaque ajustement du message.
func TestComposeVerify_TitreSeulementSiDemandé(t *testing.T) {
	svc, gen := newComposeServiceUnderTest()

	res, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft: draftÀVérifier, Signature: "Alexis", OperatorLang: "en", Announce: "Mission IA chez Galadrim",
	})

	require.NoError(t, err)
	assert.Empty(t, res.Title, "sans WantTitle, aucun titre ne doit être produit")
	for _, p := range gen.prompts {
		assert.NotContains(t, p, "TITRE", "aucun appel de nommage ne doit partir")
	}
}

func TestComposeVerify_TitreGénéréQuandDemandé(t *testing.T) {
	svc, _ := newComposeServiceUnderTest()

	res, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft: draftÀVérifier, Signature: "Alexis", OperatorLang: "en",
		Announce: "Mission IA chez Galadrim", WantTitle: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "Galadrim — AI agents", res.Title)
}

// Le titre est ce qu'elle LIT pour se repérer : il doit sortir dans SA langue, pas en français.
// Le prompt doit donc porter la langue de l'opératrice.
func TestComposeVerify_TitreDansLaLangueDeLOpératrice(t *testing.T) {
	svc, gen := newComposeServiceUnderTest()

	_, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft: draftÀVérifier, Signature: "Alexis", OperatorLang: "zh",
		Announce: "Mission IA chez Galadrim", WantTitle: true,
	})

	require.NoError(t, err)
	var promptTitre string
	for _, p := range gen.prompts {
		if strings.Contains(p, "TITRE") {
			promptTitre = p
		}
	}
	require.NotEmpty(t, promptTitre, "un appel de nommage doit partir")
	assert.Contains(t, promptTitre, "zh", "le prompt doit porter la langue de l'opératrice")
}

// Un titre à rallonge casse la mise en page de la liste et n'aide personne à se repérer. La coupe
// est faite ICI et pas côté affichage : la valeur est PERSISTÉE, donc c'est elle qu'il faut borner —
// sinon la colonne (varchar 200) déborde et l'écriture échoue.
func TestComposeVerify_TitreBorné(t *testing.T) {
	gen := &fakeGenerator{response: func(prompt string) string {
		if strings.Contains(prompt, "TITRE") {
			return strings.Repeat("Nom interminable ", 40)
		}
		return scriptedResponse(prompt)
	}}
	svc := NewComposeService(gen)

	res, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft: draftÀVérifier, Signature: "Alexis", OperatorLang: "en",
		Announce: "Mission", WantTitle: true,
	})

	require.NoError(t, err)
	assert.LessOrEqual(t, len([]rune(res.Title)), composeTitleMaxRunes)
	assert.NotEmpty(t, res.Title)
}

// Un nommage raté ne doit RIEN casser : la traduction et la checklist sont la raison d'être de cet
// appel, le titre n'est qu'un confort. Même doctrine que la traduction vis-à-vis des checks mécaniques.
func TestComposeVerify_TitreRatéNeCassePasLaVérification(t *testing.T) {
	gen := &fakeGenerator{response: func(prompt string) string {
		if strings.Contains(prompt, "TITRE") {
			return "   " // réponse vide/blanche = échec de nommage
		}
		return scriptedResponse(prompt)
	}}
	svc := NewComposeService(gen)

	res, err := svc.Verify(context.Background(), ComposeVerifyInput{
		Draft: draftÀVérifier, Signature: "Alexis", OperatorLang: "en",
		Announce: "Mission", WantTitle: true,
	})

	require.NoError(t, err)
	assert.Empty(t, res.Title)
	assert.NotEmpty(t, res.Translation, "la traduction doit survivre à un nommage raté")
	assert.NotEmpty(t, res.Mechanical, "la checklist mécanique doit survivre")
}
