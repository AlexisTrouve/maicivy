// Package dossiervivant — une équipe d'agents (un orchestrateur, quatre sous-agents) transforme le
// besoin d'un visiteur en dossier de conception, sous ses yeux, à partir des projets RÉELS
// d'Alexis (montagne + fiches maiProFiles).
//
// Rôles des fichiers (un par rôle) :
//   - events.go       — le contrat d'événements, miroir Go de frontend/lib/dossier-vivant/events.ts ;
//   - emitter.go      — émission SSE (trames, concurrence, déconnexion du client) ;
//   - callsize.go     — le plafond d'entrée d'UN appel (tarif Haiku), et ce qui y fait tenir les prompts ;
//   - tempo.go        — le temps du run, rappelé aux agents dans leur prompt (jamais un arrêt) ;
//   - mind.go         — un appel de modèle pour un agent (plafond, rappel, usage, JSON) ;
//   - tools.go        — les outils des agents : index des projets, fiches, preuves de démo ;
//   - prompts.go      — les consignes données au modèle ;
//   - phrases.go      — les lignes écrites par le CODE (statuts, compteurs), en cinq langues ;
//   - visitor.go      — le visiteur tel qu'il s'est présenté ;
//   - orchestrator.go — le run : plan, sous-agents en parallèle, relais, contrôle, remise ;
//   - reader.go, researcher.go, architect.go, writer.go — les quatre sous-agents ;
//   - diagrammer.go   — le Dessinateur, que l'Architecte lance pour dessiner le schéma (Sonnet).
package dossiervivant

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// QUOI : la liste FERMÉE des événements du flux. Règle héritée du front : un événement = une action
// visible à l'écran. Les noms de champs JSON sont ceux de events.ts, au caractère près — c'est ce
// contrat, et lui seul, que le front replie en état (runState.ts).
//
// POURQUOI une struct par événement plutôt qu'un map : le compilateur garde le contrat. Un champ
// mal orthographié ici serait un événement que le front ignore en silence.
//
// COMMENT le `type` est porté : chaque struct dit son type (kind) ; l'encodeur l'écrit en tête de
// l'objet JSON (cf. encodeEvent). Personne ne peut donc émettre un événement sans type.

// AgentID — l'équipe est FIXE.
type AgentID string

const (
	Orchestrator AgentID = "orchestrator"
	Reader       AgentID = "reader"
	Researcher   AgentID = "researcher"
	Architect    AgentID = "architect"
	// Diagrammer — AJOUT au contrat : le Dessinateur, lancé par l'Architecte via son outil
	// `draw_schema`. Il ne reçoit rien de l'orchestrateur et ne parle qu'à l'Architecte.
	Diagrammer AgentID = "diagrammer"
	Writer     AgentID = "writer"
)

// Team — l'ordre d'annonce dans run.started, qui est aussi celui des colonnes à l'écran.
var Team = []AgentID{Orchestrator, Reader, Researcher, Architect, Diagrammer, Writer}

// Visitor est l'interlocuteur `visitor` des messages et l'origine des filets.
const Visitor = "visitor"

// Event — tout événement du contrat.
type Event interface{ kind() string }

// VisitorBrief — le visiteur dont le besoin lance le run.
type VisitorBrief struct {
	Name         string `json:"name"`
	Affiliation  string `json:"affiliation"`
	Organisation string `json:"organisation"`
	Initials     string `json:"initials"`
	Need         string `json:"need"`
}

type AgentModel struct {
	ID    AgentID `json:"id"`
	Model string  `json:"model,omitempty"`
}

type RunStarted struct {
	Visitor VisitorBrief `json:"visitor"`
	Agents  []AgentModel `json:"agents"`
}

type AgentStatus struct {
	Agent    AgentID `json:"agent"`
	Status   string  `json:"status"` // waiting | working | done
	Activity string  `json:"activity,omitempty"`
}

type Message struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Text  string `json:"text"`
	Topic string `json:"topic,omitempty"` // plan | check
}

// ToolFocus — où regarde l'agent pendant l'appel : `project` (sa carte), `visitor`, `dossier`.
type ToolFocus struct {
	Kind      string `json:"kind"`
	ProjectID string `json:"projectId,omitempty"`
}

type ToolCall struct {
	Agent AgentID    `json:"agent"`
	Call  string     `json:"call"`
	Focus *ToolFocus `json:"focus,omitempty"`
}

type AgentLog struct {
	Agent AgentID `json:"agent"`
	Kind  string  `json:"kind"` // note | question | done
	Text  string  `json:"text"`
}

type IndexFiltered struct {
	Agent      AgentID  `json:"agent"`
	Total      int      `json:"total"`
	Candidates []string `json:"candidates"`
}

type ProjectVerdict struct {
	Agent     AgentID `json:"agent"`
	ProjectID string  `json:"projectId"`
	Verdict   string  `json:"verdict"` // keep | drop
	Note      string  `json:"note,omitempty"`
}

type LinkTraced struct {
	From string `json:"from"` // toujours "visitor"
	To   string `json:"to"`
}

// Usage — compteurs CUMULÉS d'un agent ; le coût est calculé ici, le front ne connaît aucun tarif.
type Usage struct {
	Agent        AgentID `json:"agent"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUsd      float64 `json:"costUsd"`
}

// Section ids, dans l'ordre de lecture du dossier.
const (
	SecNeed         = "need"
	SecArchitecture = "architecture"
	SecProofs       = "proofs"
	SecContribution = "contribution"
	SecNext         = "next"
)

type SectionPlan struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	WaitingFor string `json:"waitingFor"` // reader | researcher | architect | writer | previous
}

type DossierOpened struct {
	Sections []SectionPlan `json:"sections"`
}

type DossierSection struct {
	Section string `json:"section"`
	Status  string `json:"status"` // waiting | writing | done
}

type PipelineNode struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Col      int    `json:"col"`
	Row      int    `json:"row"`
	Human    bool   `json:"human,omitempty"`
}

type PipelineEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Dashed bool   `json:"dashed,omitempty"`
	Step   int    `json:"step,omitempty"`
}

type Demo struct {
	Kind string `json:"kind"` // video | site
	URL  string `json:"url"`
}

// Block — un bloc du dossier (union de events.ts › DossierBlock, discriminée par Type). Un bloc
// `proof` ne porte AUCUN chiffre : le front lit statut, lignes et tests sur la fiche réelle.
type Block struct {
	Type       string         `json:"type"` // lead | paragraph | item | pipeline | proof | also | contact
	Text       string         `json:"text,omitempty"`
	Nodes      []PipelineNode `json:"nodes,omitempty"`
	Edges      []PipelineEdge `json:"edges,omitempty"`
	Journal    string         `json:"journal,omitempty"`
	Label      string         `json:"label,omitempty"`
	ProjectID  string         `json:"projectId,omitempty"`
	Why        string         `json:"why,omitempty"`
	Demo       *Demo          `json:"demo,omitempty"`
	ProjectIDs []string       `json:"projectIds,omitempty"`
}

type DossierBlock struct {
	Section string `json:"section"`
	Block   Block  `json:"block"`
}

// RunCompleted — le dossier est remis. `DossierID` : AJOUT au contrat — l'identifiant de la copie que
// le serveur garde du dossier (record.go, services/dossier_store.go). C'est la seule chose que le
// navigateur renvoie pour obtenir le PDF du run : le contenu imprimé est celui du serveur. Absent
// si la copie n'a pas pu être gardée (le PDF répond alors « dossier inconnu », nommé à l'écran).
type RunCompleted struct {
	DossierID string `json:"dossierId,omitempty"`
}

// Raisons d'arrêt d'un run (RunFailed.Reason). Chacune a son libellé côté front. Ni la consommation
// ni la durée d'un run n'en sont : elles ne l'arrêtent plus (callsize.go, tempo.go).
const (
	FailModel    = "model"    // le modèle a rendu une sortie inutilisable (tronquée, JSON invalide…)
	FailUpstream = "upstream" // la passerelle ou une source de données a échoué
)

// RunFailed — AJOUT au contrat de l'étape 1 : un run réel peut s'arrêter avant la remise sur une
// VRAIE erreur (sortie inutilisable, panne), et cet arrêt doit se VOIR, nommé, jamais silencieux.
type RunFailed struct {
	Reason string `json:"reason"`
}

func (RunStarted) kind() string     { return "run.started" }
func (AgentStatus) kind() string    { return "agent.status" }
func (Message) kind() string        { return "message" }
func (ToolCall) kind() string       { return "tool.call" }
func (AgentLog) kind() string       { return "agent.log" }
func (IndexFiltered) kind() string  { return "index.filtered" }
func (ProjectVerdict) kind() string { return "project.verdict" }
func (LinkTraced) kind() string     { return "link.traced" }
func (Usage) kind() string          { return "usage" }
func (DossierOpened) kind() string  { return "dossier.opened" }
func (DossierSection) kind() string { return "dossier.section" }
func (DossierBlock) kind() string   { return "dossier.block" }
func (RunCompleted) kind() string   { return "run.completed" }
func (RunFailed) kind() string      { return "run.failed" }

// EventType — le type d'un événement, tel qu'il part dans `event:` et dans le champ `type`.
func EventType(e Event) string { return e.kind() }

// encodeEvent — l'objet JSON de l'événement, `type` en tête : `{"type":"usage","agent":…}`.
func encodeEvent(e Event) ([]byte, error) {
	body, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("dossier vivant: encodage de %s : %w", e.kind(), err)
	}
	head := []byte(`{"type":` + fmt.Sprintf("%q", e.kind()))
	if bytes.Equal(body, []byte("{}")) {
		return append(head, '}'), nil
	}
	return append(append(head, ','), body[1:]...), nil
}
