package dossiervivant

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
)

// QUOI : un RUN du dossier vivant, conduit par l'orchestrateur.
//  1. run.started (immédiat : le visiteur voit son besoin partir avant tout appel de modèle) ;
//  2. l'orchestrateur lit la demande et répartit le travail (un appel de modèle) pendant que le
//     code charge l'index (la montagne) ;
//  3. les quatre sous-agents partent EN MÊME TEMPS, chacun dans sa goroutine ; le Rédacteur attend
//     les livrables des trois autres, que l'orchestrateur lui RELAIE à mesure qu'ils arrivent ;
//  4. l'orchestrateur contrôle les preuves, remet le dossier : run.completed.
// Toute erreur d'un agent arrête toute l'équipe (contexte annulé avec sa cause) et devient UN
// `run.failed` nommé. Ce qui n'arrête RIEN : la consommation du run et sa durée. Le temps se dit
// aux agents dans leur prompt (tempo.go) ; la taille se tient appel par appel (callsize.go).
//
// POURQUOI des goroutines et pas une boucle : l'écran montre quatre colonnes au travail en même
// temps ; le flux doit le refléter pour de vrai (le test de parallélisme le vérifie sur la
// passerelle : plusieurs appels en vol au même instant).
//
// COMMENT le rythme est tenu : les événements partent au moment où le travail se fait. Seul
// ajout : un battement (`Options.Beat`) entre deux lignes d'un même lot (trois notes rendues par
// un même appel) et la durée de frappe d'un paragraphe du dossier — sans eux, un lot arriverait
// dans la même image et l'œil n'en suivrait rien. Les tests les mettent à zéro.

// Options — réglages d'un service. Les zéros prennent les valeurs de production (cf. NewService).
type Options struct {
	Model string // identifiant affiché et facturé des agents Haiku
	// SchemaModel — modèle du SEUL Dessinateur (diagrammer.go) ; vide = Model. Aucun autre appel
	// du run ne le porte.
	SchemaModel string
	Beat        time.Duration // battement entre deux lignes d'un même lot ; 0 = aucun (tests)
	// Now — l'horloge du tempo (tempo.go) ; nil = time.Now. Les tests la pilotent pour placer un
	// appel avant ou après la cible sans attendre.
	Now func() time.Time
}

const defaultBeat = 350 * time.Millisecond

// Service — produit des runs. Sans état entre deux runs.
type Service struct {
	ai      Completer
	sources Sources
	opts    Options
}

// NewService — `opts.Model` est obligatoire ; Beat négatif = aucun battement.
func NewService(ai Completer, sources Sources, opts Options) *Service {
	if opts.Beat == 0 {
		opts.Beat = defaultBeat
	}
	if opts.Beat < 0 {
		opts.Beat = 0
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.SchemaModel == "" {
		opts.SchemaModel = opts.Model
	}
	return &Service{ai: ai, sources: sources, opts: opts}
}

// Model — l'identifiant de modèle des agents.
func (s *Service) Model() string { return s.opts.Model }

// run — l'état partagé d'un run. Les champs sont posés avant le départ des sous-agents et ne
// bougent plus ensuite ; seuls meter et reads sont écrits en concurrence (ils se verrouillent).
type run struct {
	ai          Completer
	sources     Sources
	model       string
	schemaModel string // le Dessinateur, et lui seul
	lang        string
	ph          phrases
	brief       VisitorBrief
	emit        Emitter
	tempo       tempo // le temps écoulé, rappelé aux agents (jamais un arrêt)
	meter       *meter
	beatD       time.Duration
	catalog     *catalog
	reads       *readLedger // briefs lus pendant le run (mesure, journal de fin)
}

// Run — un run complet, de run.started à run.completed (ou run.failed). Bloque jusqu'à ce que
// toutes les goroutines du run soient finies : aucun Emit ne peut suivre son retour. `conn` est
// le contexte de la connexion du visiteur : s'il tombe, le run s'arrête (plus personne ne regarde).
// C'est le SEUL contexte du run : aucune échéance n'y est ajoutée (cf. tempo.go).
func (s *Service) Run(conn context.Context, brief VisitorBrief, lang string, emit Emitter) error {
	lang = RunLanguage(lang)
	r := &run{
		ai: s.ai, sources: s.sources, model: s.opts.Model, schemaModel: s.opts.SchemaModel, lang: lang, ph: phrases(lang),
		brief: brief, emit: emit, tempo: tempo{start: s.opts.Now(), now: s.opts.Now}, meter: newMeter(), beatD: s.opts.Beat,
		reads: newReadLedger(),
	}
	ctx := conn

	agents := make([]AgentModel, len(Team))
	for i, id := range Team {
		agents[i] = AgentModel{ID: id, Model: s.opts.Model}
		if id == Diagrammer {
			agents[i].Model = s.opts.SchemaModel
		}
	}
	emit.Emit(RunStarted{Visitor: brief, Agents: agents})

	err := r.orchestrate(ctx)
	in, out, cost := r.meter.totals()
	// Le Dessinateur à part : c'est le seul appel sur un autre modèle (et un autre tarif) du run.
	schema := r.meter.of(Diagrammer)
	logged := log.Info()
	if err != nil {
		logged = log.Warn().Err(err).Str("reason", failReason(err))
		// Connexion tombée : personne à prévenir. Sinon l'arrêt se VOIT.
		if conn.Err() == nil {
			emit.Emit(RunFailed{Reason: failReason(err)})
		}
	}
	logged.Str("model", s.opts.Model).Int("input_tokens", in).Int("output_tokens", out).
		Float64("cost_usd", cost).Dur("duration", r.tempo.elapsed()).Str("briefs", r.reads.summary()).
		Str("schema_model", s.opts.SchemaModel).Int("schema_input_tokens", schema.InputTokens).
		Int("schema_output_tokens", schema.OutputTokens).Float64("schema_cost_usd", schema.CostUsd).
		Msg("dossier vivant: run terminé")
	return err
}

// failReason — l'erreur d'un run, nommée pour l'écran. Il n'en reste que deux familles : le modèle
// a rendu une sortie inutilisable, ou la passerelle / une source n'a pas répondu (un appel resté
// sans réponse au-delà de son délai d'appel en fait partie : c'est une panne, pas un run trop long).
func failReason(err error) string {
	if errors.Is(err, ErrModelOutput) {
		return FailModel
	}
	return FailUpstream
}

// planOut — la réponse de l'orchestrateur à la tâche `plan`.
type planOut struct {
	Plan  string `json:"plan"`
	Tasks struct {
		Reader     string `json:"reader"`
		Researcher string `json:"researcher"`
		Architect  string `json:"architect"`
		Writer     string `json:"writer"`
	} `json:"tasks"`
}

func (r *run) orchestrate(ctx context.Context) error {
	r.status(Orchestrator, "working", r.ph.get("reading"))
	r.say(Visitor, string(Orchestrator), quote(r.brief.Need), "")

	// L'index (montagne + livre) se charge pendant que l'orchestrateur réfléchit au plan : attentes
	// réseau indépendantes, aucune raison de les enchaîner. Le livre est une source OBLIGATOIRE : sans
	// lui on ne saurait pas quels projets sont curés, donc quels briefs sont complets — le run
	// s'arrête plutôt que de juger sur des briefs amputés.
	type loaded struct {
		cat *catalog
		err error
	}
	index := make(chan loaded, 1)
	go func() {
		m, err := r.sources.Montagne(ctx, r.lang)
		if err != nil {
			index <- loaded{err: fmt.Errorf("%w: montagne : %w", ErrUpstream, err)}
			return
		}
		b, err := r.sources.Book(ctx, r.lang)
		if err != nil {
			index <- loaded{err: fmt.Errorf("%w: livre : %w", ErrUpstream, err)}
			return
		}
		index <- loaded{cat: newCatalog(m, b)}
	}()

	var plan planOut
	planErr := r.ask(ctx, Orchestrator, taskPlan, needBlock(r.brief), &plan)
	idx := <-index
	if planErr != nil {
		return planErr
	}
	if idx.err != nil {
		return idx.err
	}
	r.catalog = idx.cat

	// Le plan et les quatre consignes s'affichent dans le fil (cf. fit.go).
	t := &plan.Tasks
	r.fit(ctx, Orchestrator, span{text: &plan.Plan, max: messageMaxRunes}, span{text: &t.Reader, max: messageMaxRunes}, span{text: &t.Researcher, max: messageMaxRunes},
		span{text: &t.Architect, max: messageMaxRunes}, span{text: &t.Writer, max: messageMaxRunes})
	r.status(Orchestrator, "working", r.ph.get("planning"))
	r.say(string(Orchestrator), string(Orchestrator), plan.Plan, "plan")

	// L'équipe. Chaque sous-agent part DÈS que sa tâche lui est confiée (un battement entre deux
	// consignes, pour que l'œil les suive) : attendre la quatrième consigne pour lancer le premier
	// faisait perdre des battements au chemin le plus long du run. Chaque sous-agent rend son
	// livrable ; l'orchestrateur le reçoit (message de l'agent), le relaie au Rédacteur (son propre
	// message) et seulement alors le Rédacteur peut écrire la section qui en dépend.
	team := newCrew(ctx)
	needIn, archIn, proofsIn := newFuture[readerOut](), newFuture[archOut](), newFuture[researchOut]()
	assign := func(id AgentID, text string, work func(context.Context) error) {
		r.beat(ctx)
		r.say(string(Orchestrator), string(id), text, "")
		team.run(work)
	}
	assign(Reader, plan.Tasks.Reader, func(c context.Context) error {
		v, err := r.reader(c)
		if err != nil {
			return err
		}
		r.relay(Reader, v.Summary, "relay.need")
		needIn.set(v)
		return nil
	})
	assign(Researcher, plan.Tasks.Researcher, func(c context.Context) error {
		v, err := r.researcher(c)
		if err != nil {
			return err
		}
		r.relay(Researcher, r.ph.get("research.summary", len(v.Kept), len(v.Proofs)), "relay.proofs")
		proofsIn.set(v)
		return nil
	})
	assign(Architect, plan.Tasks.Architect, func(c context.Context) error {
		v, err := r.architect(c, needIn)
		if err != nil {
			return err
		}
		r.relay(Architect, v.Summary, "relay.architecture")
		archIn.set(v)
		return nil
	})
	var written dossierOut
	assign(Writer, plan.Tasks.Writer, func(c context.Context) (err error) {
		written, err = r.writer(c, needIn, archIn, proofsIn)
		return err
	})
	r.status(Orchestrator, "working", r.ph.get("waiting"))
	if err := team.wait(); err != nil {
		return err
	}

	// Contrôle : chaque preuve citée est une carte de la montagne jugée sur son brief complet
	// (brief.go) — ses chiffres, le front les relit sur la fiche.
	r.status(Orchestrator, "working", r.ph.get("checking"))
	r.say(string(Orchestrator), string(Orchestrator), r.ph.get("check.start"), "check")
	r.beat(ctx)
	r.say(string(Orchestrator), string(Orchestrator), r.ph.get("check.done", len(written.Proofs)), "check")
	r.status(Orchestrator, "done", r.ph.get("delivered"))
	r.say(string(Orchestrator), Visitor, r.ph.get("ready"), "")
	r.emit.Emit(RunCompleted{})
	return nil
}

// relay — l'agent rend compte à l'orchestrateur, qui transmet au Rédacteur. `report` tient déjà
// dans un message du fil (l'agent l'a passé par fit, ou c'est une phrase du code).
func (r *run) relay(from AgentID, report, relayKey string) {
	r.say(string(from), string(Orchestrator), report, "")
	r.say(string(Orchestrator), string(Writer), r.ph.get(relayKey), "")
}

// quote — le besoin cité dans le fil : son début, entre guillemets. Un extrait trop long s'arrête à
// une frontière de proposition ou de mot (cutAtBoundary), et l'ellipse dit alors qu'il s'agit d'un
// EXTRAIT des mots du visiteur — elle ne tombe jamais au milieu d'un mot.
func quote(need string) string {
	const max = 60
	need = strings.TrimSpace(need)
	if utf8.RuneCountInString(need) > max {
		need = cutAtBoundary(need, max-1) + "…"
	}
	return "« " + need + " »"
}

/* ------------------------------------------------------------------------------------------
   Émission — petites aides, une par forme d'événement.
   ------------------------------------------------------------------------------------------ */

func (r *run) status(a AgentID, status, activity string) {
	r.emit.Emit(AgentStatus{Agent: a, Status: status, Activity: activity})
}

func (r *run) say(from, to, text, topic string) {
	r.emit.Emit(Message{From: from, To: to, Text: text, Topic: topic})
}

func (r *run) tool(a AgentID, call string, focus ToolFocus) {
	r.emit.Emit(ToolCall{Agent: a, Call: call, Focus: &focus})
}

func (r *run) note(a AgentID, kind, text string) {
	r.emit.Emit(AgentLog{Agent: a, Kind: kind, Text: text})
}

var (
	onVisitor = ToolFocus{Kind: "visitor"}
	onDossier = ToolFocus{Kind: "dossier"}
)

func onProject(id string) ToolFocus { return ToolFocus{Kind: "project", ProjectID: id} }

// beat — le battement entre deux lignes d'un même lot (cf. en-tête).
func (r *run) beat(ctx context.Context) { r.pause(ctx, r.beatD) }

// typePause — le temps que le front met à taper un paragraphe (choreography.ts › typingDuration :
// 60 caractères/s, borné à [0,8 s ; 1,6 s]) : le bloc suivant n'arrive pas pendant la frappe.
func (r *run) typePause(ctx context.Context, text string) {
	if r.beatD == 0 {
		return
	}
	d := time.Duration(float64(utf8.RuneCountInString(text)) / 60 * float64(time.Second))
	r.pause(ctx, min(max(d, 800*time.Millisecond), 1600*time.Millisecond))
}

func (r *run) pause(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

/* ------------------------------------------------------------------------------------------
   Concurrence — l'équipe et ses livrables.
   ------------------------------------------------------------------------------------------ */

// crew — des goroutines qui partagent un contexte : la PREMIÈRE erreur annule les autres et reste
// celle du run (les suivantes ne sont que des « contexte annulé »).
type crew struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	wg     sync.WaitGroup
	once   sync.Once
	err    error
}

func newCrew(parent context.Context) *crew {
	ctx, cancel := context.WithCancelCause(parent)
	return &crew{ctx: ctx, cancel: cancel}
}

func (c *crew) run(f func(context.Context) error) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := f(c.ctx); err != nil {
			c.once.Do(func() {
				// Une goroutine qui ne fait que constater l'annulation du parent (client parti)
				// rapporte la cause du parent, pas son propre « annulé ».
				if cause := context.Cause(c.ctx); cause != nil && c.ctx.Err() != nil {
					err = cause
				}
				c.err = err
				c.cancel(err)
			})
		}
	}()
}

func (c *crew) wait() error {
	c.wg.Wait()
	c.cancel(nil)
	return c.err
}

// future — un livrable attendu par un autre agent.
type future[T any] struct {
	done chan struct{}
	v    T
}

func newFuture[T any]() *future[T] { return &future[T]{done: make(chan struct{})} }

func (f *future[T]) set(v T) {
	f.v = v
	close(f.done)
}

func (f *future[T]) wait(ctx context.Context) (T, error) {
	select {
	case <-f.done:
		return f.v, nil
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}
