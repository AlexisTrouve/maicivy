package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : vérifie les contrôles MÉCANIQUES d'un draft (ceux qui ne demandent aucun LLM).
//
// POURQUOI ils existent : l'opérateur de l'écran compose ne lit pas le français. Il ne peut pas juger
// le message qu'il envoie. Une checklist notée par le modèle qui a écrit le message serait une machine
// à fabriquer de la fausse confiance — il répondrait « oui » et l'opérateur n'aurait aucun moyen
// d'attraper l'erreur. Ce qui est vérifiable en code doit donc l'être en code : une case cochée ici est
// une PREUVE, pas une affirmation.
//
// COMMENT : les fixtures sont les VRAIS messages produits en session réelle le 2026-07-30 — dont la
// version que l'opératrice a dû nettoyer à la main parce que l'agent avait rendu du markdown. C'est le
// cas de non-régression central.

// Message réel v1 : produit par le générateur, AVANT la demande d'ajout du remote.
const fixtureDraftSansRemote = `Bonjour,

Vous devez assembler des use cases IA de POC à production sans laisser traîner des briques isolées.

J'ai construit maicivy de zéro : génération IA, connecteurs, interface web, tout déployé en production.

Disponible pour un call cette semaine ?

Alexis`

// Message réel v2 : l'agent a ajouté le remote MAIS a rendu tout le message en citation markdown avec
// du gras. C'est exactement ce que l'opératrice a reçu, et qu'elle a dû nettoyer à la main sans
// comprendre pourquoi ces caractères étaient là.
const fixtureDraftMarkdown = `Bonjour,
>
> Vous devez assembler des use cases IA de POC à production sans laisser traîner des briques isolées.
>
> **Je travaille exclusivement en remote.** Disponible pour un call cette semaine ?
>
> Alexis`

// Message réel v3 : la version effectivement envoyée, nettoyée à la main.
const fixtureDraftPropre = `Bonjour,

Vous devez assembler des use cases IA de POC à production sans laisser traîner des briques isolées.

Je travaille exclusivement en remote. Disponible pour un call cette semaine ?

Alexis`

// byID indexe le résultat pour des assertions lisibles.
func byID(t *testing.T, checks []MechanicalCheck) map[string]MechanicalCheck {
	t.Helper()
	m := map[string]MechanicalCheck{}
	for _, c := range checks {
		m[c.ID] = c
	}
	return m
}

func TestMechanicalChecks_MessageRéelPropre(t *testing.T) {
	got := byID(t, RunMechanicalChecks(ComposeCheckInput{
		Draft: fixtureDraftPropre, Signature: "Alexis", RetainedRate: 300,
	}))

	assert.Equal(t, CheckPass, got[CheckRemote].Status)
	assert.Equal(t, CheckPass, got[CheckNoMarkdown].Status)
	assert.Equal(t, CheckPass, got[CheckSignature].Status)
	assert.Equal(t, CheckPass, got[CheckLength].Status)
	// Le TJM n'est PAS dans le message : chez l'opératrice il va dans un champ du formulaire de la
	// plateforme, pas dans le texte. Exiger sa présence recalerait un bon message.
	assert.Equal(t, CheckNA, got[CheckRateCoherence].Status)
}

// LE cas de non-régression : le markdown que l'opératrice a dû retirer à la main doit être attrapé.
func TestMechanicalChecks_AttrapeLeMarkdownRéel(t *testing.T) {
	got := byID(t, RunMechanicalChecks(ComposeCheckInput{
		Draft: fixtureDraftMarkdown, Signature: "Alexis",
	}))

	require.Equal(t, CheckFail, got[CheckNoMarkdown].Status,
		"le markdown est parti dans un message client sans que l'opératrice puisse le juger")
	assert.NotEmpty(t, got[CheckNoMarkdown].Detail, "l'opérateur doit voir CE qui cloche, pas juste un ✗")
	// La demande d'ajout du remote, elle, a bien été honorée.
	assert.Equal(t, CheckPass, got[CheckRemote].Status)
}

func TestMechanicalChecks_DétecteRemoteAbsent(t *testing.T) {
	got := byID(t, RunMechanicalChecks(ComposeCheckInput{
		Draft: fixtureDraftSansRemote, Signature: "Alexis",
	}))

	assert.Equal(t, CheckFail, got[CheckRemote].Status)
}

// Les formulations acceptables du remote : le modèle n'écrit pas toujours le même mot, et l'opérateur
// ne peut pas arbitrer. Accents et casse ne doivent pas faire échouer le contrôle.
func TestMechanicalChecks_VariantesDuRemote(t *testing.T) {
	for _, variante := range []string{
		"Je travaille exclusivement en remote.",
		"Télétravail uniquement.",
		"TÉLÉTRAVAIL uniquement.",
		"Teletravail uniquement.", // sans accent : le repli d'accents doit l'attraper
		"Mission à distance uniquement.",
		"Je suis en distanciel.",
	} {
		t.Run(variante, func(t *testing.T) {
			got := byID(t, RunMechanicalChecks(ComposeCheckInput{
				Draft: "Bonjour,\n\n" + variante + "\n\nAlexis", Signature: "Alexis",
			}))
			assert.Equal(t, CheckPass, got[CheckRemote].Status)
		})
	}
}

// Le vrai risque sur le TJM n'est pas son absence, c'est l'INCOHÉRENCE : le modèle qui écrit 250 alors
// que le tarif retenu est 300. L'opérateur ne relit pas le français mais il verra la contradiction.
func TestMechanicalChecks_CohérenceDuTarif(t *testing.T) {
	for _, tc := range []struct {
		label    string
		draft    string
		retained int
		want     CheckStatus
	}{
		{"aucun tarif mentionné", "Bonjour,\n\nAlexis", 300, CheckNA},
		{"tarif cohérent", "Mon TJM est de 300€.\n\nAlexis", 300, CheckPass},
		{"tarif contradictoire", "Mon TJM est de 250€.\n\nAlexis", 300, CheckFail},
		{"cohérent écrit en euros", "Mon TJM est de 300 euros.\n\nAlexis", 300, CheckPass},
		// 1300 ne doit pas être lu comme un 300 : sans frontière de mot, le contrôle validerait un
		// montant faux.
		{"sous-chaîne trompeuse", "Budget de 1300€ sur la mission.\n\nAlexis", 300, CheckFail},
		{"aucun tarif retenu", "Mon TJM est de 300€.\n\nAlexis", 0, CheckNA},
	} {
		t.Run(tc.label, func(t *testing.T) {
			got := byID(t, RunMechanicalChecks(ComposeCheckInput{
				Draft: tc.draft, Signature: "Alexis", RetainedRate: tc.retained,
			}))
			assert.Equal(t, tc.want, got[CheckRateCoherence].Status)
		})
	}
}

func TestMechanicalChecks_Signature(t *testing.T) {
	got := byID(t, RunMechanicalChecks(ComposeCheckInput{
		Draft: "Bonjour,\n\nUn message sans signature.", Signature: "Alexis",
	}))

	assert.Equal(t, CheckFail, got[CheckSignature].Status)
}

// Les plateformes plafonnent la longueur des messages : un draft trop long est tronqué à l'envoi, et
// l'opérateur ne s'en rendrait compte qu'après coup.
func TestMechanicalChecks_Longueur(t *testing.T) {
	long := "Bonjour,\n\n"
	for len([]rune(long)) < defaultMaxDraftRunes+50 {
		long += "Une phrase de remplissage pour dépasser la limite. "
	}
	long += "\n\nAlexis"

	got := byID(t, RunMechanicalChecks(ComposeCheckInput{Draft: long, Signature: "Alexis"}))

	assert.Equal(t, CheckFail, got[CheckLength].Status)
	assert.NotEmpty(t, got[CheckLength].Detail, "l'opérateur doit voir de combien il dépasse")
}

// --- Lien vers maicivy -------------------------------------------------------
//
// POURQUOI ce contrôle est SÉPARÉ de « maicivy cité » : citer le nom et donner le lien ne produisent
// pas le même effet. Le nom est une affirmation ; le lien est une preuve qu'on peut ouvrir en un
// clic. C'est le lien qui fait la différence sur une candidature (constat d'Alexi), et l'opératrice
// doit voir lequel des deux manque, pas un verdict global.

func TestCheckMaicivyLink(t *testing.T) {
	cas := []struct {
		nom     string
		draft   string
		attendu CheckStatus
	}{
		{"url complète https", "Voir mon portfolio : https://maicivy.etheryale.com", CheckPass},
		{"url sans schéma", "Tout est là : maicivy.etheryale.com", CheckPass},
		{"url avec chemin", "Le détail : https://maicivy.etheryale.com/fr/cv", CheckPass},
		{"url en majuscules", "MAICIVY.ETHERYALE.COM", CheckPass},
		{"url entre parenthèses", "mon portfolio (maicivy.etheryale.com) le montre", CheckPass},

		// LE cas qui motive ce contrôle : le nom est cité, le lien absent. « maicivy cité » passe,
		// celui-ci doit échouer — sinon l'opératrice croit le message complet alors qu'il manque
		// justement ce qui convertit.
		{"nom cité sans lien", "J'ai construit maicivy, un portfolio agentique.", CheckFail},
		{"rien du tout", "Bonjour, je suis développeur.", CheckFail},
		// Un autre domaine ne compte pas : c'est le lien vers SON site qui fait la preuve.
		{"autre domaine", "Voir https://github.com/AlexisTrouve", CheckFail},
		// Piège : le mot maicivy suivi d'un point et d'une phrase n'est pas une URL.
		{"faux positif ponctuation", "J'ai fait maicivy. Ensuite j'ai fait autre chose.", CheckFail},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			checks := RunMechanicalChecks(ComposeCheckInput{Draft: c.draft})
			var got CheckStatus
			for _, ch := range checks {
				if ch.ID == CheckMaicivyLink {
					got = ch.Status
				}
			}
			assert.Equal(t, c.attendu, got, "draft: %q", c.draft)
		})
	}
}

// Liens vers la PAGE d'un projet (/portfolio/<id>). Constat Malt 2026 : 20 messages sur 25 liaient
// déjà l'accueil du site ; le seul prospect qui a répondu demandait une preuve du projet CITÉ. Le
// contrôle verrouille surtout le pire cas : un lien profond vers une page qui n'existe pas.
func TestCheckProjectLink(t *testing.T) {
	pages := map[string]bool{"melodicode": true, "grove_engine": true}
	cas := []struct {
		nom, draft string
		pages      map[string]bool
		attendu    CheckStatus
		detail     string
	}{
		{"lien valide", "Voir https://maicivy.etheryale.com/fr/portfolio/melodicode pour le détail.", pages, CheckPass, ""},
		{"sans schéma ni locale, ponctuation collée", "(maicivy.etheryale.com/portfolio/grove_engine).", pages, CheckPass, ""},
		{"lien mort", "https://maicivy.etheryale.com/fr/portfolio/projet_invente", pages, CheckFail, "projet_invente"},
		{"un mort suffit à échouer", "https://maicivy.etheryale.com/fr/portfolio/melodicode et https://maicivy.etheryale.com/en/portfolio/fantome", pages, CheckFail, "fantome"},
		{"accueil seul : rien à vérifier", "https://maicivy.etheryale.com", pages, CheckNA, ""},
		{"pages inconnues : invérifiable", "https://maicivy.etheryale.com/fr/portfolio/melodicode", nil, CheckNA, ""},
	}
	for _, c := range cas {
		got := checkProjectLink(c.draft, c.pages)
		assert.Equal(t, c.attendu, got.Status, c.nom)
		assert.Equal(t, c.detail, got.Detail, c.nom)
		assert.Equal(t, CheckProjectLink, got.ID)
	}
	// Branché dans la batterie complète.
	got := byID(t, RunMechanicalChecks(ComposeCheckInput{Draft: "maicivy.etheryale.com/fr/portfolio/x", PortfolioPages: pages}))
	assert.Equal(t, CheckFail, got[CheckProjectLink].Status)
}
