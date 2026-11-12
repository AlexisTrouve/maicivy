package dossiervivant

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

// QUOI : le CHERCHEUR DE PREUVES. Lit l'index (la montagne), en filtre les candidats (un appel),
// ouvre leurs briefs et rend un verdict sur chacun (un appel par brief, EN PARALLÈLE), puis classe
// les retenus : les meilleurs deviennent les preuves du dossier, les autres « aussi retenus ».
//
// POURQUOI en parallèle : les verdicts enchaînés un à un étaient le chemin le plus long du run —
// mesuré le 2026-11-07 sur un run réel, six verdicts de 13,9 s à 63,3 s, soit les deux tiers de ses
// 74,8 s. Les verdicts ne dépendent pas les uns des autres. Les lectures partent à un battement
// d'écart : le curseur du Chercheur passe encore de carte en carte, sans que les appels s'attendent.
//
// POURQUOI un appel par brief : c'est le geste qu'on voit à l'écran (le curseur va sur la carte,
// `get_project` s'écrit, le verdict tombe). Et un brief lu entier juge mieux qu'un résumé d'une
// ligne — leçon du projet : un agent qui a l'inventaire sous les yeux cesse d'ouvrir les fiches.
// L'index sert à CHOISIR quoi lire ; le verdict, lui, ne se rend que sur le brief (brief.go).
//
// POURQUOI au plus maxReads candidats : chaque candidat teinté sur la montagne reçoit un verdict,
// et un verdict ne part que sur un brief lu. Garder plus de candidats qu'on n'en lit obligeait à
// rendre au rang des projets que personne n'avait ouverts (`drop` sans lecture, mesuré sur le
// premier run réel : `wanmira`).
//
// COMMENT les identifiants restent réels : la liste de candidats rendue par le modèle passe par
// catalog.validate — un id absent de la montagne est journalisé et écarté, aucun événement ne le
// porte. Le classement est fait par le CODE (score du verdict, puis rang rendu par le filtre).
// Le nombre de fiches ouvertes se resserre quand le run approche de sa cible (tempo.cap).

type researchOut struct {
	Kept   []keptProject // dans l'ordre du classement
	Proofs []keptProject // les preuves principales (tête de Kept)
	Also   []string      // le reste des retenus
}

// keptProject — un projet retenu par un verdict rendu sur son brief. Le brief suit : le Rédacteur
// le relit pour écrire sur ce projet.
type keptProject struct {
	ID    string
	Why   string
	Score int
	brief projectBrief
}

const (
	maxReads    = 7 // candidats = briefs ouverts
	tightReads  = 4 // … quand le run approche de sa cible de durée (tempo.go)
	maxProofs   = 4 // lignes du tableau des preuves
	whyMaxRunes = 110
)

type filterOut struct {
	Candidates []string `json:"candidates"`
}

type verdictOut struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
	Why     string `json:"why"`
	Score   int    `json:"score"`
}

func (r *run) researcher(ctx context.Context) (researchOut, error) {
	r.status(Researcher, "working", r.ph.get("research.index"))
	r.tool(Researcher, "get_projects_index()", onVisitor)
	total := len(r.catalog.entries)
	r.note(Researcher, "note", r.ph.get("research.indexed", total))

	r.status(Researcher, "working", r.ph.get("research.filtering"))
	r.tool(Researcher, "filter_projects(need)", onVisitor)
	var f filterOut
	if err := r.ask(ctx, Researcher, taskFilter, r.indexPrompt(taskFilter), &f); err != nil {
		return researchOut{}, err
	}
	candidates, invented := r.catalog.validate(f.Candidates, r.tempo.cap(maxReads, tightReads))
	if len(invented) > 0 {
		log.Warn().Strs("ids", invented).Msg("dossier vivant: ids inventés par le modèle, écartés")
	}
	r.emit.Emit(IndexFiltered{Agent: Researcher, Total: total, Candidates: candidates})
	r.note(Researcher, "note", r.ph.get("research.candidates", len(candidates)))

	r.status(Researcher, "working", r.ph.get("research.reading"))
	kept, err := judgeAll(ctx, r, candidates, r.judge)
	if err != nil {
		return researchOut{}, err
	}

	r.status(Researcher, "working", r.ph.get("research.ranking"))
	r.tool(Researcher, "rank(kept)", onVisitor)
	// Stable : à score égal, le rang rendu par le filtre (meilleur d'abord) départage.
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].Score > kept[j].Score })
	out := researchOut{Kept: kept, Proofs: kept[:min(len(kept), maxProofs)]}
	for _, k := range kept[len(out.Proofs):] {
		out.Also = append(out.Also, k.ID)
	}
	r.note(Researcher, "note", r.ph.get("research.ranked", len(out.Proofs), len(kept)))
	r.tool(Researcher, `send("orchestrator", proofs)`, onVisitor)
	r.status(Researcher, "done", "")
	r.note(Researcher, "done", r.ph.get("research.done"))
	return out, ctx.Err()
}

// judge — ouvre le brief de `id` et rend le verdict du modèle, jugé sur ce brief ; publie le
// verdict (et le filet si le projet est retenu).
func (r *run) judge(ctx context.Context, id string) (keptProject, bool, error) {
	r.tool(Researcher, fmt.Sprintf("get_project(%q)", id), onProject(id))
	b, err := r.readBrief(ctx, id)
	if err != nil {
		return keptProject{}, false, err
	}
	var v verdictOut
	j, ok, err := r.judgeOn(ctx, Researcher, taskVerdict, b, &v)
	if err != nil || !ok {
		return keptProject{}, false, err
	}
	verdict := strings.ToLower(strings.TrimSpace(v.Verdict))
	if verdict != "keep" && verdict != "drop" {
		return keptProject{}, false, fmt.Errorf("%w: verdict %q sur %s", ErrModelOutput, v.Verdict, id)
	}
	keep := verdict == "keep"
	// La raison s'affiche dans la colonne, la phrase `why` dans le dossier et le PDF (cf. fit.go) ;
	// `why` n'est réécrite que pour un projet retenu — elle n'est affichée que pour lui.
	spans := []span{{text: &v.Reason, max: r.reasonRoom(j.brief.Name)}}
	if keep {
		spans = append(spans, span{text: &v.Why, max: whyMaxRunes})
	}
	r.fit(ctx, Researcher, spans...)
	r.emitVerdict(Researcher, j, keep, v.Reason)
	if !keep {
		return keptProject{}, false, nil
	}
	r.emit.Emit(LinkTraced{From: Visitor, To: id})
	return keptProject{ID: id, Why: v.Why, Score: v.Score, brief: j.brief}, true, nil
}

// judgeAll — `judge` sur chaque id, EN PARALLÈLE, les départs espacés d'un battement (le curseur de
// l'agent va de carte en carte). Rend les retenus dans l'ordre des ids, pas dans l'ordre d'arrivée :
// le classement en dépend. La première erreur annule les autres jugements et devient celle de
// l'agent. Partagé par le Chercheur (verdicts) et l'Architecte (leçons).
func judgeAll[T any](ctx context.Context, r *run, ids []string, judge func(context.Context, string) (T, bool, error)) ([]T, error) {
	type result struct {
		v    T
		keep bool
	}
	results := make([]result, len(ids))
	group := newCrew(ctx)
	for i, id := range ids {
		if i > 0 {
			r.beat(group.ctx)
		}
		group.run(func(c context.Context) error {
			v, keep, err := judge(c, id)
			results[i] = result{v, keep}
			return err
		})
	}
	if err := group.wait(); err != nil {
		return nil, err
	}
	var kept []T
	for _, res := range results {
		if res.keep {
			kept = append(kept, res.v)
		}
	}
	return kept, nil
}

// indexPrompt — le besoin et l'index, pour une tâche qui choisit dans l'index (filtre, références).
// L'index vient de la montagne (donnée externe, non bornée) : ses lignes entrent tant qu'elles
// tiennent dans l'appel (callsize.go). À 79 projets il pèse quelques milliers de tokens, loin du
// plafond ; une ligne laissée de côté serait journalisée.
func (r *run) indexPrompt(t task) string {
	head, tail := needBlock(r.brief)+"<index>\n", "</index>\n"
	lines, dropped := fitLines(r.catalog.indexLines(), r.room(t)-estimateTokens(head, tail))
	if dropped > 0 {
		log.Warn().Int("dropped", dropped).Str("task", t.name).Msg("dossier vivant: index rogné pour tenir dans un appel")
	}
	return head + lines + tail
}
