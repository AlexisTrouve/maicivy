package dossiervivant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
	"maicivy/internal/services"
)

// QUOI : un run complet de l'équipe, contre une FAUSSE passerelle (motif ai_fallback_test.go) et
// une montagne en mémoire. Le vrai AIService y est pointé : le chemin réseau, le décodage des blocs
// `text`, le calcul du coût et l'enregistrement des métriques sont ceux de la production.
//
// Ce qui est verrouillé : la séquence d'événements conforme au contrat, le parallélisme réel des
// sous-agents (appels en vol simultanés côté passerelle), le rejet des ids inventés par le modèle,
// l'arrêt nommé sur sortie tronquée (cf. tempo_run_test.go : aucun arrêt sur budget ni sur durée).

const (
	testModel       = "claude-haiku-5-5"
	testSchemaModel = "claude-sonnet-5-5"
)

// fakeGateway — répond selon la tâche lue en tête du prompt système (`dossier-vivant/<tâche>`).
type fakeGateway struct {
	srv  *httptest.Server
	hold time.Duration // durée de chaque requête : laisse le temps aux appels parallèles de se chevaucher
	// inputTokens — tokens d'entrée déclarés par réponse (le budget se règle dessus).
	inputTokens int
	// truncate — tâche dont la réponse revient tronquée (stop_reason max_tokens).
	truncate string
	// answers — réponse imposée à une tâche, à la place de cannedAnswer (ex. un schéma hors contrat).
	answers map[string]any

	mu          sync.Mutex
	tasks       []string
	prompts     []string // message utilisateur de chaque appel, dans l'ordre d'arrivée
	inFlight    int
	maxInFlight int
	bodies      []map[string]any
}

func newFakeGateway(t *testing.T) *fakeGateway {
	t.Helper()
	g := &fakeGateway{inputTokens: 120}
	g.srv = httptest.NewServer(http.HandlerFunc(g.serve))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *fakeGateway) serve(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	system := body["system"].([]any)[0].(map[string]any)["text"].(string)
	prompt := body["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	task := strings.TrimPrefix(strings.SplitN(system, "\n", 2)[0], "dossier-vivant/")

	g.mu.Lock()
	call := len(g.tasks)
	g.tasks = append(g.tasks, task)
	g.prompts = append(g.prompts, prompt)
	g.bodies = append(g.bodies, body)
	g.inFlight++
	g.maxInFlight = max(g.maxInFlight, g.inFlight)
	g.mu.Unlock()
	time.Sleep(g.hold)
	g.mu.Lock()
	g.inFlight--
	g.mu.Unlock()

	stop := "end_turn"
	if task == g.truncate {
		stop = "max_tokens"
	}
	answer := cannedAnswer(task, prompt, call)
	if a, ok := g.answers[task]; ok {
		answer = a
	}
	text, _ := json.Marshal(answer)
	content := []any{map[string]any{"type": "text", "text": string(text)}}
	if task == g.truncate {
		// Forme RÉELLE d'une troncature sur un modèle à réflexion : tout le plafond brûlé en
		// réflexion, aucun bloc `text` (mesuré au premier run réel).
		content = []any{map[string]any{"type": "thinking", "thinking": "je réfléchis…", "signature": "sig"}}
	}
	resp, _ := json.Marshal(map[string]any{
		"id": "msg_fake", "type": "message", "role": "assistant", "model": body["model"],
		"content":     content,
		"stop_reason": stop,
		"usage":       map[string]any{"input_tokens": g.inputTokens, "output_tokens": 40},
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

func (g *fakeGateway) calls() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.tasks...)
}

// cannedAnswer — une réponse plausible par tâche. Le filtre et l'Architecte glissent chacun un id
// INVENTÉ : le code doit l'écarter. Le filtre rend plus de candidats que le Chercheur n'ouvre de
// fiches : c'est le cas où un candidat pourrait être tranché sans avoir été lu.
//
// `call` — rang de l'appel côté passerelle. Les textes qui deviennent une note de verdict ou une
// raison de preuve le portent (`appel-N`) : un événement du flux désigne ainsi l'appel qui l'a
// DÉCIDÉ, et le test vérifie ce que le modèle avait sous les yeux dans CET appel.
func cannedAnswer(task, prompt string, call int) any {
	tag := fmt.Sprintf("appel-%d", call)
	switch task {
	case "plan":
		return map[string]any{"plan": "quatre tâches en parallèle.", "tasks": map[string]string{
			"reader": "reformule le besoin.", "researcher": "trouve les preuves.", "architect": "dessine la chaîne.", "writer": "monte le dossier."}}
	case "reader":
		return map[string]any{"notes": []string{"1 400 tickets par jour", "six personnes", "trace de chaque décision"},
			"gaps": []string{"Quel outil de ticketing ?"}, "summary": "besoin reformulé, une question."}
	case "filter":
		return map[string]any{"candidates": append([]string{"projet_invente"}, memProjects...)}
	case "verdict":
		if strings.Contains(prompt, "id: liveconf") {
			return map[string]any{"verdict": "drop", "reason": tag + " aucune délégation", "why": "", "score": 10}
		}
		return map[string]any{"verdict": "keep", "reason": tag + " agents en production", "why": tag + " Des agents en production.", "score": 80}
	case "refs":
		return map[string]any{"refs": []string{"autre_invente", "envoia"}}
	case "lesson":
		return map[string]any{"useful": true, "lesson": tag + " planifier, puis exécuter sans LLM"}
	case "design":
		return map[string]any{
			"stages":     []string{"le ticket entre, cité tel quel", "un modèle léger le trie", "le répondeur cite la base", "un humain reprend les cas sensibles"},
			"branches":   []string{"doute du trieur → file humaine"},
			"guardrails": []string{"Le ticket est cité, jamais exécuté.", "Contrôle sans LLM.", "Humain sur règle."},
			"check":      "Couvre les trois exigences", "summary": "chaîne en quatre étapes.",
		}
	case "schema":
		return cannedSchema
	case "section-need":
		return map[string]any{"lead": "Traiter 1 400 tickets par jour à six.", "items": []string{"Source citée.", "Humain sur règle.", "Trace lisible."}}
	case "closing":
		return map[string]any{"contribution": []string{"L’architecture.", "Les garde-fous.", "Le coût mesuré."}, "next": "Un appel de 30 minutes, puis un prototype."}
	case "shorten":
		// Un modèle qui tient la consigne : un texte court par texte numéroté du message (fit.go).
		n := len(regexp.MustCompile(`(?m)^\d+\. \[`).FindAllString(prompt, -1))
		texts := make([]string, n)
		for i := range texts {
			texts[i] = fmt.Sprintf("Réécrit %d.", i+1)
		}
		return map[string]any{"texts": texts}
	}
	return map[string]any{}
}

// cannedSchema — la réponse du Dessinateur. Ses ids (`sn-*`) n'existent nulle part ailleurs : un
// bloc schéma qui les porte vient de CETTE réponse, et d'aucune autre.
var cannedSchema = map[string]any{
	"notes": []string{"Le modèle trie, le code tranche", "L'humain voit le doute, pas tout"},
	"nodes": []map[string]any{
		{"id": "sn-ticket", "title": "Ticket", "subtitle": "cité, jamais exécuté", "col": 0, "row": 0},
		{"id": "sn-triage", "title": "Trieur", "subtitle": "modèle léger", "col": 1, "row": 0},
		{"id": "sn-answer", "title": "Répondeur", "subtitle": "cite la base", "col": 2, "row": 0},
		{"id": "sn-human", "title": "File humaine", "subtitle": "brief joint", "col": 1, "row": 2, "human": true},
	},
	"edges": []map[string]any{
		{"from": "sn-ticket", "to": "sn-triage", "step": 1},
		{"from": "sn-triage", "to": "sn-answer", "step": 2},
		{"from": "sn-triage", "to": "sn-human", "dashed": true, "step": 3},
	},
	"journal": "Journal des décisions", "label": "Chaîne proposée, du ticket à la réponse",
	"summary": "schéma en quatre nœuds, trois garde-fous posés.",
}

// memSources — la montagne, les fiches et le livre, en mémoire.
type memSources struct{}

// memProjects — les cartes de la montagne. Dix : plus que le Chercheur n'ouvre de fiches.
var memProjects = []string{"melodicode", "envoia", "liveconf", "maicivy", "veracite", "aria", "wanmira", "feedgen", "orchestre", "studio"}

// memCurated — les projets qui ont une entrée dans le livre (couche curée).
var memCurated = map[string]bool{"melodicode": true, "envoia": true, "veracite": true, "studio": true}

func (memSources) Montagne(context.Context, string) (*services.Montagne, error) {
	m := &services.Montagne{}
	for _, id := range memProjects {
		m.Nodes = append(m.Nodes, services.MontagneNoeud{ID: id, Name: strings.ToUpper(id[:1]) + id[1:], Mini: "mini " + id, Status: "production"})
	}
	video := services.MontagnePreuve{MontagneVisuel: services.MontagneVisuel{Kind: "video", URL: "/images/demo-melodicode"}}
	m.Summit = []services.MontagneCarte{{ID: "melodicode", Proofs: []services.MontagnePreuve{video}}}
	return m, nil
}

// memStory — un récit `portfolio` à la taille RÉELLE (médiane mesurée sur les 75 fiches de
// maiProFiles : ~2 800 caractères), dont la FIN porte une marque propre au projet : une fiche
// rognée la perd, et le test le voit.
func memStory(id string) string {
	return "# " + id + "\n" + strings.Repeat("Planificateur, validateur sans LLM, exécuteur ; trace de chaque décision. ", 38) +
		"Valeur transférable : fin-du-recit-" + id
}

func (memSources) Project(_ context.Context, id, _ string) (*services.MPFProject, error) {
	p := &services.MPFProject{ID: id, Status: "production", Stats: services.MPFStats{LOC: 1000, Tests: 100}, Stack: []string{"go", "redis"}}
	p.Description.Short = "court " + id
	p.Description.Portfolio = memStory(id)
	return p, nil
}

// memBookEntry — l'entrée du livre d'un projet curé : deux paragraphes de récit et un chiffre.
func memBookEntry(id string) services.MPFBookProject {
	return services.MPFBookProject{ID: id, Name: id, Category: "Agents IA", IndexDomain: "domaine " + id,
		Paragraphs: []string{"livre-paragraphe-1-" + id, "livre-paragraphe-2-" + id},
		Stats:      []services.MPFStat{{Value: "9 000+", Label: " lignes-" + id}}}
}

func (memSources) Book(context.Context, string) (*services.MPFBook, error) {
	b := &services.MPFBook{Lang: "fr"}
	for _, id := range memProjects {
		if memCurated[id] {
			b.Projects = append(b.Projects, memBookEntry(id))
		}
	}
	return b, nil
}

// memBriefPieces — ce que le brief COMPLET de `id` doit faire entrer dans l'appel : le récit
// entier de la fiche, son résumé, et pour un projet curé chaque paragraphe et chiffre du livre.
func memBriefPieces(id string) []string {
	pieces := []string{memStory(id), "court " + id}
	if memCurated[id] {
		e := memBookEntry(id)
		pieces = append(pieces, e.Paragraphs...)
		pieces = append(pieces, e.Stats[0].Label)
	}
	return pieces
}

// recorder — émetteur qui garde tout, dans l'ordre.
type recorder struct {
	mu     sync.Mutex
	events []Event
}

func (r *recorder) Emit(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recorder) all() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Event(nil), r.events...)
}

// sse — tout le flux tel qu'il partirait sur le fil.
func (r *recorder) sse(t *testing.T) string {
	var b bytes.Buffer
	for _, e := range r.all() {
		f, err := FormatSSE(e)
		require.NoError(t, err)
		b.Write(f)
	}
	return b.String()
}

func newTestService(t *testing.T, g *fakeGateway) *Service {
	t.Helper()
	ai, err := services.NewAIService(&config.AIConfig{
		AnthropicAPIKey: "eai_test", AnthropicBaseURL: g.srv.URL, ClaudeModel: "inutilise",
		FallbackModel:        "gpt-5.6-terra", // présent pour prouver que Complete ne s'en sert JAMAIS
		MaxRequestsPerMinute: 1, MaxTokensPerRequest: 4000, RequestTimeout: 5 * time.Second,
	}, nil)
	require.NoError(t, err)
	return NewService(ai, memSources{}, Options{Model: testModel, SchemaModel: testSchemaModel, Beat: -1})
}

var tessera = VisitorBrief{Name: "Inès Morel", Affiliation: "CTO chez Tessera", Organisation: "Tessera", Initials: "IM",
	Need: "Six personnes traitent 1 400 tickets support par jour ; nous voulons des agents de tri."}

func runOnce(t *testing.T, s *Service) (*recorder, error) {
	t.Helper()
	rec := &recorder{}
	err := s.Run(context.Background(), tessera, "fr", rec)
	return rec, err
}

func TestRun_SequenceConformeAuContrat(t *testing.T) {
	g := newFakeGateway(t)
	rec, err := runOnce(t, newTestService(t, g))
	require.NoError(t, err)
	events := rec.all()

	// Bornes du flux : run.started (modèle annoncé pour chaque agent : le Dessinateur sur le sien),
	// run.completed, rien après.
	started, ok := events[0].(RunStarted)
	require.True(t, ok, "le premier événement doit être run.started")
	require.Len(t, started.Agents, len(Team))
	for _, a := range started.Agents {
		want := testModel
		if a.ID == Diagrammer {
			want = testSchemaModel
		}
		assert.Equal(t, want, a.Model, "modèle annoncé pour %s", a.ID)
	}
	assert.Equal(t, RunCompleted{}, events[len(events)-1])
	for _, e := range events {
		_, failed := e.(RunFailed)
		assert.False(t, failed)
	}

	// Chaque section : ouverte (writing) → blocs → close (done), après dossier.opened.
	opened, writing, done := -1, map[string]int{}, map[string]int{}
	usage := map[AgentID]int{}
	for i, e := range events {
		switch ev := e.(type) {
		case DossierOpened:
			opened = i
			require.Len(t, ev.Sections, 5)
		case DossierSection:
			require.Greater(t, i, opened)
			if ev.Status == "writing" {
				writing[ev.Section] = i
			} else {
				done[ev.Section] = i
			}
		case DossierBlock:
			w, ok := writing[ev.Section]
			require.True(t, ok, "bloc %s avant l'ouverture de sa section", ev.Section)
			assert.Greater(t, i, w)
			_, closed := done[ev.Section]
			assert.False(t, closed, "bloc %s après la clôture de sa section", ev.Section)
		case Usage:
			// Compteurs CUMULÉS : jamais en recul.
			total := ev.InputTokens + ev.OutputTokens
			assert.GreaterOrEqual(t, total, usage[ev.Agent])
			usage[ev.Agent] = total
			assert.Greater(t, ev.CostUsd, 0.0, "le coût est calculé par le producteur")
		}
	}
	for _, id := range []string{SecNeed, SecArchitecture, SecProofs, SecContribution, SecNext} {
		assert.Contains(t, done, id, "section %s jamais terminée", id)
	}
	for _, id := range Team {
		assert.Greater(t, usage[id], 0, "l'agent %s n'a rien consommé", id)
	}

	// Sur le fil : `event:` et `type` coïncident, et sont dans la liste fermée du contrat.
	known := map[string]bool{"run.started": true, "agent.status": true, "message": true, "tool.call": true, "agent.log": true,
		"index.filtered": true, "project.verdict": true, "link.traced": true, "usage": true, "dossier.opened": true,
		"dossier.section": true, "dossier.block": true, "run.completed": true, "run.failed": true}
	for _, frame := range strings.Split(strings.TrimSpace(rec.sse(t)), "\n\n") {
		lines := strings.Split(frame, "\n")
		require.Len(t, lines, 2)
		kind := strings.TrimPrefix(lines[0], "event: ")
		var payload struct{ Type string }
		require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(lines[1], "data: ")), &payload))
		assert.Equal(t, kind, payload.Type)
		assert.True(t, known[kind], "type hors contrat : %s", kind)
	}

	// Aucun chiffre de preuve dans le flux : les blocs `proof` ne portent qu'id, raison et démo.
	assert.NotContains(t, rec.sse(t), `"loc"`)
	assert.NotContains(t, rec.sse(t), `"tests"`)
	// La démo vient de la montagne (carte du sommet), pas du modèle.
	assert.Contains(t, rec.sse(t), `"demo":{"kind":"video","url":"/images/demo-melodicode"}`)
	// Les modèles annoncés, et eux seuls : Complete n'a pas de repli (le repli configuré n'est jamais
	// visé). Les appels Haiku demandent un effort BAS (mind.go › agentEffort), le Dessinateur un
	// effort moyen (cf. TestRun_SchemaDessineParLeDessinateur) ; jamais de temperature ni de
	// budget_tokens.
	for i, b := range g.bodies {
		_, hasTemp := b["temperature"]
		assert.False(t, hasTemp, "jamais de temperature (refusée par les modèles 5.x)")
		assert.NotContains(t, fmt.Sprint(b["thinking"]), "budget_tokens", "jamais budget_tokens")
		if g.calls()[i] == schemaTask {
			continue
		}
		assert.Equal(t, testModel, b["model"])
		assert.Equal(t, map[string]any{"type": "adaptive"}, b["thinking"], "réflexion adaptive sur les appels Haiku")
		assert.Equal(t, map[string]any{"effort": "low"}, b["output_config"], "effort bas sur les appels Haiku")
	}
	// Le texte du modèle suit la langue du run : la consigne la nomme.
	assert.Contains(t, g.bodies[0]["system"].([]any)[0].(map[string]any)["text"], "Write every text value in French")
}

func TestRun_SousAgentsEnParallele(t *testing.T) {
	g := newFakeGateway(t)
	g.hold = 150 * time.Millisecond
	_, err := runOnce(t, newTestService(t, g))
	require.NoError(t, err)
	// Lecteur, Chercheur et Architecte partent ensemble : leurs premiers appels sont en vol au même
	// instant côté passerelle. Séquentiels, il n'y en aurait jamais plus d'un.
	assert.GreaterOrEqual(t, g.maxInFlight, 3, "les sous-agents doivent appeler le modèle en parallèle")
}

func TestRun_IdInventeEcarte(t *testing.T) {
	g := newFakeGateway(t)
	rec, err := runOnce(t, newTestService(t, g))
	require.NoError(t, err)
	stream := rec.sse(t)
	assert.NotContains(t, stream, "projet_invente", "un id hors montagne ne doit atteindre aucun événement")
	assert.NotContains(t, stream, "autre_invente")

	// Tout projet nommé dans le flux est une carte de la montagne.
	m, _ := memSources{}.Montagne(context.Background(), "fr")
	real := map[string]bool{}
	for _, n := range m.Nodes {
		real[n.ID] = true
	}
	for _, e := range rec.all() {
		var ids []string
		switch ev := e.(type) {
		case IndexFiltered:
			ids = ev.Candidates
		case ProjectVerdict:
			ids = []string{ev.ProjectID}
		case LinkTraced:
			ids = []string{ev.To}
		case DossierBlock:
			ids = append([]string{ev.Block.ProjectID}, ev.Block.ProjectIDs...)
		case ToolCall:
			if ev.Focus != nil && ev.Focus.Kind == "project" {
				ids = []string{ev.Focus.ProjectID}
			}
		}
		for _, id := range ids {
			if id != "" {
				assert.True(t, real[id], "projet hors montagne dans le flux : %s", id)
			}
		}
	}
}

// TestRun_VerdictEtPreuveSurBriefComplet — l'invariant des briefs : aucun `project.verdict`, aucun
// bloc de preuve n'atteint le flux pour un projet dont le brief COMPLET (récit entier de la fiche +
// entrée du livre s'il est curé) n'était pas dans l'appel de modèle qui l'a décidé.
//
// COMMENT l'appel décideur est retrouvé : la note d'un verdict et la raison d'une preuve portent
// le rang de l'appel qui les a rendues (`appel-N`, cf. cannedAnswer). Un verdict sans note n'a été
// décidé par AUCUN appel — c'est le cas d'un candidat rendu au rang sans avoir été lu.
func TestRun_VerdictEtPreuveSurBriefComplet(t *testing.T) {
	g := newFakeGateway(t)
	rec, err := runOnce(t, newTestService(t, g))
	require.NoError(t, err)
	g.mu.Lock()
	prompts := append([]string(nil), g.prompts...)
	tasks := append([]string(nil), g.tasks...)
	g.mu.Unlock()

	callTag := regexp.MustCompile(`appel-(\d+)`)
	// briefedIn — l'appel désigné par `text` avait-il le brief complet de `id` sous les yeux ?
	briefedIn := func(text, id string) (int, bool) {
		m := callTag.FindStringSubmatch(text)
		if m == nil {
			return -1, false
		}
		n, _ := strconv.Atoi(m[1])
		for _, piece := range memBriefPieces(id) {
			if !strings.Contains(prompts[n], piece) {
				return n, false
			}
		}
		return n, true
	}

	keptOnBrief := map[string]bool{}
	verdicts := 0
	for _, e := range rec.all() {
		switch ev := e.(type) {
		case ProjectVerdict:
			verdicts++
			n, ok := briefedIn(ev.Note, ev.ProjectID)
			assert.True(t, ok, "verdict %s/%s sur %s rendu sans son brief complet (appel %d)", ev.Agent, ev.Verdict, ev.ProjectID, n)
			if ok {
				assert.Contains(t, []string{"verdict", "lesson"}, tasks[n], "un verdict vient d'un appel de jugement")
			}
			if ok && ev.Verdict == "keep" {
				keptOnBrief[ev.ProjectID] = true
			}
		case DossierBlock:
			switch ev.Block.Type {
			case "proof":
				n, ok := briefedIn(ev.Block.Why, ev.Block.ProjectID)
				assert.True(t, ok, "preuve %s écrite sans son brief complet (appel %d)", ev.Block.ProjectID, n)
			case "also":
				for _, id := range ev.Block.ProjectIDs {
					assert.True(t, keptOnBrief[id], "« aussi retenu » %s sans verdict rendu sur son brief", id)
				}
			}
		}
	}
	assert.GreaterOrEqual(t, verdicts, 7, "le Chercheur et l'Architecte ont bien tranché")

	// Le Rédacteur qui écrit « ce qu'Alexis apporte » cite les preuves : il a leurs briefs.
	for i, task := range tasks {
		if task != "closing" {
			continue
		}
		for _, e := range rec.all() {
			if b, ok := e.(DossierBlock); ok && b.Block.Type == "proof" {
				for _, piece := range memBriefPieces(b.Block.ProjectID) {
					assert.Contains(t, prompts[i], piece, "le Rédacteur écrit sur %s sans son brief", b.Block.ProjectID)
				}
			}
		}
	}
}

func TestRun_SortieTronqueeNommee(t *testing.T) {
	g := newFakeGateway(t)
	g.truncate = schemaTask // le Dessinateur aussi : une troncature reste un arrêt nommé
	rec, err := runOnce(t, newTestService(t, g))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelOutput), "%v", err)
	events := rec.all()
	assert.Equal(t, RunFailed{Reason: FailModel}, events[len(events)-1])
}

func TestPhrasebook_CinqLangues(t *testing.T) {
	for key, langs := range phrasebook {
		for lang := range languageNames {
			assert.NotEmpty(t, langs[lang], "phrase %q sans traduction %s", key, lang)
		}
	}
	assert.Equal(t, "en", RunLanguage("ka"), "langue non servie → anglais, jamais le français")
}

func TestParseVisitor(t *testing.T) {
	need := "Six personnes traitent 1 400 tickets par jour."
	for _, c := range []struct{ who, name, aff, org, initials string }{
		{"Inès Morel, CTO chez Tessera", "Inès Morel", "CTO chez Tessera", "Tessera", "IM"},
		{"Jane Doe, Head of Support at Acme Corp", "Jane Doe", "Head of Support at Acme Corp", "Acme Corp", "JD"},
		{"张伟，Tessera", "张伟", "Tessera", "Tessera", "张伟"},
	} {
		b, err := ParseVisitor(c.who, need)
		require.NoError(t, err, c.who)
		assert.Equal(t, VisitorBrief{Name: c.name, Affiliation: c.aff, Organisation: c.org, Initials: c.initials, Need: need}, b)
	}
	_, err := ParseVisitor("Inès Morel", need)
	assert.ErrorIs(t, err, ErrVisitorShape)
	_, err = ParseVisitor("Inès Morel, CTO", "court")
	assert.ErrorIs(t, err, ErrNeedLength)
}

// TestRun_SchemaDessineParLeDessinateur — le schéma passe par l'OUTIL de l'Architecte : son appel
// `draw_schema` lance le Dessinateur, dont l'appel est le seul à porter le modèle schéma (effort
// moyen, forme adaptive), et le bloc schéma du dossier est exactement sa réponse.
func TestRun_SchemaDessineParLeDessinateur(t *testing.T) {
	g := newFakeGateway(t)
	rec, err := runOnce(t, newTestService(t, g))
	require.NoError(t, err)
	events := rec.all()

	// 1. Un seul appel sur le modèle schéma, et c'est la tâche du Dessinateur.
	var schemaCalls []int
	for i, b := range g.bodies {
		if b["model"] == testSchemaModel {
			schemaCalls = append(schemaCalls, i)
		}
	}
	require.Len(t, schemaCalls, 1, "un seul appel porte le modèle schéma")
	n := schemaCalls[0]
	assert.Equal(t, schemaTask, g.calls()[n])
	body := g.bodies[n]
	assert.Equal(t, map[string]any{"type": "adaptive"}, body["thinking"], "réflexion adaptive, jamais budget_tokens")
	assert.Equal(t, map[string]any{"effort": "medium"}, body["output_config"], "effort moyen")
	// Ce que l'Architecte lui transmet : le besoin reformulé (Lecteur), la conception, les garde-fous numérotés.
	for _, piece := range []string{"1 400 tickets par jour", "un modèle léger le trie", "doute du trieur", "1. Le ticket est cité", "3. Humain sur règle."} {
		assert.Contains(t, g.prompts[n], piece)
	}

	// 2. L'ordre à l'écran : l'Architecte appelle son outil, le Dessinateur travaille et lui rend
	//    compte, PUIS le schéma s'écrit dans le dossier.
	idx := func(match func(Event) bool) int {
		for i, e := range events {
			if match(e) {
				return i
			}
		}
		return -1
	}
	tool := idx(func(e Event) bool {
		c, ok := e.(ToolCall)
		return ok && c.Agent == Architect && strings.HasPrefix(c.Call, "draw_schema(")
	})
	working := idx(func(e Event) bool {
		s, ok := e.(AgentStatus)
		return ok && s.Agent == Diagrammer && s.Status == "working"
	})
	usage := idx(func(e Event) bool { u, ok := e.(Usage); return ok && u.Agent == Diagrammer })
	report := idx(func(e Event) bool {
		m, ok := e.(Message)
		return ok && m.From == string(Diagrammer) && m.To == string(Architect)
	})
	block := idx(func(e Event) bool { b, ok := e.(DossierBlock); return ok && b.Block.Type == "pipeline" })
	for name, i := range map[string]int{"draw_schema": tool, "statut": working, "usage": usage, "compte rendu": report, "bloc": block} {
		require.GreaterOrEqual(t, i, 0, "événement absent : %s", name)
	}
	assert.Less(t, tool, working)
	assert.Less(t, working, usage)
	assert.Less(t, usage, report)
	assert.Less(t, report, block)
	u := events[usage].(Usage)
	// Sonnet 5.5 : $2/$10 (source corrigée le 2026-10-09, cf. services/ai_pricing.go).
	assert.InDelta(t, 120.0/1e6*2+40.0/1e6*10, u.CostUsd, 1e-12, "l'usage du Dessinateur est compté à SON tarif")

	// 3. Le bloc schéma est la réponse du Dessinateur, telle quelle (ids `sn-*`).
	p := events[block].(DossierBlock).Block
	ids := make([]string, len(p.Nodes))
	for i, nd := range p.Nodes {
		ids[i] = nd.ID
	}
	assert.Equal(t, []string{"sn-ticket", "sn-triage", "sn-answer", "sn-human"}, ids)
	assert.Equal(t, "Chaîne proposée, du ticket à la réponse", p.Label)
	assert.Len(t, p.Edges, 3)
}

// TestRun_SchemaHorsContratRejete — un schéma qui sort de la grille est refusé, NOMMÉ, et rien
// n'est affiché : le run s'arrête (raison `model`) sans bloc schéma ni dossier remis.
func TestRun_SchemaHorsContratRejete(t *testing.T) {
	g := newFakeGateway(t)
	bad := map[string]any{}
	for k, v := range cannedSchema {
		bad[k] = v
	}
	bad["edges"] = []map[string]any{{"from": "sn-ticket", "to": "sn-inconnu", "step": 1}}
	g.answers = map[string]any{schemaTask: bad}
	rec, err := runOnce(t, newTestService(t, g))

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSchemaRejected)
	assert.ErrorIs(t, err, ErrModelOutput)
	events := rec.all()
	assert.Equal(t, RunFailed{Reason: FailModel}, events[len(events)-1])
	rejected := false
	for _, e := range events {
		if b, ok := e.(DossierBlock); ok {
			assert.NotEqual(t, "pipeline", b.Block.Type, "un schéma refusé n'est jamais affiché")
		}
		if l, ok := e.(AgentLog); ok && l.Agent == Diagrammer && l.Text == phrases("fr").get("diag.rejected") {
			rejected = true
		}
	}
	assert.True(t, rejected, "le refus se voit dans la colonne du Dessinateur")
}

func TestDrawable_RefuseUnSchemaHorsContrat(t *testing.T) {
	base := func() []PipelineNode {
		return []PipelineNode{{ID: "a", Title: "A"}, {ID: "b", Title: "B", Col: 1}, {ID: "c", Title: "C", Col: 1, Row: 1}}
	}
	ok := []PipelineEdge{{From: "a", To: "b", Step: 1}, {From: "b", To: "c"}}
	for name, c := range map[string]struct {
		nodes      func([]PipelineNode) []PipelineNode
		edges      []PipelineEdge
		guardrails int
	}{
		"deux nœuds dans la même case": {func(n []PipelineNode) []PipelineNode { n[2].Row = 0; return n }, ok, 1},
		"hors grille":                  {func(n []PipelineNode) []PipelineNode { n[2].Col = 6; return n }, ok, 1},
		"id hors contrat":              {func(n []PipelineNode) []PipelineNode { n[0].ID = "a b"; return n }, ok, 1},
		"arête vers un nœud absent":    {nil, []PipelineEdge{{From: "a", To: "z", Step: 1}, {From: "b", To: "c"}}, 1},
		"nœud isolé":                   {nil, []PipelineEdge{{From: "a", To: "b", Step: 1}}, 1},
		"pastille hors garde-fous":     {nil, []PipelineEdge{{From: "a", To: "b", Step: 2}, {From: "b", To: "c"}}, 1},
		"garde-fou sans pastille":      {nil, ok, 2},
		"garde-fou porté deux fois":    {nil, []PipelineEdge{{From: "a", To: "b", Step: 1}, {From: "b", To: "c", Step: 1}}, 1},
	} {
		t.Run(name, func(t *testing.T) {
			nodes := base()
			if c.nodes != nil {
				nodes = c.nodes(nodes)
			}
			_, _, err := drawable(nodes, c.edges, c.guardrails)
			assert.ErrorIs(t, err, ErrSchemaRejected)
			assert.ErrorIs(t, err, ErrModelOutput, "raison `model` à l'écran")
		})
	}

	n, e, err := drawable(base(), append(append([]PipelineEdge{}, ok...), PipelineEdge{From: "a", To: "b"}), 1)
	require.NoError(t, err)
	assert.Len(t, n, 3)
	assert.Equal(t, ok, e, "doublon retiré (le premier fait foi)")
}

func TestEncodeEvent_TypeEnTete(t *testing.T) {
	b, err := encodeEvent(RunCompleted{})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"run.completed"}`, string(b))
	b, err = encodeEvent(LinkTraced{From: Visitor, To: "maicivy"})
	require.NoError(t, err)
	assert.Equal(t, `{"type":"link.traced","from":"visitor","to":"maicivy"}`, string(b))
	assert.Equal(t, "run.failed", EventType(RunFailed{}))
}
