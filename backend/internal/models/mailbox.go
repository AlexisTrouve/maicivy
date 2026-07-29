package models

import (
	"time"

	"github.com/google/uuid"
)

// MailboxEmail — un email capté par ingestion IMAP (Malt, puis autres plateformes freelance),
// dispatché automatiquement vers une adresse tierce fixe (cf. services.MailboxService).
//
// POURQUOI ces champs : MessageID (uniqueIndex) est la clé de dédup — un même mail ne doit jamais
// être stocké deux fois même si le poll le revoit (ex: reprise après crash). ImapUID sert au suivi du
// curseur de lecture (cf. MailboxCursor) mais n'est PAS unique en soi (l'UID est propre à la boîte, pas
// au message — il change si la boîte est recréée, d'où UIDValidity côté curseur). FromDomain est
// dérivé de FromAddress et dénormalisé pour permettre un filtre SQL simple sans reparser à chaque
// requête. BodyText est plafonné à 200KB en amont (dans le service) — défense contre un mail-bombe
// qui gonflerait la DB.
type MailboxEmail struct {
	BaseModel
	MessageID    string     `gorm:"type:varchar(998);uniqueIndex;not null" json:"message_id"` // RFC 5322 limite l'en-tête à 998 caractères
	ImapUID      uint32     `gorm:"not null" json:"imap_uid"`
	FromAddress  string     `gorm:"type:varchar(320);not null" json:"from_address"` // RFC 5321 : 320 = max théorique d'une adresse
	FromDomain   string     `gorm:"type:varchar(255);index;not null" json:"from_domain"`
	Platform     string     `gorm:"type:varchar(100);index;not null" json:"platform"` // label dérivé de l'allowlist (pas une entrée utilisateur)
	Subject      string     `gorm:"type:text" json:"subject"`
	BodyText     string     `gorm:"type:text" json:"body_text"`
	ReceivedAt   time.Time  `gorm:"not null;index" json:"received_at"`
	Read         bool       `gorm:"default:false;index" json:"read"`
	ForwardedAt  *time.Time `json:"forwarded_at,omitempty"`
	ForwardError string     `gorm:"type:text" json:"forward_error,omitempty"`

	// Filtre de pertinence LLM (cf. services.MailboxRelevanceEvaluator) — évite qu'une mission hors
	// profil soit transférée automatiquement à l'adresse de dispatch. IsOpportunity distingue une
	// VRAIE proposition de mission d'une newsletter/digest/notif de compte (seules les opportunités
	// sont jugées ; RelevanceScore reste nil pour le reste — jamais 0, qui se lirait comme "jugé et
	// mauvais"). ForwardBlocked : le transfert AUTO a été bloqué (score sous le seuil) ; le mail reste
	// stocké et consultable, un forçage manuel depuis le panel (RetryForward) l'ignore.
	IsOpportunity   bool   `gorm:"default:false;index" json:"is_opportunity"`
	RelevanceScore  *int   `json:"relevance_score,omitempty"`
	RelevanceReason string `gorm:"type:text" json:"relevance_reason,omitempty"`
	ForwardBlocked  bool   `gorm:"default:false;index" json:"forward_blocked"`
	// RelevanceCoT : raisonnement pas-à-pas de l'agent (ce qu'il a vérifié via les tools maiProFiles
	// avant de conclure). RelevanceLink : URL principale (voir/postuler à la mission) extraite du
	// corps par l'agent. Les deux sont en ANGLAIS (langue de travail de l'agent) et affichés tels
	// quels dans le panel admin — contenu technique interne, pas de l'UI i18n.
	// column:relevance_cot explicite : le naming strategy de GORM convertit "CoT" en "co_t" (coupe
	// avant chaque majuscule suivant une minuscule) sans cette override — mismatch silencieux avec
	// la colonne réellement créée par la migration SQL manuelle sinon.
	RelevanceCoT  string `gorm:"column:relevance_cot;type:text" json:"relevance_cot,omitempty"`
	RelevanceLink string `gorm:"type:text" json:"relevance_link,omitempty"`
}

// TableName fixe le nom de table (cohérent avec l'entrée AutoMigrate "mailbox_emails").
func (MailboxEmail) TableName() string { return "mailbox_emails" }

// MailboxCursor — position de lecture IMAP par boîte suivie (une ligne par MailboxKey, "primary" pour
// la seule boîte actuelle).
//
// POURQUOI une table dédiée plutôt que MAX(imap_uid) sur mailbox_emails : les mails qui ne matchent pas
// l'allowlist ne sont JAMAIS stockés — un MAX() sur les seuls mails retenus régresserait silencieusement
// et ferait re-fetcher (et re-transférer si un domaine est ajouté après coup) tout ce qui a été ignoré
// entre-temps. Le curseur doit suivre tout ce qui a été INSPECTÉ, pas seulement ce qui a été RETENU.
// UIDValidity : si la boîte IMAP est recréée (migration, corruption), le serveur change cette valeur —
// les anciens UID ne veulent plus rien dire, on doit reset le curseur (cf. MailboxService.PollOnce).
type MailboxCursor struct {
	BaseModel
	MailboxKey  string `gorm:"type:varchar(100);uniqueIndex;not null;default:'primary'" json:"mailbox_key"`
	UIDValidity uint32 `gorm:"not null" json:"uid_validity"`
	LastUID     uint32 `gorm:"not null" json:"last_uid"`
}

// TableName fixe le nom de table (cohérent avec l'entrée AutoMigrate "mailbox_cursors").
func (MailboxCursor) TableName() string { return "mailbox_cursors" }

// MailboxEmailTranslation — cache PERMANENT d'une traduction (sujet+corps) d'un MailboxEmail vers une
// langue donnée. Un seul appel LLM par (mail, langue) pour toute la durée de vie du mail : jamais
// recalculé une fois présent. Traduction déclenchée UNIQUEMENT à la demande (clic "Traduire" côté
// panel admin) — jamais en arrière-plan à l'ingestion, pour ne pas payer une traduction sur des mails
// jamais consultés dans une autre langue que le français.
type MailboxEmailTranslation struct {
	BaseModel
	MailboxEmailID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_mailbox_translation_email_lang" json:"mailbox_email_id"`
	Lang           string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_mailbox_translation_email_lang" json:"lang"`
	Subject        string    `gorm:"type:text" json:"subject"`
	Body           string    `gorm:"type:text" json:"body"`
}

// TableName fixe le nom de table (cohérent avec l'entrée AutoMigrate "mailbox_email_translations").
func (MailboxEmailTranslation) TableName() string { return "mailbox_email_translations" }
