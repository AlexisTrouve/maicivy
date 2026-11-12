package models

import "time"

// Prospect : compte d'un apporteur d'affaires (« prospecteur »).
//
// QUOI : une personne extérieure, non proche, qui cherche des clients pour Alexis sur ses propres
// canaux. Elle n'a accès qu'à deux outils du panneau (CV depuis offre, Composer) et JAMAIS aux
// privilèges owner (Opus, bypass du rate limit).
//
// POURQUOI une table et non des comptes en config : l'owner ouvre et ferme les accès lui-même, sans
// redéploiement, et la révocation doit couper la session en cours IMMÉDIATEMENT — ce qui exige un état
// relu en base à chaque requête (cf. api.ProspectAuth), pas seulement une signature de cookie.
//
// COMMENT (cycle de vie) :
//  1. l'owner crée une INVITATION : ligne avec un nom, sans mot de passe, Active=false, et l'empreinte
//     SHA-256 d'un jeton à usage unique qui expire (InviteExpiresAt) ;
//  2. le prospecteur ouvre le lien, choisit son mot de passe : hash bcrypt posé, jeton effacé (donc
//     inutilisable une seconde fois), Active=true ;
//  3. l'owner révoque / réactive : bascule de Active.
type Prospect struct {
	ID string `gorm:"type:varchar(36);primaryKey" json:"id"`

	// Name : identifiant de connexion ET signature des messages composés en son nom. Unique, sinon
	// le login par nom serait ambigu.
	Name string `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`

	// Email : facultatif, second identifiant de connexion possible. Pointeur pour que plusieurs
	// comptes sans email ne se heurtent pas à l'index unique (NULL n'est pas égal à NULL).
	Email *string `gorm:"type:varchar(320);uniqueIndex" json:"email,omitempty"`

	// PasswordHash : bcrypt. Vide tant que l'invitation n'a pas été acceptée.
	PasswordHash string `gorm:"type:varchar(100)" json:"-"`

	// Ref : code de suivi court et unique, ajouté en `?ref=` aux liens que l'agent insère. Il attribue
	// les visites au prospecteur sans exposer son identifiant interne.
	Ref string `gorm:"type:varchar(16);uniqueIndex;not null" json:"ref"`

	// Active : seul interrupteur d'accès, relu à chaque requête authentifiée.
	Active bool `gorm:"default:false" json:"active"`

	// InviteTokenHash : SHA-256 hex du jeton d'invitation. On ne stocke jamais le jeton en clair : une
	// fuite de la base ne doit pas permettre d'accepter une invitation en attente. Vide = consommé.
	InviteTokenHash string     `gorm:"type:varchar(64);index" json:"-"`
	InviteExpiresAt *time.Time `json:"invite_expires_at,omitempty"`

	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Prospect) TableName() string { return "prospects" }

// Types d'événement d'attribution.
const (
	ProspectEventVisit = "visit" // un visiteur arrivé par un lien `?ref=<code>`
	ProspectEventCV    = "cv"    // un CV généré depuis une offre par le prospecteur lui-même
)

// ProspectEvent : un fait attribué à un prospecteur (visite entrante ou CV généré).
//
// POURQUOI une table d'événements plutôt que deux compteurs sur Prospect : la déduplication des
// visites (un visiteur qui recharge la page ne compte qu'une fois) a besoin de la clé de visiteur, et
// des lignes horodatées permettent plus tard un « sur 30 jours » sans migration.
type ProspectEvent struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	ProspectID string `gorm:"type:varchar(36);index;not null" json:"prospect_id"`
	Kind       string `gorm:"type:varchar(16);not null" json:"kind"`
	// VisitorKey : cookie de session du visiteur (ou à défaut son IP) — sert uniquement à dédupliquer
	// les visites. Vide pour les CV.
	VisitorKey string    `gorm:"type:varchar(128);index" json:"-"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ProspectEvent) TableName() string { return "prospect_events" }
