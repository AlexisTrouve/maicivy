package dossiervivant

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : l'invariant des textes affichés (fit.go) — aucun texte visible du flux (blocs du dossier,
// donc le PDF ; notes de colonne, verdicts, messages du fil) ne finit coupé en plein mot ni sur une
// ellipse, et aucun ne dépasse sa borne d'affichage. Sur le chemin de PRODUCTION : vrai AIService,
// fausse passerelle qui répond comme claude-haiku-5-5 en effort bas — des textes plus longs que
// demandé (mesuré : raisons de preuve demandées à 90 caractères, rendues à 115–130).

// longText — une phrase réaliste, répétée jusqu'à `n` runes au moins : des mots, des virgules,
// des points-virgules, une fin de phrase — les frontières qu'une coupe propre doit trouver.
func longText(n int) string {
	const s = "Des agents en production relisent chaque devis, un humain valide l'envoi ; chaque décision reste tracée dans le journal du bureau. "
	return strings.TrimSpace(strings.Repeat(s, n/utf8.RuneCountInString(s)+1))
}

// longAnswers — chaque tâche rend ses textes affichés AU-DELÀ de leur borne.
func longAnswers() map[string]any {
	l := longText
	schema := map[string]any{}
	for k, v := range cannedSchema {
		schema[k] = v
	}
	nodes := []map[string]any{}
	for _, n := range cannedSchema["nodes"].([]map[string]any) {
		c := map[string]any{}
		for k, v := range n {
			c[k] = v
		}
		c["title"], c["subtitle"] = "Contrôle des anomalies", "règles du cabinet appliquées"
		nodes = append(nodes, c)
	}
	schema["nodes"], schema["notes"] = nodes, []string{l(100), l(100)}
	schema["journal"], schema["label"], schema["summary"] = l(70), l(240), l(100)
	return map[string]any{
		"plan":    map[string]any{"plan": l(120), "tasks": map[string]string{"reader": l(100), "researcher": l(100), "architect": l(100), "writer": l(100)}},
		"reader":  map[string]any{"notes": []string{l(100), l(100), l(100)}, "gaps": []string{l(100)}, "summary": l(100)},
		"verdict": map[string]any{"verdict": "keep", "reason": l(90), "why": l(140), "score": 80},
		"lesson":  map[string]any{"useful": true, "lesson": l(90)},
		"design": map[string]any{"stages": []string{l(100), l(100), l(100)}, "branches": []string{l(120)},
			"guardrails": []string{l(100), l(100), l(100)}, "check": l(100), "summary": l(100)},
		"schema":       schema,
		"section-need": map[string]any{"lead": l(200), "items": []string{l(140), l(140), l(140)}},
		"closing":      map[string]any{"contribution": []string{l(140), l(140), l(140)}, "next": l(320)},
	}
}

// shown — un texte visible du flux, et la borne de la place qu'il occupe.
type shown struct {
	where, text string
	max         int
}

// visibleTexts — tous les textes affichés d'un run, chacun avec sa borne. La citation du besoin par
// le visiteur est à part (extrait voulu, cf. TestQuote_ExtraitAUneFrontiere).
func visibleTexts(events []Event) []shown {
	var out []shown
	for _, e := range events {
		switch ev := e.(type) {
		case DossierBlock:
			b := ev.Block
			switch b.Type {
			case "lead":
				out = append(out, shown{"lead", b.Text, leadMaxRunes})
			case "paragraph":
				out = append(out, shown{"paragraphe", b.Text, nextMaxRunes})
			case "item":
				max := itemMaxRunes
				if ev.Section == SecArchitecture {
					max = noteMaxRunes // les garde-fous, tels que l'Architecte les a énoncés
				}
				out = append(out, shown{"puce " + ev.Section, b.Text, max})
			case "proof":
				out = append(out, shown{"preuve", b.Why, whyMaxRunes})
			case "pipeline":
				for _, n := range b.Nodes {
					out = append(out, shown{"titre", n.Title, nodeTitleMaxRunes}, shown{"sous-titre", n.Subtitle, nodeSubtitleMaxRunes})
				}
				out = append(out, shown{"journal", b.Journal, journalMaxRunes}, shown{"label", b.Label, labelMaxRunes})
			}
		case AgentLog:
			out = append(out, shown{"note " + string(ev.Agent), ev.Text, noteMaxRunes})
		case ProjectVerdict:
			out = append(out, shown{"verdict", ev.Note, noteMaxRunes})
		case Message:
			if ev.From != Visitor {
				out = append(out, shown{"message", ev.Text, messageMaxRunes})
			}
		}
	}
	return out
}

func TestRun_TextesAffichesJamaisCoupes(t *testing.T) {
	g := newFakeGateway(t)
	g.answers = longAnswers()
	rec, err := runOnce(t, newTestService(t, g))
	require.NoError(t, err)

	texts := visibleTexts(rec.all())
	require.NotEmpty(t, texts)
	for _, s := range texts {
		assert.False(t, strings.HasSuffix(s.text, "…"), "%s coupé : %q", s.where, s.text)
		assert.LessOrEqual(t, utf8.RuneCountInString(s.text), s.max, "%s hors borne : %q", s.where, s.text)
	}
	// Le modèle a tenu la consigne de réécriture : ce sont SES textes courts qui s'affichent, pas
	// une coupe (la coupe aux frontières est un repli, cf. le test suivant).
	for _, s := range texts {
		if s.where == "preuve" || strings.HasPrefix(s.where, "puce") || s.where == "lead" || s.where == "paragraphe" {
			assert.Contains(t, s.text, "Réécrit", "%s : %q", s.where, s.text)
		}
	}
	shorten := 0
	for _, task := range g.calls() {
		if task == "shorten" {
			shorten++
		}
	}
	// Une réécriture par réponse trop longue, jamais une par texte : plan, lecteur, chaque verdict
	// et leçon, conception, schéma, section 1, apport.
	assert.GreaterOrEqual(t, shorten, 6)
	// Toutes sur le modèle des agents, en effort bas — celle des libellés du schéma comprise :
	// Sonnet ne sert qu'au dessin.
	for i, task := range g.calls() {
		if task == "shorten" {
			assert.Equal(t, testModel, g.bodies[i]["model"])
			assert.Equal(t, map[string]any{"effort": "low"}, g.bodies[i]["output_config"])
		}
	}
}

// Sans rien de trop long, aucun appel de réécriture : le chemin courant ne paie rien.
func TestRun_TextesQuiTiennentSansReecriture(t *testing.T) {
	g := newFakeGateway(t)
	_, err := runOnce(t, newTestService(t, g))
	require.NoError(t, err)
	assert.NotContains(t, g.calls(), "shorten")
}

// Repli : la réécriture rend encore trop long, ou l'appel échoue (sortie tronquée). Le run va au
// bout — pas de nouvel arrêt — et chaque texte affiché s'arrête à une frontière de phrase, de
// proposition ou de mot : jamais en plein mot, jamais d'ellipse.
func TestRun_ReecritureRateeCoupeAuxFrontieres(t *testing.T) {
	for name, setup := range map[string]func(*fakeGateway){
		"réécriture trop longue": func(g *fakeGateway) {
			g.answers["shorten"] = map[string]any{"texts": []string{longText(400), longText(400), longText(400), longText(400), longText(400), longText(400)}}
		},
		"réécriture tronquée": func(g *fakeGateway) { g.truncate = "shorten" },
	} {
		t.Run(name, func(t *testing.T) {
			g := newFakeGateway(t)
			g.answers = longAnswers()
			setup(g)
			rec, err := runOnce(t, newTestService(t, g))
			require.NoError(t, err, "une réécriture ratée n'arrête pas le run")
			if g.truncate == "" {
				// Chaque réponse trop longue a eu sa seconde chance avant la coupe.
				shorten := 0
				for _, task := range g.calls() {
					if task == "shorten" {
						shorten++
					}
				}
				assert.GreaterOrEqual(t, shorten, 12, "deux réécritures avant de couper")
			}
			// Un élément de liste qui ne tient toujours pas est ÉCARTÉ, pas amputé : ni puce du
			// besoin ni de l'apport, ni note du Lecteur ; les textes seuls restent (coupés proprement).
			for _, e := range rec.all() {
				if b, ok := e.(DossierBlock); ok && b.Block.Type == "item" {
					assert.Equal(t, SecArchitecture, b.Section, "puce trop longue affichée : %q", b.Block.Text)
				}
				if l, ok := e.(AgentLog); ok && l.Agent == Reader {
					assert.NotEqual(t, "question", l.Kind, "question trop longue affichée : %q", l.Text)
				}
			}
			whole := longText(400)
			for _, s := range visibleTexts(rec.all()) {
				assert.False(t, strings.HasSuffix(s.text, "…"), "%s coupé : %q", s.where, s.text)
				assert.LessOrEqual(t, utf8.RuneCountInString(s.text), s.max, "%s hors borne : %q", s.where, s.text)
				// Un texte tiré du long texte s'arrête là où le long texte avait une frontière.
				if text := strings.TrimSuffix(s.text, "."); strings.HasPrefix(whole, text) && len(text) < len(whole) {
					next, _ := utf8.DecodeRuneInString(whole[len(text):])
					assert.False(t, unicode.IsLetter(next) || unicode.IsDigit(next), "%s coupé en plein mot : %q", s.where, s.text)
				}
			}
		})
	}
}

func TestCutAtBoundary(t *testing.T) {
	for name, c := range map[string]struct {
		in   string
		max  int
		want string
	}{
		"tient":             {"Un texte court.", 40, "Un texte court."},
		"fin de phrase":     {"Le modèle propose. Un humain valide chaque envoi avant le départ.", 30, "Le modèle propose."},
		"proposition":       {"Alexis a conçu un pipeline asynchrone, relu par un second agent puis testé", 50, "Alexis a conçu un pipeline asynchrone"},
		"point-virgule fr":  {"Chaque devis est préparé par un agent ; un humain le valide avant envoi", 45, "Chaque devis est préparé par un agent"},
		"mot":               {"Validation humaine explicite avant tout envoi au client final", 40, "Validation humaine explicite avant tout"},
		"décimale ≠ phrase": {"Un coût de 1.5 centime par devis traité, mesuré en production", 30, "Un coût de 1.5 centime par"},
		"chinois":           {"系统先由模型起草报价，再由人工审核后发送给客户。", 12, "系统先由模型起草报价"},
	} {
		t.Run(name, func(t *testing.T) {
			got := cutAtBoundary(c.in, c.max)
			assert.Equal(t, c.want, got)
			assert.LessOrEqual(t, utf8.RuneCountInString(got), c.max)
		})
	}
}

// La citation du besoin dans le fil : un EXTRAIT (l'ellipse le dit), jamais coupé en plein mot.
func TestQuote_ExtraitAUneFrontiere(t *testing.T) {
	q := quote("Notre bureau d’études reçoit 300 demandes de devis par semaine ; nous voulons des agents.")
	assert.Equal(t, "« Notre bureau d’études reçoit 300 demandes de devis par… »", q)
	assert.Equal(t, "« Court besoin. »", quote("  Court besoin. "))
}

// La note « Chaîne : … » ne liste que des titres ENTIERS ; ceux qui n'y tiennent plus sont comptés.
func TestChainNote_TitresEntiers(t *testing.T) {
	r := &run{ph: phrases("fr")}
	titles := []string{"Demande devis", "Contrôle champs", "Chiffrage", "Projet de devis", "Validation", "Secrétariat", "Archivage"}
	note := r.chainNote(titles)
	assert.LessOrEqual(t, utf8.RuneCountInString(note), noteMaxRunes)
	assert.Equal(t, "Chaîne : Demande devis, Contrôle champs, Chiffrage, Projet de devis (+3)", note)
	assert.Equal(t, "Chaîne : Demande devis, Chiffrage", r.chainNote([]string{"Demande devis", "Chiffrage"}))
}

// Le premier objet de la réponse fait foi : clôture de code autour, seconde pensée après (mesuré
// sur des réécritures réelles : « {...}\nWait, … {...} »).
func TestDecodeJSON_PremierObjet(t *testing.T) {
	var out shortenOut
	require.NoError(t, decodeJSON("```json\n{\"texts\": [\"Court.\"]}\n```\nWait, {\"texts\": [\"autre\"]}", &out))
	assert.Equal(t, []string{"Court."}, out.Texts)
	assert.Error(t, decodeJSON("pas d'objet", &out))
}
