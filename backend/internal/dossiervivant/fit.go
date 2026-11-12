package dossiervivant

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
)

// QUOI : faire TENIR chaque texte de modèle affiché (dossier, PDF, colonnes, fil) dans sa borne,
// sans jamais le couper en plein mot ni finir sur une ellipse.
//
// POURQUOI : l'ancien `clip` rognait à N runes et posait « … ». Tant que les agents réfléchissaient,
// ils tenaient à peu près leurs longueurs ; avec l'effort bas (mind.go › agentEffort, gain de 35 s
// par run) ils ne comptent plus. Mesuré le 2026-11-07 sur trois runs réels : 11 raisons de preuve
// sur 12 dépassaient leur borne, et le PDF remis portait « avec auto-save s… », « avant tout e… ».
// Élargir les bornes n'est pas une issue : elles sont la place que le texte a dans sa colonne.
//
// COMMENT, dans l'ordre :
//  1. la consigne de chaque tâche demande une longueur sous la borne (prompts.go : en caractères,
//     pas en mots, qui font réfléchir le modèle) — une partie des réponses tient d'emblée ;
//  2. `fit` renvoie à l'agent, en UN seul appel, les textes de sa réponse qui dépassent (tâche
//     `shorten`) : il les réécrit plus courts, complets. Aucun appel quand tout tient ;
//  3. un texte qui dépasse encore est renvoyé une seconde fois, cible resserrée ;
//  4. s'il dépasse toujours (ou si les appels échouent) : un élément de LISTE (notes, questions,
//     puces) est écarté — deux puces entières valent mieux qu'une troisième amputée (mesuré : une
//     question du Lecteur finissait sur « …et quelles contraintes de ») ; un texte seul est raccourci
//     par cutAtBoundary, à une fin de phrase ou de proposition de préférence, jamais en plein mot,
//     jamais d'ellipse. Ce repli est journalisé : il ne doit pas devenir l'usage.
// Un appel de réécriture en échec n'arrête PAS le run : le texte a toujours une forme affichable.

// span — un texte de modèle à afficher, et sa borne en runes. `item` : élément d'une liste, que le
// repli écarte (texte vidé, cf. filled) plutôt que de le couper.
type span struct {
	text *string
	max  int
	item bool
}

// spansOf — une span par élément de `lines`, toutes à la même borne, JAMAIS écartées (le nombre
// compte : les garde-fous sont numérotés sur le schéma).
func spansOf(lines []string, max int) []span {
	out := make([]span, len(lines))
	for i := range lines {
		out[i] = span{text: &lines[i], max: max}
	}
	return out
}

// itemSpans — comme spansOf, pour une liste dont un élément peut être écarté (cf. en-tête, 4).
// L'appelant relit la liste par filled après fit.
func itemSpans(lines []string, max int) []span {
	out := spansOf(lines, max)
	for i := range out {
		out[i].item = true
	}
	return out
}

// filled — les éléments non vides de `lines` (ceux que fit n'a pas écartés), dans leur ordre.
func filled(lines []string) []string {
	out := lines[:0:0]
	for _, l := range lines {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

type shortenOut struct {
	Texts []string `json:"texts"`
}

// agentRoles — le rôle de chaque agent dans la consigne de réécriture (même voix que ses tâches).
var agentRoles = map[AgentID]string{
	Orchestrator: "the orchestrator",
	Reader:       "the need reader",
	Researcher:   "the proof researcher",
	Architect:    "the architect",
	Diagrammer:   "the diagram drafter",
	Writer:       "the writer",
}

// shortenTask — la réécriture, au nom de l'agent qui a écrit les textes.
func shortenTask(agent AgentID) task {
	return task{"shorten", agentRoles[agent], 1500,
		`{"texts": ["every text below rewritten, in the same order, one entry per text"]}
Each text below is too long for the room it has in the dossier. Rewrite each one to fit the limit given in its brackets: same language, same form (a sentence stays one complete sentence, a short label stays a label), same meaning, same names and project names. Drop secondary details rather than cutting words. Never end with an ellipsis.`}
}

// fit — chaque span, ramenée sous sa borne (cf. en-tête). Les textes sont aussi débarrassés de leurs
// blancs de bord. `agent` réécrit ses propres textes ; les appels comptent dans son usage.
func (r *run) fit(ctx context.Context, agent AgentID, spans ...span) {
	var long []span
	for _, s := range spans {
		*s.text = strings.TrimSpace(*s.text)
		if utf8.RuneCountInString(*s.text) > s.max {
			long = append(long, s)
		}
	}
	if len(long) > 0 {
		// Mesure : quels textes dépassent, et de combien (cf. NOTES-mesures, chantier dv-texte).
		over := make([]string, len(long))
		for i, s := range long {
			over[i] = fmt.Sprintf("%d>%d", utf8.RuneCountInString(*s.text), s.max)
		}
		log.Info().Str("agent", string(agent)).Strs("over", over).Msg("dossier vivant: textes trop longs, réécriture")
	}
	// Deux réécritures au plus, la seconde sur les seuls textes encore trop longs, cible resserrée.
	// POURQUOI une seconde : mesuré sur cinq runs réels, une réécriture sur ~20 rend encore trop
	// long ; coupée aux frontières, la question du Lecteur finissait sur « …automatiquement ou ».
	// Elle part rarement, et toujours sur un texte qu'on a déjà sous la main.
	for _, share := range []int{85, 70} {
		if len(long) == 0 {
			return
		}
		long = r.rewrite(ctx, agent, long, share)
	}
	for _, s := range long {
		l := log.Warn().Str("agent", string(agent)).Int("max", s.max).Int("runes", utf8.RuneCountInString(*s.text))
		if s.item {
			l.Msg("dossier vivant: élément encore trop long après deux réécritures, écarté")
			*s.text = ""
			continue
		}
		l.Msg("dossier vivant: texte encore trop long après deux réécritures, coupé à une frontière")
		*s.text = cutAtBoundary(*s.text, s.max)
	}
}

// rewrite — UN appel de réécriture pour les textes `long`, chacun visé à `share` % de sa borne.
// Rend ceux qui dépassent encore. Une réécriture plus courte mais encore trop longue remplace
// l'original : le passage suivant (réécriture ou coupe) part du texte le plus proche de tenir.
func (r *run) rewrite(ctx context.Context, agent AgentID, long []span, share int) []span {
	var sb strings.Builder
	sb.WriteString("<texts>\n")
	for i, s := range long {
		// La cible est SOUS la borne : un modèle qui ne réfléchit pas dépasse volontiers de quelques
		// caractères ce qu'on lui demande. La longueur actuelle lui dit combien retrancher.
		target := s.max * share / 100
		fmt.Fprintf(&sb, "%d. [now %d characters; at most %d characters] %s\n", i+1, utf8.RuneCountInString(*s.text), target, *s.text)
	}
	sb.WriteString("</texts>\n")
	// Toujours le modèle des agents, Dessinateur compris : Sonnet ne sert qu'au dessin du schéma
	// (une réécriture de ses libellés sur Sonnet a coûté jusqu'à 0,004 $, le tiers d'un run).
	// Effort bas : réécrire n'appelle pas de réflexion.
	var out shortenOut
	if err := r.ask(ctx, agent, shortenTask(agent), sb.String(), &out); err != nil {
		// Pas d'arrêt : le texte garde sa forme, le passage suivant le reprend.
		log.Warn().Err(err).Str("agent", string(agent)).Msg("dossier vivant: réécriture des textes trop longs en échec")
		return long
	}
	var still []span
	for i, s := range long {
		t := ""
		if i < len(out.Texts) {
			t = strings.TrimSpace(out.Texts[i])
		}
		if t == "" || strings.HasSuffix(t, "…") {
			still = append(still, s)
			continue
		}
		if n := utf8.RuneCountInString(t); n <= s.max {
			*s.text = t
			continue
		} else if n < utf8.RuneCountInString(*s.text) {
			*s.text = t
		}
		still = append(still, s)
	}
	return still
}

// later — ce qu'un agent laisse courir pendant qu'il poursuit son chemin : la réécriture d'un texte
// à afficher, et son affichage. POURQUOI : sur le chemin le plus long du run (l'Architecte), une
// réécriture attendue sur place coûte un appel de passerelle entier (3 à 5 s). COMMENT : l'agent
// attend `wait` avant de lire les textes, et avant de rendre la main — en erreur comprise, pour
// qu'aucun événement ne suive la fin du run. Sur un contexte qui vit aussi longtemps que l'agent.
type later struct {
	ctx context.Context
	wg  sync.WaitGroup
}

func (l *later) do(f func(context.Context)) {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		f(l.ctx)
	}()
}

func (l *later) wait() { l.wg.Wait() }

// firstLines — au plus `n` lignes non vides de `lines`, sans blancs de bord, dans leur ordre.
func firstLines(lines []string, n int) []string {
	out := make([]string, 0, n)
	for _, l := range lines {
		if l = strings.TrimSpace(l); l == "" {
			continue
		}
		if len(out) == n {
			break
		}
		out = append(out, l)
	}
	return out
}

// cutAtBoundary — `s` ramené à `max` runes au plus, à la dernière frontière qui laisse un texte
// lisible : fin de phrase (gardée), sinon ponctuation de proposition (retirée), sinon espace.
// Jamais en plein mot, jamais d'ellipse. Une phrase ou une proposition est retenue si elle garde
// au moins le tiers de la borne (une idée entière, même courte, plutôt qu'une idée amputée) ; un
// simple espace, s'il garde la moitié. Sans aucune frontière (chinois sans
// ponctuation), la coupe tombe entre deux caractères, qui y sont chacun un mot ou une syllabe.
func cutAtBoundary(s string, max int) string {
	s = strings.TrimSpace(s)
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	floor, wordFloor := max/3, max/2
	// 1. Fin de phrase : la ponctuation est gardée ; elle doit être suivie d'un blanc (« 1.5 » n'en
	//    est pas une) — sauf la ponctuation chinoise, qui ne prend pas d'espace.
	for i := max - 1; i >= floor; i-- {
		if strings.ContainsRune("。！？", rs[i]) || (strings.ContainsRune(".!?", rs[i]) && unicode.IsSpace(rs[i+1])) {
			return string(rs[:i+1])
		}
	}
	// 2. Proposition : on coupe AVANT la ponctuation (« , » « ; » « : » « — », et leurs formes
	//    chinoises), le texte s'arrête sur le dernier mot de la proposition.
	for i := max; i >= floor; i-- {
		if strings.ContainsRune(",;:—–(，；：、（", rs[i]) {
			if t := strings.TrimSpace(string(rs[:i])); utf8.RuneCountInString(t) >= floor {
				return t
			}
		}
	}
	// 3. Mot : le dernier espace sous la borne.
	for i := max; i >= wordFloor; i-- {
		if unicode.IsSpace(rs[i]) {
			return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(string(rs[:i])), ",;:—–"))
		}
	}
	return string(rs[:max])
}
