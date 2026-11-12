package dossiervivant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"maicivy/internal/services"
)

// QUOI : UN appel de modèle pour UN agent — vérifier que l'appel tient sous le plafond d'entrée,
// joindre le rappel de tempo, appeler la passerelle, publier l'usage cumulé de l'agent, décoder la
// réponse JSON.
//
// POURQUOI un seul chemin : c'est le seul endroit où un token peut être dépensé. Le plafond par appel
// (callsize.go), le compteur affiché et les métriques IA y sont donc tenus ensemble — impossible de
// dépenser sans compter, ou d'émettre un appel trop gros.
//
// COMMENT les échecs sont nommés : passerelle en échec → ErrUpstream ; sortie tronquée
// (stop_reason max_tokens) ou JSON indécodable → ErrModelOutput. Le run les traduit en `run.failed`
// (orchestrator.go › failReason). Ni la consommation d'un run ni sa durée n'en sont : le tempo
// (tempo.go) se dit dans le prompt, il n'arrête rien.

// Completer — ce que les agents demandent au client IA (services.AIService.Complete : une seule
// passerelle, ni `temperature` ni `budget_tokens`, blocs `text` seulement).
type Completer interface {
	Complete(ctx context.Context, req services.CompletionRequest) (*services.Completion, error)
}

var (
	ErrUpstream    = errors.New("dossier vivant: source ou passerelle en échec")
	ErrModelOutput = errors.New("dossier vivant: sortie du modèle inutilisable")
)

// meter — usage cumulé par agent, et total du run.
type meter struct {
	mu  sync.Mutex
	per map[AgentID]*Usage
}

func newMeter() *meter { return &meter{per: map[AgentID]*Usage{}} }

// add — ajoute un appel et rend le CUMUL de l'agent (c'est ce que porte l'événement `usage`).
func (m *meter) add(agent AgentID, in, out int, cost float64) Usage {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.per[agent]
	if !ok {
		u = &Usage{Agent: agent}
		m.per[agent] = u
	}
	u.InputTokens += in
	u.OutputTokens += out
	u.CostUsd += cost
	return *u
}

// of — le cumul d'un agent (zéro s'il n'a rien consommé).
func (m *meter) of(agent AgentID) Usage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.per[agent]; ok {
		return *u
	}
	return Usage{Agent: agent}
}

// totals — tokens d'entrée, de sortie et coût de tout le run.
func (m *meter) totals() (in, out int, cost float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.per {
		in += u.InputTokens
		out += u.OutputTokens
		cost += u.CostUsd
	}
	return in, out, cost
}

// ask — `agent` exécute la tâche `t` sur `prompt`, sur le modèle des agents ; la réponse JSON est
// décodée dans `out`.
func (r *run) ask(ctx context.Context, agent AgentID, t task, prompt string, out any) error {
	return r.complete(ctx, agent, call{task: t.name, model: r.model, effort: agentEffort, system: systemPrompt(t, r.lang), prompt: prompt, maxTokens: t.maxTokens}, out)
}

// agentEffort — la réflexion demandée aux agents Haiku : `low`.
//
// POURQUOI : sans réglage, claude-haiku-5-5 réfléchit à sa guise, et c'est là que passait le temps
// d'un run. Mesuré le 2026-11-07 sur des verdicts réels (même fiche, même consigne) : 570 à 1 121
// tokens de sortie sans réglage contre 100 à 134 en `low`, pour les mêmes décisions ; 6,5 à 13,1 s
// contre 2,8 à 7,8 s. Et deux runs réels sur trois s'arrêtaient sur une sortie TRONQUÉE (plan,
// section 1) : la réflexion avait brûlé les 1 500 tokens du plafond. Le modèle est inchangé ; seul
// le Dessinateur (Sonnet) garde son effort moyen — le schéma est le seul endroit où le jugement se voit.
const agentEffort = "low"

// call — un appel de modèle entièrement décrit. `ask` en remplit un pour les tâches Haiku ; le
// Dessinateur (diagrammer.go) le sien, avec son modèle, son effort et son délai d'appel.
type call struct {
	task      string // nom de la tâche, pour les erreurs nommées
	model     string
	effort    string        // vide = aucun réglage de réflexion envoyé
	timeout   time.Duration // 0 = délai par défaut du client IA
	system    string
	prompt    string
	maxTokens int
}

// complete — LE chemin d'un appel : joindre le rappel de tempo, vérifier le plafond d'entrée, appeler,
// publier l'usage de l'agent, décoder.
func (r *run) complete(ctx context.Context, agent AgentID, k call, out any) error {
	k.prompt += r.tempo.reminder()
	if err := checkCallSize(agent, k); err != nil {
		return err
	}
	c, callErr := r.ai.Complete(ctx, services.CompletionRequest{Model: k.model, System: k.system, Prompt: k.prompt, MaxTokens: k.maxTokens, Effort: k.effort, Timeout: k.timeout})
	if c.Metrics.TokensInput+c.Metrics.TokensOutput > 0 {
		r.emit.Emit(r.meter.add(agent, c.Metrics.TokensInput, c.Metrics.TokensOutput, c.Metrics.EstimatedCost))
	}
	switch {
	// Avant l'erreur d'appel : un modèle à réflexion qui épuise son plafond EN RÉFLÉCHISSANT rend
	// un bloc `thinking` et aucun bloc `text` — le client y voit « pas de texte », mais la vraie
	// cause est la troncature (mesuré au premier run réel : architect/refs, 400 tokens de sortie).
	case c.StopReason == "max_tokens":
		return fmt.Errorf("%w: %s/%s tronquée à %d tokens de sortie", ErrModelOutput, agent, k.task, k.maxTokens)
	case callErr != nil:
		return fmt.Errorf("%w: %s/%s : %w", ErrUpstream, agent, k.task, callErr)
	}
	if err := decodeJSON(c.Text, out); err != nil {
		return fmt.Errorf("%w: %s/%s : %v", ErrModelOutput, agent, k.task, err)
	}
	return nil
}

// decodeJSON — le PREMIER objet JSON de la réponse. Tolère une clôture de code autour (le modèle en
// met parfois malgré la consigne) et du texte après l'objet : mesuré sur des réécritures réelles
// (fit.go), le modèle ajoute parfois une seconde pensée après son objet (« invalid character 'W'
// after top-level value » quand on lisait jusqu'à la dernière accolade).
func decodeJSON(text string, out any) error {
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return errors.New("aucun objet JSON dans la réponse")
	}
	return json.NewDecoder(strings.NewReader(escapeControlsInStrings(text[start:]))).Decode(out)
}

// escapeControlsInStrings — échappe les caractères de contrôle BRUTS (< 0x20) trouvés DANS une
// chaîne JSON.
//
// POURQUOI : panne réelle du 2026-11-07 en prod — le Lecteur a rendu un retour à la ligne brut au
// milieu d'une chaîne (« invalid character '\n' in string literal ») et le run du visiteur s'est
// arrêté. Un contrôle dans une chaîne n'a qu'une lecture possible : c'est le caractère lui-même.
// Ce n'est pas un repli qui masque une erreur : un JSON réellement cassé (chaîne non fermée,
// littéral faux) échoue toujours au décodage.
//
// COMMENT : un seul passage octet par octet, qui suit deux états — dans une chaîne ou non, et
// « l'octet précédent était un antislash » (pour ne pas prendre \" pour une fin de chaîne). Hors
// chaîne, rien n'est touché : un saut de ligne y est un blanc légal. Les octets UTF-8 multi-octets
// sont tous ≥ 0x80, donc jamais pris pour un contrôle.
func escapeControlsInStrings(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inString, escaped := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case !inString:
			if c == '"' {
				inString = true
			}
			b.WriteByte(c)
		case escaped:
			escaped = false
			b.WriteByte(c)
		case c == '\\':
			escaped = true
			b.WriteByte(c)
		case c == '"':
			inString = false
			b.WriteByte(c)
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\r':
			b.WriteString(`\r`)
		case c == '\t':
			b.WriteString(`\t`)
		case c < 0x20:
			fmt.Fprintf(&b, `\u%04x`, c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
