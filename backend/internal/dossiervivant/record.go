package dossiervivant

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"maicivy/internal/services"
)

// record.go — la copie SERVEUR du dossier qu'un run remet.
//
// QUOI : RecordingEmitter s'intercale entre le run et le flux SSE. Il laisse passer chaque événement
// tel quel et en replie, au passage, ce qui fait le dossier : le visiteur (run.started), le plan des
// sections (dossier.opened), les blocs (dossier.block), la durée (du premier agent au travail à la
// remise) et le coût (derniers compteurs `usage`). À run.completed, il fait garder ce dossier
// (DossierSaver → Redis) et ajoute son identifiant à l'événement : `run.completed.dossierId`.
//
// POURQUOI : le PDF public d'un run imprime cette copie, et elle seule. Le navigateur n'en renvoie
// que l'identifiant ; un corps forgé ne peut donc plus rien faire imprimer sous le nom d'Alexis.
// Le texte du visiteur (besoin, « nom, rôle ») n'y entre que tel que le run l'a validé (ParseVisitor).
//
// COMMENT le repli reste celui de l'écran (runState.ts) : mêmes règles — blocs reçus seulement une
// fois le dossier ouvert, sections dans l'ordre de lecture avec le titre de leur plan, durée mesurée
// comme elapsedAt, coût sommé comme totalsAt (même ordre d'agents, donc la même somme flottante).
// Seul un run arrivé à run.completed est gardé : un run.failed, une connexion tombée ne laissent
// aucun dossier imprimable. Le verrou couvre repli ET transmission : l'ordre replié est l'ordre du
// flux.

// DossierSaver — où un dossier remis est gardé (services.DossierStore).
type DossierSaver interface {
	Save(ctx context.Context, rec *services.DossierRecord) (string, error)
}

type RecordingEmitter struct {
	ctx   context.Context
	next  Emitter
	saver DossierSaver
	now   func() time.Time

	mu        sync.Mutex
	visitor   services.DossierRecordVisitor
	agents    int
	firstWork time.Time
	costs     map[AgentID]float64
	opened    bool
	titles    map[string]string
	blocks    map[string][]services.DossierRecordBlock
}

// NewRecordingEmitter — `ctx` : celui de la connexion (la sauvegarde se fait pendant le run).
func NewRecordingEmitter(ctx context.Context, next Emitter, saver DossierSaver) *RecordingEmitter {
	return newRecordingEmitter(ctx, next, saver, time.Now)
}

func newRecordingEmitter(ctx context.Context, next Emitter, saver DossierSaver, now func() time.Time) *RecordingEmitter {
	return &RecordingEmitter{ctx: ctx, next: next, saver: saver, now: now, costs: map[AgentID]float64{},
		titles: map[string]string{}, blocks: map[string][]services.DossierRecordBlock{}}
}

func (r *RecordingEmitter) Emit(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch ev := e.(type) {
	case RunStarted:
		r.visitor = services.DossierRecordVisitor{Name: ev.Visitor.Name, Affiliation: ev.Visitor.Affiliation, Organisation: ev.Visitor.Organisation}
		r.agents = len(ev.Agents)
	case AgentStatus:
		if ev.Status == "working" && r.firstWork.IsZero() {
			r.firstWork = r.now()
		}
	case Usage:
		r.costs[ev.Agent] = ev.CostUsd
	case DossierOpened:
		r.opened = true
		for _, s := range ev.Sections {
			r.titles[s.ID] = s.Title
		}
	case DossierBlock:
		if r.opened {
			r.blocks[ev.Section] = append(r.blocks[ev.Section], recordBlock(ev.Block))
		}
	case RunCompleted:
		e = RunCompleted{DossierID: r.save()}
	}
	r.next.Emit(e)
}

// save — garde le dossier replié ; rend son identifiant, ou "" si la sauvegarde a échoué (le run
// reste remis à l'écran ; seul le PDF répondra « dossier inconnu »).
func (r *RecordingEmitter) save() string {
	rec := &services.DossierRecord{Visitor: r.visitor, Agents: r.agents}
	if !r.firstWork.IsZero() {
		rec.ElapsedSeconds = r.now().Sub(r.firstWork).Seconds()
	}
	for _, id := range Team {
		rec.CostUsd += r.costs[id]
	}
	for _, id := range services.DossierSectionIDs {
		blocks := r.blocks[id]
		if blocks == nil {
			blocks = []services.DossierRecordBlock{}
		}
		rec.Sections = append(rec.Sections, services.DossierRecordSection{ID: id, Title: r.titles[id], Blocks: blocks})
	}
	id, err := r.saver.Save(r.ctx, rec)
	if err != nil {
		log.Error().Err(err).Msg("dossier vivant: dossier remis non gardé — son PDF sera indisponible")
		return ""
	}
	return id
}

// recordBlock — un bloc du flux dans la forme de la copie serveur (mêmes champs, même JSON).
func recordBlock(b Block) services.DossierRecordBlock {
	out := services.DossierRecordBlock{Type: b.Type, Text: b.Text, Journal: b.Journal, Label: b.Label,
		ProjectID: b.ProjectID, Why: b.Why, ProjectIDs: b.ProjectIDs}
	for _, n := range b.Nodes {
		out.Nodes = append(out.Nodes, services.DossierPipelineNode{ID: n.ID, Title: n.Title, Subtitle: n.Subtitle, Col: n.Col, Row: n.Row, Human: n.Human})
	}
	for _, e := range b.Edges {
		out.Edges = append(out.Edges, services.DossierPipelineEdge{From: e.From, To: e.To, Dashed: e.Dashed, Step: e.Step})
	}
	if b.Demo != nil {
		out.Demo = &services.DossierPDFDemo{Kind: b.Demo.Kind, URL: b.Demo.URL}
	}
	return out
}
