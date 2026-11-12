package database

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// addMissingColumns ajoute à une table EXISTANTE les colonnes présentes dans le modèle mais absentes
// du schéma. Retourne les noms ajoutés (vide si rien à faire).
//
// POURQUOI ce geste plutôt qu'un AutoMigrate complet : mesuré sur la forme réelle de la production,
// AutoMigrate n'échoue plus depuis la montée du driver postgres en v1.6.0, mais il émettrait cinq
// `ALTER ... TYPE varchar(N)` de RÉTRÉCISSEMENT. En postgres, ce cast explicite tronque en SILENCE au
// lieu d'échouer. Ces cinq-là seraient sans effet sur les données actuelles (longueurs vérifiées :
// 57/255, 62/200, 19/255, 0/200, 28/100), mais le mécanisme resterait armé pour le prochain changement
// de modèle — exécuté au démarrage, en production, sans qu'aucun humain ne relise le DDL.
//
// On ne prend donc QUE la partie qui manquait : créer les colonnes nouvelles. Jamais de changement de
// type, jamais de suppression. Un besoin de faire évoluer un type existant reste une migration SQL
// explicite et relue.
//
// COMMENT : 1. parser le modèle pour connaître ses champs, 2. ignorer les champs qui ne correspondent
// à aucune colonne (relations, champs ignorés), 3. ajouter ceux dont la colonne est absente.
func addMissingColumns(db *gorm.DB, model interface{}) ([]string, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return nil, fmt.Errorf("parse model: %w", err)
	}

	migrator := db.Migrator()
	added := []string{}
	for _, field := range stmt.Schema.Fields {
		// DBName vide = champ sans colonne (relation, champ ignoré) : rien à créer.
		if field.DBName == "" {
			continue
		}
		if migrator.HasColumn(model, field.DBName) {
			continue
		}
		if err := migrator.AddColumn(model, field.DBName); err != nil {
			return added, fmt.Errorf("add column %s: %w", field.DBName, err)
		}
		added = append(added, field.DBName)
	}
	return added, nil
}

// migrateModel : crée la table si elle est absente, sinon complète ses colonnes manquantes.
//
// Les deux cas sont volontairement asymétriques. Table absente → AutoMigrate complet, il n'y a aucune
// donnée à risquer. Table présente → ajout des colonnes manquantes UNIQUEMENT, jamais de modification
// de l'existant (cf. addMissingColumns pour le pourquoi).
func migrateModel(db *gorm.DB, model interface{}, tableName string) error {
	if !tableExists(db, tableName) {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("auto-migration failed for %s: %w", tableName, err)
		}
		log.Info().Str("table", tableName).Msg("Created table")
		return nil
	}

	added, err := addMissingColumns(db, model)
	if err != nil {
		return fmt.Errorf("adding columns to %s: %w", tableName, err)
	}
	if len(added) > 0 {
		// Journalisé explicitement : un déploiement qui modifie le schéma doit le dire.
		log.Info().Str("table", tableName).Strs("columns", added).Msg("Added missing columns")
	}
	return nil
}

// tableExists checks if a table exists in the database
func tableExists(db *gorm.DB, tableName string) bool {
	var count int64
	db.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = ?", tableName).Scan(&count)
	return count > 0
}

// RunAutoMigrations aligne le schéma sur les modèles GORM, de façon STRICTEMENT ADDITIVE.
//
// Ce qu'elle fait : crée les tables absentes, et ajoute aux tables existantes les colonnes présentes
// dans le modèle mais pas encore en base.
//
// Ce qu'elle ne fait JAMAIS : modifier le type d'une colonne existante, la renommer, la supprimer.
// Faire évoluer un type déjà déployé reste une migration SQL explicite et relue par un humain.
//
// POURQUOI cette limite : mesuré sur la forme réelle de la production, un AutoMigrate complet
// émettrait cinq `ALTER ... TYPE varchar(N)` de rétrécissement. En postgres ce cast tronque en SILENCE.
// Sans effet sur les données actuelles, mais le mécanisme resterait armé pour le prochain changement
// de modèle — appliqué au démarrage, en prod, sans relecture. Voir addMissingColumns.
//
// HISTORIQUE : cette fonction sautait auparavant TOUTE table existante, si bien qu'ajouter un champ à
// un modèle déjà déployé ne créait jamais la colonne — piège silencieux, rencontré en juillet 2026. Le
// contournement visait des erreurs de scan GORM dont la cause racine était en fait une incompatibilité
// entre gorm v1.30 et gorm.io/driver/postgres v1.5.4, corrigée en v1.6.0 (cf. migrations_pg_test.go).
//
// Couverture : TestRunAutoMigrations_BaseVierge (base postgres vierge, exige PG_TEST_DSN),
// TestAddMissingColumns_* (ajout, préservation des données, non-modification, idempotence).
func RunAutoMigrations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	log.Info().Msg("Starting database auto-migration...")

	// Models that have JSONB with custom types - migrate only if table doesn't exist
	// This avoids GORM scanning issues with existing JSONB data
	jsonbModels := []struct {
		model     interface{}
		tableName string
	}{
		{&models.Experience{}, "experiences"},
		{&models.Project{}, "projects"},
	}

	for _, m := range jsonbModels {
		if err := migrateModel(db, m.model, m.tableName); err != nil {
			return err
		}
	}

	// Other models - also skip if table exists to avoid GORM scanning issues
	// Note: GitHubToken is embedded in GitHubProfile (JSONB), not a separate table
	otherModels := []struct {
		model     interface{}
		tableName string
	}{
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
		{&models.Prospect{}, "prospects"},
		{&models.ProspectEvent{}, "prospect_events"},
	}

	for _, m := range otherModels {
		if err := migrateModel(db, m.model, m.tableName); err != nil {
			return err
		}
	}

	log.Info().Msg("Database auto-migration completed successfully")
	return nil
}
