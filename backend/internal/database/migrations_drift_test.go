package database

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"maicivy/internal/models"
)

// QUOI : mesure ce que ferait AutoMigrate sur un schéma AYANT LA FORME DE LA PRODUCTION.
//
// POURQUOI : RunAutoMigrations saute délibérément les tables existantes, en invoquant des « GORM
// scanning issues ». Conséquence lourde — ajouter un champ à un modèle déjà déployé ne crée jamais la
// colonne. La question est de savoir si ce contournement est encore justifié depuis la montée du
// driver postgres v1.6.0. On ne peut PAS y répondre sur un schéma qu'on vient de créer soi-même : il
// serait par construction conforme aux modèles, AutoMigrate serait un no-op et le test passerait sans
// rien prouver. Le vrai risque est la DÉRIVE accumulée en production.
//
// COMMENT : on pointe une base restaurée depuis un pg_dump --schema-only de la prod (aucune donnée
// réelle), on lance AutoMigrate modèle par modèle, et on RAPPORTE le DDL qu'il émettrait. Ce test ne
// juge pas : il donne à lire. La décision d'activer ou non la migration des tables existantes touche
// des données de production et n'appartient pas au code.
//
// Lancer :
//
//	PG_DRIFT_DSN='postgres://maicivy:devpass@127.0.0.1:5434/prodshape?sslmode=disable' \
//	  go test ./internal/database/ -run TestAutoMigrate_DeriveSurFormeProd -v

// ddlRecorder capture les instructions émises par GORM, en écartant l'introspection (SELECT) qui n'a
// aucun effet sur le schéma.
type ddlRecorder struct {
	gormlogger.Interface
	stmts []string
}

func (r *ddlRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	upper := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(upper, "SELECT") {
		return
	}
	r.stmts = append(r.stmts, strings.TrimSpace(sql))
}

// TestRunAutoMigrations_SurFormeProd : ce que la migration RÉELLE ferait sur la production.
//
// C'est le test qui autorise (ou non) un déploiement : il doit ne produire QUE des créations de tables
// et des ajouts de colonnes. Le moindre ALTER ... TYPE serait une modification de schéma non relue
// appliquée au démarrage sur des données réelles — précisément ce qu'on refuse.
func TestRunAutoMigrations_SurFormeProd(t *testing.T) {
	dsn := os.Getenv("PG_DRIFT_DSN")
	if dsn == "" {
		t.Skip("PG_DRIFT_DSN non défini — mesure sur forme de prod sautée")
	}

	rec := &ddlRecorder{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: rec})
	require.NoError(t, err)

	require.NoError(t, RunAutoMigrations(db))

	for _, s := range rec.stmts {
		short := s
		if len(short) > 140 {
			short = short[:140] + "…"
		}
		t.Logf("  %s", short)

		upper := strings.ToUpper(s)
		require.False(t, strings.Contains(upper, "ALTER COLUMN") && strings.Contains(upper, "TYPE "),
			"changement de type sur la production : %s", s)
		require.NotContains(t, upper, "DROP COLUMN", "suppression de colonne : %s", s)
		require.NotContains(t, upper, "DROP TABLE", "suppression de table : %s", s)
	}
	t.Logf("=== %d instruction(s), aucune modification destructive ===", len(rec.stmts))
}

func TestAutoMigrate_DeriveSurFormeProd(t *testing.T) {
	dsn := os.Getenv("PG_DRIFT_DSN")
	if dsn == "" {
		t.Skip("PG_DRIFT_DSN non défini — mesure de dérive sautée")
	}

	rec := &ddlRecorder{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: rec})
	require.NoError(t, err)

	// Même liste que RunAutoMigrations, à jour.
	all := []struct {
		model interface{}
		name  string
	}{
		{&models.Experience{}, "experiences"},
		{&models.Project{}, "projects"},
		{&models.Skill{}, "skills"},
		{&models.Visitor{}, "visitors"},
		{&models.GeneratedLetter{}, "generated_letters"},
		{&models.AnalyticsEvent{}, "analytics_events"},
		{&models.GitHubProfile{}, "github_profiles"},
		{&models.GitHubRepository{}, "github_repositories"},
		{&models.ActivityProject{}, "activity_projects"},
		{&models.ActivityStats{}, "activity_stats"},
		{&models.BlogPost{}, "blog_posts"},
		{&models.ChatConversation{}, "chat_conversations"},
		{&models.BlogSubscriber{}, "blog_subscribers"},
		{&models.MailboxEmail{}, "mailbox_emails"},
		{&models.MailboxCursor{}, "mailbox_cursors"},
		{&models.MailboxEmailTranslation{}, "mailbox_email_translations"},
		{&models.ComposeDraft{}, "compose_drafts"},
	}

	failures, totalDDL := 0, 0
	for _, m := range all {
		rec.stmts = nil
		err := db.AutoMigrate(m.model)
		if err != nil {
			failures++
			t.Errorf("[ECHEC] %-28s %v", m.name, err)
			continue
		}
		totalDDL += len(rec.stmts)
		if len(rec.stmts) == 0 {
			t.Logf("[OK   ] %-28s aucun changement", m.name)
			continue
		}
		t.Logf("[DDL  ] %-28s %d instruction(s) :", m.name, len(rec.stmts))
		for _, s := range rec.stmts {
			if len(s) > 160 {
				s = s[:160] + "…"
			}
			t.Logf("         %s", s)
		}
	}

	t.Logf("=== BILAN : %d modèle(s) en échec, %d instruction(s) DDL au total ===", failures, totalDDL)
}
