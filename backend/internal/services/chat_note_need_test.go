package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : l'outil note_visitor_need — offert SEULEMENT quand l'appelant a branché NoteNeed (chat
// public d'un visiteur à session signée), et qui transmet le résumé tel quel au rappel.
// POURQUOI : il déclenche une génération payante en fond ; offert ailleurs (Composer, Agent d'un
// prospecteur), il ferait générer des CV pour des sessions qui ne sont pas des visiteurs.

func TestBuildTools_NoteNeed_SeulementSiBranche(t *testing.T) {
	s := &ChatService{}
	assert.NotContains(t, toolNames(t, s, ChatCapabilities{}), noteNeedTool)
	assert.NotContains(t, toolNames(t, s, ChatCapabilities{Compose: true}), noteNeedTool)
	assert.Contains(t, toolNames(t, s, ChatCapabilities{NoteNeed: func(string, string) {}}), noteNeedTool)
}

func TestExecuteTool_NoteNeed_TransmetLeResume(t *testing.T) {
	s := &ChatService{}
	var gotSummary, gotLang string
	calls := 0
	caps := ChatCapabilities{NoteNeed: func(summary, lang string) { calls++; gotSummary, gotLang = summary, lang }}

	// Langue hors liste du chat (« de ») : acceptée ici, la normalisation est celle du CV (→ en).
	res, err := s.executeTool(noteNeedTool, map[string]interface{}{"summary": "Lead Go, plateforme d'agents", "language": " DE "}, caps)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"ok": true}, res)
	assert.Equal(t, 1, calls)
	assert.Equal(t, "Lead Go, plateforme d'agents", gotSummary)
	assert.Equal(t, "de", gotLang)
}

func TestExecuteTool_NoteNeed_RefusVideEtNonBranche(t *testing.T) {
	s := &ChatService{}
	calls := 0
	caps := ChatCapabilities{NoteNeed: func(string, string) { calls++ }}

	_, err := s.executeTool(noteNeedTool, map[string]interface{}{"summary": "  ", "language": "fr"}, caps)
	assert.Error(t, err)
	_, err = s.executeTool(noteNeedTool, map[string]interface{}{"summary": "besoin", "language": "fr"}, ChatCapabilities{})
	assert.Error(t, err, "outil non offert → un appel est refusé, rien n'est noté")
	assert.Zero(t, calls)
}
