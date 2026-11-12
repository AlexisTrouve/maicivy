package database

import (
	"testing"
	"time"

	"maicivy/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// QUOI : vérifie l'ajout des colonnes MANQUANTES sur une table existante.
//
// POURQUOI : jusqu'ici RunAutoMigrations sautait intégralement toute table déjà présente. Conséquence
// silencieuse et déjà rencontrée : ajouter un champ à un modèle déjà déployé ne créait PAS la colonne
// en production, et le code se mettait à lire une colonne inexistante. Le piège ne se voyait ni en
// test (SQLite recrée tout) ni en développement (base neuve).
//
// POURQUOI ne pas simplement activer AutoMigrate sur les tables existantes : mesuré sur la forme réelle
// de la production, AutoMigrate n'échoue plus (le driver v1.6.0 a corrigé ça) mais émettrait cinq
// ALTER ... TYPE varchar(N) de RÉTRÉCISSEMENT. En postgres, le cast explicite tronque en SILENCE au
// lieu d'échouer. Ces cinq-là seraient sans effet aujourd'hui (longueurs réelles vérifiées : 57/255,
// 62/200, 19/255, 0/200, 28/100), mais le mécanisme resterait armé pour le prochain changement de
// modèle, exécuté au démarrage, en prod, sans relecture. On garde donc le geste STRICTEMENT additif :
// il supprime le piège sans ouvrir cette porte.

type migColsV1 struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func (migColsV1) TableName() string { return "mig_cols" }

// migColsV2 : même table, un champ en plus — le cas « on a ajouté un champ au modèle ».
type migColsV2 struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Email string
}

func (migColsV2) TableName() string { return "mig_cols" }

// migColsNarrow : même table, mais Name RÉTRÉCI. Ne doit provoquer aucune modification.
type migColsNarrow struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"type:varchar(3)"`
}

func (migColsNarrow) TableName() string { return "mig_cols" }

func newColsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&migColsV1{}))
	return db
}

func TestAddMissingColumns_AjouteLeChampNouveau(t *testing.T) {
	db := newColsDB(t)
	require.False(t, db.Migrator().HasColumn(&migColsV2{}, "email"))

	added, err := addMissingColumns(db, &migColsV2{})

	require.NoError(t, err)
	assert.Equal(t, []string{"email"}, added)
	assert.True(t, db.Migrator().HasColumn(&migColsV2{}, "email"),
		"sans ça, le code lirait une colonne qui n'existe pas en production")
}

// La donnée déjà présente doit survivre : on ajoute, on ne reconstruit pas.
func TestAddMissingColumns_PreserveLesDonnees(t *testing.T) {
	db := newColsDB(t)
	require.NoError(t, db.Exec("INSERT INTO mig_cols (id, name) VALUES (1, 'Alexis')").Error)

	_, err := addMissingColumns(db, &migColsV2{})
	require.NoError(t, err)

	var name string
	require.NoError(t, db.Raw("SELECT name FROM mig_cols WHERE id = 1").Scan(&name).Error)
	assert.Equal(t, "Alexis", name)
}

// LE garde-fou : un champ dont le type a rétréci dans le modèle ne doit RIEN déclencher. C'est
// exactement le cas qui, en postgres, tronquerait des données en silence.
func TestAddMissingColumns_NeModifieJamaisUneColonneExistante(t *testing.T) {
	db := newColsDB(t)
	require.NoError(t, db.Exec("INSERT INTO mig_cols (id, name) VALUES (1, 'un nom bien plus long que trois caracteres')").Error)

	added, err := addMissingColumns(db, &migColsNarrow{})

	require.NoError(t, err)
	assert.Empty(t, added, "aucune colonne ne manque : il ne doit rien se passer")

	var name string
	require.NoError(t, db.Raw("SELECT name FROM mig_cols WHERE id = 1").Scan(&name).Error)
	assert.Equal(t, "un nom bien plus long que trois caracteres", name,
		"un rétrécissement de type ne doit jamais amputer la donnée existante")
}

// Idempotence : un second passage ne doit rien réajouter (le démarrage rejoue la migration).
func TestAddMissingColumns_Idempotent(t *testing.T) {
	db := newColsDB(t)

	_, err := addMissingColumns(db, &migColsV2{})
	require.NoError(t, err)

	added, err := addMissingColumns(db, &migColsV2{})
	require.NoError(t, err)
	assert.Empty(t, added)
}

// RÉGRESSION — les colonnes d'historique de l'atelier de composition (title, archived_at, sent_at)
// sont ajoutées à une table compose_drafts DÉJÀ DÉPLOYÉE.
//
// POURQUOI ce test existe : c'est le piège documenté du projet. RunAutoMigrations SAUTE les tables
// existantes ; sans addMissingColumns, ces trois colonnes ne seraient jamais créées en production —
// la table y existe depuis le 2026-07-30. Le symptôme serait un écran qui marche parfaitement en
// local et échoue en prod à la première écriture, sur une erreur de colonne inconnue.
func TestAddMissingColumns_ComposeDraftHistorique(t *testing.T) {
	// Base NUE : on recrée nous-mêmes la table telle qu'elle est en production, sans passer par
	// AutoMigrate — sinon le modèle à jour créerait les colonnes et le test ne prouverait rien.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	// Table telle qu'elle existe en production AVANT cette fonctionnalité : sans les trois colonnes.
	require.NoError(t, db.Exec(`CREATE TABLE compose_drafts (
		id text PRIMARY KEY,
		announce text,
		draft text,
		translation text,
		rate integer DEFAULT 0,
		rate_rationale text,
		messages text DEFAULT '[]',
		checks text DEFAULT '{}',
		created_at datetime,
		updated_at datetime
	)`).Error)

	require.NoError(t, migrateModel(db, &models.ComposeDraft{}, "compose_drafts"))

	for _, col := range []string{"title", "archived_at", "sent_at"} {
		assert.True(t, db.Migrator().HasColumn(&models.ComposeDraft{}, col),
			"colonne %q absente : elle ne serait jamais créée en production", col)
	}

	// Et la table reste utilisable : une écriture complète doit passer.
	now := time.Now()
	require.NoError(t, db.Create(&models.ComposeDraft{
		ID: "t1", Title: "Galadrim — AI agents", Draft: "d", Messages: "[]", Checks: "{}",
		ArchivedAt: &now, SentAt: &now,
	}).Error)
}

// RÉGRESSION — les colonnes du suivi « traité » + rappels (handled_at, reminders_sent,
// last_reminder_at) sont ajoutées à une table mailbox_emails DÉJÀ DÉPLOYÉE (depuis juillet 2026).
// Même piège que ci-dessus : sans addMissingColumns, le premier cycle de rappels échouerait en prod
// sur une colonne inconnue, alors que tout passe en local.
func TestAddMissingColumns_MailboxRappels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	// Table telle qu'en production AVANT cette fonctionnalité.
	require.NoError(t, db.Exec(`CREATE TABLE mailbox_emails (
		id text PRIMARY KEY,
		message_id varchar(998) NOT NULL,
		imap_uid integer NOT NULL,
		from_address varchar(320) NOT NULL,
		from_domain varchar(255) NOT NULL,
		platform varchar(100) NOT NULL,
		subject text,
		body_text text,
		received_at datetime NOT NULL,
		read boolean DEFAULT false,
		forwarded_at datetime,
		forward_error text,
		is_opportunity boolean DEFAULT false,
		relevance_score integer,
		relevance_reason text,
		forward_blocked boolean DEFAULT false,
		relevance_cot text,
		relevance_link text,
		created_at datetime,
		updated_at datetime,
		deleted_at datetime
	)`).Error)

	require.NoError(t, migrateModel(db, &models.MailboxEmail{}, "mailbox_emails"))

	for _, col := range []string{"handled_at", "reminders_sent", "last_reminder_at"} {
		assert.True(t, db.Migrator().HasColumn(&models.MailboxEmail{}, col),
			"colonne %q absente : elle ne serait jamais créée en production", col)
	}

	// Une ligne déjà présente (reminders_sent ajouté après coup) doit rester sélectionnable par la
	// requête des rappels : le défaut 0 doit s'appliquer, sinon `reminders_sent < 2` l'exclut (NULL).
	require.NoError(t, db.Exec(`INSERT INTO mailbox_emails (id, message_id, imap_uid, from_address, from_domain, platform, received_at)
		VALUES ('a', 'm@malt.fr', 1, 'n@malt.fr', 'malt.fr', 'malt', CURRENT_TIMESTAMP)`).Error)
	var n int64
	require.NoError(t, db.Model(&models.MailboxEmail{}).Where("reminders_sent < ?", 2).Count(&n).Error)
	assert.Equal(t, int64(1), n)
}

// RÉGRESSION — la colonne propriétaire prospect_id est ajoutée à une table compose_drafts DÉJÀ
// DÉPLOYÉE (avec title/archived_at/sent_at). Sans addMissingColumns elle ne serait jamais créée en
// prod, et le cloisonnement des ateliers par prospecteur échouerait à la première requête.
// Les lignes existantes doivent la recevoir à NULL : c'est ce qui les rattache à l'owner.
func TestAddMissingColumns_ComposeDraftProprietaire(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	// Forme de production AVANT cette fonctionnalité.
	require.NoError(t, db.Exec(`CREATE TABLE compose_drafts (
		id text PRIMARY KEY,
		announce text,
		draft text,
		translation text,
		rate integer DEFAULT 0,
		rate_rationale text,
		messages text DEFAULT '[]',
		checks text DEFAULT '{}',
		title varchar(200),
		archived_at datetime,
		sent_at datetime,
		created_at datetime,
		updated_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO compose_drafts (id, draft) VALUES ('ancien', 'd')`).Error)

	require.NoError(t, migrateModel(db, &models.ComposeDraft{}, "compose_drafts"))

	assert.True(t, db.Migrator().HasColumn(&models.ComposeDraft{}, "prospect_id"),
		"colonne prospect_id absente : elle ne serait jamais créée en production")

	// L'atelier existant reste celui de l'owner (NULL), et la donnée survit.
	var row models.ComposeDraft
	require.NoError(t, db.First(&row, "id = ?", "ancien").Error)
	assert.Nil(t, row.ProspectID)
	assert.Equal(t, "d", row.Draft)

	// Une écriture de prospecteur passe.
	pid := "p-1"
	require.NoError(t, db.Create(&models.ComposeDraft{ID: "neuf", ProspectID: &pid, Messages: "[]", Checks: "{}"}).Error)
}

// RÉGRESSION — prospect_id ajoutée à une table chat_conversations DÉJÀ DÉPLOYÉE (onglet Agent ouvert
// aux prospecteurs). Sans addMissingColumns, le cloisonnement des conversations échouerait en prod à
// la première requête. Les conversations existantes reçoivent NULL : elles restent celles de l'owner.
func TestAddMissingColumns_ChatConversationProprietaire(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	// Forme de production AVANT cette fonctionnalité.
	require.NoError(t, db.Exec(`CREATE TABLE chat_conversations (
		id text PRIMARY KEY,
		title varchar(200),
		messages text DEFAULT '[]',
		created_at datetime,
		updated_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO chat_conversations (id, title) VALUES ('ancienne', 't')`).Error)

	require.NoError(t, migrateModel(db, &models.ChatConversation{}, "chat_conversations"))

	assert.True(t, db.Migrator().HasColumn(&models.ChatConversation{}, "prospect_id"),
		"colonne prospect_id absente : elle ne serait jamais créée en production")
	var row models.ChatConversation
	require.NoError(t, db.First(&row, "id = ?", "ancienne").Error)
	assert.Nil(t, row.ProspectID)
	assert.Equal(t, "t", row.Title)
}
