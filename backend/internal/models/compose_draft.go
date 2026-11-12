package models

import "time"

// ComposeDraft : état persisté d'un atelier de composition (/admin/compose).
//
// POURQUOI cette table existe : sans elle, un simple rafraîchissement de page perd le message en
// cours. Pour un opérateur qui ne relit pas le français et vient de passer plusieurs tours à faire
// ajuster le texte par l'agent, c'est tout le travail qui disparaît — et il n'a aucun moyen de le
// reconstituer lui-même.
//
// POURQUOI une table dédiée et non une colonne sur chat_conversations : la boucle de migration du
// projet SAUTE les tables déjà existantes (cf. database/migrations.go). Une colonne ajoutée à une
// table déjà déployée ne serait donc jamais créée en prod, alors qu'une table neuve l'est. C'est
// aussi la meilleure séparation — l'artefact n'est pas une affaire de conversation.
type ComposeDraft struct {
	ID string `gorm:"type:uuid;primaryKey" json:"id"`

	// Announce : l'annonce source, telle que collée par l'opérateur. Conservée parce que le juge
	// sémantique en a besoin à CHAQUE vérification, y compris après restauration.
	Announce string `gorm:"type:text" json:"announce"`

	// Draft / Translation : le livrable et son miroir de contrôle.
	Draft       string `gorm:"type:text" json:"draft"`
	Translation string `gorm:"type:text" json:"translation"`

	// Rate / RateRationale : tarif retenu et sa justification. La justification est persistée parce
	// que c'est ce qui permet à l'opérateur de défendre le montant face au client, parfois des jours
	// après l'avoir généré.
	Rate          int    `gorm:"default:0" json:"rate"`
	RateRationale string `gorm:"type:text" json:"rate_rationale"`

	// Messages / Checks : JSON opaque côté backend — seul le front interprète leur forme. Stockés en
	// chaîne pour la même raison que ChatConversation.Messages : GORM et le jsonb postgres ne
	// s'entendent pas bien sur des types custom, et ces champs ne sont jamais requêtés par contenu.
	Messages string `gorm:"type:jsonb;default:'[]'" json:"-"`
	Checks   string `gorm:"type:jsonb;default:'{}'" json:"-"`

	// Title : nom lisible de l'atelier, généré automatiquement à la première vérification DANS LA
	// LANGUE DE L'OPÉRATRICE (cf. ComposeService.title). Modifiable à la main ensuite.
	//
	// POURQUOI généré et non déduit du texte de l'annonce : l'annonce est parfois une CAPTURE D'ÉCRAN,
	// auquel cas il n'y a aucun texte à découper. Et un titre en français ne servirait à rien à
	// quelqu'un qui ne lit pas le français — or c'est justement ce qui doit lui permettre de
	// retrouver un atelier dans une liste.
	Title string `gorm:"type:varchar(200)" json:"title"`

	// ArchivedAt : rangé hors de la liste courante. On n'efface JAMAIS un atelier (décision d'Alexi) —
	// un message envoyé à un client est une trace qu'on peut vouloir relire des mois plus tard, et une
	// suppression est irréversible alors qu'un archivage ne l'est pas. Nul = actif.
	ArchivedAt *time.Time `json:"archived_at,omitempty"`

	// SentAt : l'opératrice a effectivement envoyé ce message sur la plateforme. Nul = pas encore.
	//
	// POURQUOI un horodatage plutôt qu'un booléen : « quand » répond à la question qu'on se pose
	// vraiment en relisant un atelier — depuis combien de temps j'attends une réponse. Le coût est le
	// même.
	SentAt *time.Time `json:"sent_at,omitempty"`

	// ProspectID : propriétaire de l'atelier. NUL = l'owner — c'est ce que valent tous les ateliers
	// créés avant l'arrivée des prospecteurs, sans backfill. Renseigné = l'atelier d'un prospecteur,
	// invisible de tout autre compte (cf. ComposeDraftsHandler.scope).
	//
	// POURQUOI nullable et non une chaîne « owner » : la colonne est AJOUTÉE à une table déjà déployée
	// (addMissingColumns) ; les lignes existantes y reçoivent NULL, qui doit donc signifier « owner ».
	ProspectID *string `gorm:"type:varchar(36)" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ComposeDraft) TableName() string {
	return "compose_drafts"
}
