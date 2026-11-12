package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// QUOI : vérifie qu'une base POSTGRES VIERGE peut être migrée de bout en bout.
//
// POURQUOI ce test existe : la suite habituelle tourne sur SQLite, et la prod ne repasse jamais par ce
// chemin puisque son schéma existe depuis longtemps et que RunAutoMigrations saute les tables déjà
// présentes. Résultat : le seul scénario qui exécute réellement la création complète du schéma sur
// postgres — un PREMIER déploiement, ou une restauration sur base neuve — n'était couvert par rien.
// Il échouait, et personne ne pouvait le voir.
//
// COMMENT : le test s'exécute dans un SCHÉMA dédié, créé puis détruit, pour ne rien laisser derrière
// lui. tableExists() interroge CURRENT_SCHEMA(), donc un schéma vide se comporte exactement comme une
// base neuve. Sans PG_TEST_DSN le test est sauté — la suite reste exécutable sans postgres.
//
// Lancer :
//
//	PG_TEST_DSN='postgres://maicivy:devpass@127.0.0.1:5434/maicivy_db?sslmode=disable' \
//	  go test ./internal/database/ -run TestRunAutoMigrations_BaseVierge -v
func TestRunAutoMigrations_BaseVierge(t *testing.T) {
	dsn := os.Getenv("PG_TEST_DSN")
	if dsn == "" {
		t.Skip("PG_TEST_DSN non défini — migration postgres sur base vierge sautée")
	}

	const schema = "migr_vierge_test"

	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	// Schéma neuf : on repart d'un état strictement vide à chaque exécution.
	require.NoError(t, admin.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema)).Error)
	require.NoError(t, admin.Exec(fmt.Sprintf("CREATE SCHEMA %s", schema)).Error)
	t.Cleanup(func() {
		_ = admin.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema)).Error
	})

	// Connexion dont le search_path pointe sur le schéma vide : CURRENT_SCHEMA() y répond, donc
	// RunAutoMigrations croit — à raison — avoir affaire à une base vierge.
	db, err := gorm.Open(postgres.Open(dsn+"&search_path="+schema), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	err = RunAutoMigrations(db)
	require.NoError(t, err, "une base vierge doit pouvoir être migrée : sans ça, un premier "+
		"déploiement ou une restauration sur base neuve casse au démarrage")

	// La migration doit avoir réellement créé les tables, pas juste ne pas échouer.
	for _, table := range []string{"visitors", "generated_letters", "chat_conversations", "compose_drafts"} {
		require.True(t, tableExists(db, table), "table %s absente après migration", table)
	}
}
