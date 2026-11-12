package services

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"maicivy/internal/models"
)

// providerCompteur — ContentProvider factice qui compte les lectures de projets.
type providerCompteur struct {
	appels atomic.Int32
	titre  atomic.Value // string
}

func (p *providerCompteur) GetExperiences(string) []models.Experience { return nil }
func (p *providerCompteur) GetSkills(string) []models.Skill           { return nil }
func (p *providerCompteur) GetProjects(string) []models.Project {
	p.appels.Add(1)
	t, _ := p.titre.Load().(string)
	return []models.Project{{Title: t}}
}

// Le constructeur est appelé au démarrage, AVANT l'ouverture du port HTTP : il ne doit pas lire
// maiProFiles. Panne du 2026-09-27 : cette lecture bloquait le démarrage ~50 s (maiprofiles lent)
// et le déploiement échouait sur son contrôle /health.
func TestNewLetterGenerator_NeLitPasLesProjetsAuDemarrage(t *testing.T) {
	src := &providerCompteur{}
	src.titre.Store("Theomen")

	NewLetterGenerator(nil, nil, nil, models.UserProfile{}, src)

	require.EqualValues(t, 0, src.appels.Load(), "aucune lecture de projets dans le constructeur")
}

// Les projets sont lus à chaque prompt : une liste vide au démarrage (maiprofiles en échec) ne
// doit plus rester figée jusqu'au redémarrage suivant.
func TestNewLetterGenerator_ProjetsLusAChaquePrompt(t *testing.T) {
	src := &providerCompteur{}
	src.titre.Store("Theomen")
	lg := NewLetterGenerator(nil, nil, nil, models.UserProfile{}, src)

	require.Contains(t, lg.promptBuilder.buildProjectsSection(), "Theomen")

	src.titre.Store("Drifterra")
	require.Contains(t, lg.promptBuilder.buildProjectsSection(), "Drifterra",
		"le prompt suivant doit refléter la liste courante, pas celle du démarrage")
}
