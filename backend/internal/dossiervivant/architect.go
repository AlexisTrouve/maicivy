package dossiervivant

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
)

// QUOI : l'ARCHITECTE. Choisit dans l'index deux ou trois projets dont l'architecture éclaire la
// solution (un appel), en lit les briefs et en tire une leçon chacun (un appel par brief), puis
// CONÇOIT la chaîne proposée — étapes, embranchements, trois garde-fous (un appel). Son livrable :
// le schéma de la section 2 et ses trois garde-fous.
//
// Le schéma, il ne le dessine pas : il appelle son outil `draw_schema`, qui lance le DESSINATEUR
// (diagrammer.go), un agent sur un autre modèle dont c'est le seul travail. POURQUOI : dessiné par
// Haiku, le schéma restait générique (« Réception → Extraction → Validation ») alors que c'est la
// figure qu'un décideur lit en premier. L'Architecte garde la conception ; le Dessinateur reçoit le
// besoin reformulé (Lecteur), la conception et les leçons, et rend une grille que le code valide.
//
// COMMENT les livrables se croisent : le Dessinateur a besoin du besoin reformulé — l'Architecte
// attend donc le Lecteur avant d'appeler l'outil (en pratique déjà fini : le Lecteur rend en un
// appel, l'Architecte en a fait trois ou quatre avant).

type archOut struct {
	Pipeline   Block
	Guardrails []string
	Kept       []string // projets dont la leçon a servi (allumés par l'Architecte)
	Summary    string
}

type refsOut struct {
	Refs []string `json:"refs"`
}

type lessonOut struct {
	Useful bool   `json:"useful"`
	Lesson string `json:"lesson"`
}

// designOut — la réponse de l'Architecte à la tâche `design` : ce qu'il confie au Dessinateur.
type designOut struct {
	Stages     []string `json:"stages"`
	Branches   []string `json:"branches"`
	Guardrails []string `json:"guardrails"`
	Check      string   `json:"check"`
	Summary    string   `json:"summary"`
}

const (
	maxStages      = 6
	maxBranches    = 2
	stageMaxRunes  = 90 // une étape va au Dessinateur (pas à l'écran) : assez pour dire qui agit
	branchMaxRunes = 110
)

const (
	maxRefs   = 3
	tightRefs = 2 // … quand le run approche de sa cible de durée (tempo.go)
)

// learnedLesson — une leçon retenue : le projet, et « Nom : idée à réutiliser ».
type learnedLesson struct{ id, text string }

// lesson — ouvre le brief de `id` et en tire la leçon. La leçon — et le verdict qu'elle porte — se
// tire du brief complet (brief.go), jamais de la ligne d'index qui a fait choisir le projet.
//
// Le verdict s'affiche via `shown` : sa note est réécrite si elle dépasse (fit.go) PENDANT que
// l'Architecte conçoit — la leçon brute part aussitôt dans le prompt de conception, qui ne
// l'affiche pas. Réécrite sur le chemin de l'Architecte, elle retardait d'autant le schéma (mesuré :
// deux réécritures, +9 s sur un run réel).
func (r *run) lesson(ctx context.Context, shown *later, id string) (learnedLesson, bool, error) {
	r.tool(Architect, fmt.Sprintf("get_project(%q)", id), onProject(id))
	b, err := r.readBrief(ctx, id)
	if err != nil {
		return learnedLesson{}, false, err
	}
	var l lessonOut
	j, ok, err := r.judgeOn(ctx, Architect, taskLesson, b, &l)
	if err != nil || !ok {
		return learnedLesson{}, false, err
	}
	reason := strings.TrimSpace(l.Lesson)
	shown.do(func(c context.Context) {
		r.fit(c, Architect, span{text: &reason, max: r.reasonRoom(j.brief.Name)})
		r.emitVerdict(Architect, j, l.Useful, reason)
		if l.Useful {
			r.emit.Emit(LinkTraced{From: Visitor, To: id})
		}
	})
	if !l.Useful {
		return learnedLesson{}, false, nil
	}
	return learnedLesson{id: id, text: b.Name + ": " + strings.TrimSpace(l.Lesson)}, true, nil
}

func (r *run) architect(ctx context.Context, needIn *future[readerOut]) (archOut, error) {
	r.status(Architect, "working", r.ph.get("arch.reading"))
	r.tool(Architect, "search_architectures(need)", onVisitor)
	var refs refsOut
	if err := r.ask(ctx, Architect, taskRefs, r.indexPrompt(taskRefs), &refs); err != nil {
		return archOut{}, err
	}
	ids, invented := r.catalog.validate(refs.Refs, r.tempo.cap(maxRefs, tightRefs))
	if len(invented) > 0 {
		log.Warn().Strs("ids", invented).Msg("dossier vivant: ids inventés par le modèle, écartés")
	}

	// Les leçons, EN PARALLÈLE (comme les verdicts du Chercheur, cf. judgeAll) : enchaînées, elles
	// coûtaient trois appels de suite sur le chemin de l'Architecte (mesuré : 11,3 → 24,5 s). Leurs
	// verdicts s'affichent par `shown`, attendu avant que l'Architecte rende la main (y compris en
	// erreur : aucun événement ne suit la fin du run). Sur le contexte de l'Architecte, pas sur celui
	// de judgeAll, annulé dès qu'il rend.
	shown := &later{ctx: ctx}
	defer shown.wait()
	learned, err := judgeAll(ctx, r, ids, func(c context.Context, id string) (learnedLesson, bool, error) {
		return r.lesson(c, shown, id)
	})
	if err != nil {
		return archOut{}, err
	}
	var out archOut
	var lessons []string
	for _, l := range learned {
		out.Kept = append(out.Kept, l.id)
		lessons = append(lessons, l.text)
	}

	r.status(Architect, "working", r.ph.get("arch.designing"))
	r.tool(Architect, "design_pipeline(need, lessons)", onVisitor)
	var d designOut
	if err := r.ask(ctx, Architect, taskDesign, needBlock(r.brief)+"<lessons>\n"+bullets(lessons)+"</lessons>\n", &d); err != nil {
		return archOut{}, err
	}
	stages := firstLines(d.Stages, maxStages)
	if len(stages) == 0 {
		return archOut{}, fmt.Errorf("%w: conception sans étape", ErrModelOutput)
	}
	branches := firstLines(d.Branches, maxBranches)
	// Étapes et embranchements ne s'affichent pas (ils vont au Dessinateur) : une coupe aux
	// frontières suffit à borner son prompt. Garde-fous (colonne ET dossier), vérification et compte
	// rendu s'affichent : réécrits s'ils dépassent (cf. fit.go).
	for i := range stages {
		stages[i] = cutAtBoundary(stages[i], stageMaxRunes)
	}
	for i := range branches {
		branches[i] = cutAtBoundary(branches[i], branchMaxRunes)
	}
	r.note(Architect, "note", r.ph.get("arch.stages", len(stages), len(branches)))
	// Les garde-fous partent tels quels dans la commande du Dessinateur (il ne les affiche pas), et
	// leur réécriture éventuelle court PENDANT qu'il dessine : faite avant la commande, elle
	// retardait le schéma, dernier maillon du chemin de l'Architecte.
	out.Guardrails = firstLines(d.Guardrails, 3)
	ordered := append([]string(nil), out.Guardrails...)
	shown.do(func(c context.Context) {
		r.fit(c, Architect, append(spansOf(out.Guardrails, noteMaxRunes), span{text: &d.Check, max: noteMaxRunes}, span{text: &d.Summary, max: messageMaxRunes})...)
	})

	// L'outil `draw_schema` : le Dessinateur travaille PENDANT que l'Architecte énonce ses
	// garde-fous (déjà décidés : ils sont dans la commande). Énoncés avant la commande, leurs trois
	// battements retardaient d'autant le schéma, dernier maillon du chemin de l'Architecte.
	need, err := needIn.wait(ctx)
	if err != nil {
		return archOut{}, err
	}
	r.status(Architect, "working", r.ph.get("arch.commission"))
	r.tool(Architect, fmt.Sprintf("draw_schema(%d stages, %d guardrails)", len(stages), len(out.Guardrails)), onVisitor)
	r.say(string(Architect), string(Diagrammer), r.ph.get("arch.order", len(stages), len(out.Guardrails)), "")
	type drawn struct {
		pipeline Block
		err      error
	}
	schema := make(chan drawn, 1)
	go func() {
		p, err := r.diagrammer(ctx, schemaOrder{Need: need, Stages: stages, Branches: branches, Guardrails: ordered, Lessons: lessons})
		schema <- drawn{p, err}
	}()
	r.status(Architect, "working", r.ph.get("arch.guardrails"))
	// Garde-fous, vérification et compte rendu ne se lisent qu'une fois réécrits (et les verdicts
	// des leçons émis) : `shown` fini.
	shown.wait()
	for _, g := range out.Guardrails {
		r.beat(ctx)
		r.note(Architect, "note", g)
	}
	r.status(Architect, "waiting", r.ph.get("arch.waitDiagram"))
	// Toujours attendu, erreur comprise : le run ne rend la main qu'une fois le Dessinateur fini.
	got := <-schema
	if got.err != nil {
		return archOut{}, got.err
	}
	pipeline := got.pipeline
	titles := make([]string, len(pipeline.Nodes))
	for i, n := range pipeline.Nodes {
		titles[i] = n.Title
	}
	r.status(Architect, "working", r.ph.get("arch.checking"))
	r.note(Architect, "note", r.chainNote(titles))
	r.note(Architect, "note", d.Check)
	r.tool(Architect, `send("orchestrator", architecture)`, onVisitor)
	r.status(Architect, "done", "")
	r.note(Architect, "done", r.ph.get("arch.done"))

	out.Pipeline = pipeline
	out.Summary = d.Summary
	return out, ctx.Err()
}

// chainNote — « Chaîne : A, B, C » : les titres du schéma qui tiennent ENTIERS dans une ligne de
// journal, puis « (+n) » pour ceux qui n'y entrent plus. POURQUOI : rognée au caractère, la liste
// finissait sur « Valida… » — un titre coupé ne désigne plus aucune boîte du schéma.
func (r *run) chainNote(titles []string) string {
	for n := len(titles); n > 0; n-- {
		note := r.ph.get("arch.chain", strings.Join(titles[:n], ", "))
		if n < len(titles) {
			note += fmt.Sprintf(" (+%d)", len(titles)-n)
		}
		if utf8.RuneCountInString(note) <= noteMaxRunes {
			return note
		}
	}
	return cutAtBoundary(r.ph.get("arch.chain", titles[0]), noteMaxRunes)
}
