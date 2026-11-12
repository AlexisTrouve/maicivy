package dossiervivant

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/services"
)

// QUOI : la copie serveur du dossier d'un run (record.go).
//   - sur le flux brut d'un VRAI run (fixture E2E dossier-vivant-schema-run.ndjson, horodatée à la
//     réception) : la requête PDF construite depuis la copie serveur est celle que le front de
//     6f78f6c construisait depuis son état replié (services/testdata/dossier_pdf_avant/live-*.json) —
//     même rendu, durée et coût compris, et document HTML identique ;
//   - un run complet du service, contre la fausse passerelle : la copie est gardée, son id part dans
//     run.completed ; un run qui échoue ne laisse rien ; une sauvegarde en échec laisse le run remis.

// savedRecords — un DossierSaver en mémoire.
type savedRecords struct {
	mu   sync.Mutex
	recs []*services.DossierRecord
	err  error
}

func (s *savedRecords) Save(_ context.Context, rec *services.DossierRecord) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return "", s.err
	}
	s.recs = append(s.recs, rec)
	return "0123456789abcdef0123456789abcdef", nil
}

// decodeEvent — un événement du flux (JSON du contrat) en événement Go ; nil pour les types que la
// copie ne lit pas (message, tool.call…).
func decodeEvent(t *testing.T, raw json.RawMessage) Event {
	t.Helper()
	var head struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(raw, &head))
	var e Event
	switch head.Type {
	case "run.started":
		var v RunStarted
		require.NoError(t, json.Unmarshal(raw, &v))
		e = v
	case "agent.status":
		var v AgentStatus
		require.NoError(t, json.Unmarshal(raw, &v))
		e = v
	case "usage":
		var v Usage
		require.NoError(t, json.Unmarshal(raw, &v))
		e = v
	case "dossier.opened":
		var v DossierOpened
		require.NoError(t, json.Unmarshal(raw, &v))
		e = v
	case "dossier.block":
		var v DossierBlock
		require.NoError(t, json.Unmarshal(raw, &v))
		e = v
	case "run.completed":
		e = RunCompleted{}
	}
	return e
}

type avantFacts struct{}

func (avantFacts) Montagne(context.Context, string) (*services.Montagne, error) {
	return &services.Montagne{Nodes: []services.MontagneNoeud{{ID: "envoia", Name: "EnvoIA", Monogram: "EI"}}}, nil
}
func (avantFacts) ListProjects(context.Context, string) ([]services.MPFProject, error) {
	return []services.MPFProject{{ID: "envoia", Status: "production", Stats: services.MPFStats{LOC: 8971, Tests: 246}}}, nil
}
func (avantFacts) Skills(context.Context, string) ([]services.MPFSkill, error) { return nil, nil }

func TestRecordingEmitter_VraiRunMemeRenduQuAvant(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "..", "frontend", "tests", "e2e", "fixtures", "dossier-vivant-schema-run.ndjson"))
	require.NoError(t, err)
	defer f.Close()

	// L'horloge du serveur suit les instants de réception du flux enregistré : c'est la mesure que
	// faisait le front (runState.ts › elapsedAt sur `at`).
	base := time.Unix(1_700_000_000, 0)
	var clock time.Time
	saved := &savedRecords{}
	out := &recorder{}
	rec := newRecordingEmitter(context.Background(), out, saved, func() time.Time { return clock })
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var line struct {
			ReceivedMs int64           `json:"receivedMs"`
			Event      json.RawMessage `json:"event"`
		}
		require.NoError(t, json.Unmarshal(sc.Bytes(), &line))
		clock = base.Add(time.Duration(line.ReceivedMs) * time.Millisecond)
		if e := decodeEvent(t, line.Event); e != nil {
			rec.Emit(e)
		}
	}
	require.NoError(t, sc.Err())
	require.Len(t, saved.recs, 1, "le dossier remis est gardé, une fois")
	events := out.all()
	assert.Equal(t, RunCompleted{DossierID: "0123456789abcdef0123456789abcdef"}, events[len(events)-1], "l'id part dans run.completed")

	svc := services.NewDossierPDFService(avantFacts{})
	for _, lang := range []string{"fr", "en"} {
		t.Run(lang, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "services", "testdata", "dossier_pdf_avant", "live-"+lang+".json"))
			require.NoError(t, err)
			var before services.DossierPDFRequest
			require.NoError(t, json.Unmarshal(raw, &before))
			got, err := saved.recs[0].PDFRequest(lang)
			require.NoError(t, err)
			assert.Equal(t, &before, got)

			wantHTML, err := svc.RenderHTML(context.Background(), &before, false)
			require.NoError(t, err)
			gotHTML, err := svc.RenderHTML(context.Background(), got, false)
			require.NoError(t, err)
			assert.True(t, wantHTML == gotHTML, "document imprimé différent de celui d'avant")
		})
	}
}

func TestRecordingEmitter_RunCompletGarde(t *testing.T) {
	saved := &savedRecords{}
	out := &recorder{}
	s := newTestService(t, newFakeGateway(t))
	require.NoError(t, s.Run(context.Background(), tessera, "fr", NewRecordingEmitter(context.Background(), out, saved)))
	require.Len(t, saved.recs, 1)
	r := saved.recs[0]
	assert.Equal(t, services.DossierRecordVisitor{Name: "Inès Morel", Affiliation: "CTO chez Tessera", Organisation: "Tessera"}, r.Visitor)
	assert.Equal(t, len(Team), r.Agents)
	assert.Positive(t, r.CostUsd)
	require.Len(t, r.Sections, 5)
	assert.Equal(t, "need", r.Sections[0].ID)
	assert.NotEmpty(t, r.Sections[0].Blocks)
	events := out.all()
	assert.Equal(t, RunCompleted{DossierID: "0123456789abcdef0123456789abcdef"}, events[len(events)-1])
}

func TestRecordingEmitter_RunEnEchecNeGardeRien(t *testing.T) {
	g := newFakeGateway(t)
	g.truncate = "closing"
	saved := &savedRecords{}
	out := &recorder{}
	err := newTestService(t, g).Run(context.Background(), tessera, "fr", NewRecordingEmitter(context.Background(), out, saved))
	require.Error(t, err)
	assert.Empty(t, saved.recs, "un run qui échoue ne laisse aucun dossier imprimable")
	for _, e := range out.all() {
		_, done := e.(RunCompleted)
		assert.False(t, done)
	}
}

func TestRecordingEmitter_SauvegardeEnEchecRunRemisSansId(t *testing.T) {
	saved := &savedRecords{err: errors.New("redis indisponible")}
	out := &recorder{}
	rec := NewRecordingEmitter(context.Background(), out, saved)
	rec.Emit(RunStarted{Visitor: tessera})
	rec.Emit(RunCompleted{})
	assert.Equal(t, []Event{RunStarted{Visitor: tessera}, RunCompleted{}}, out.all())
}
