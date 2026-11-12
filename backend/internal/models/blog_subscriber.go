package models

import "github.com/lib/pq"

// BlogSubscriber — un lecteur abonné aux notifications email de nouveaux articles.
//
// POURQUOI la granularité par TOPIC : un lecteur peut ne vouloir QUE certains sujets (ex: que
// Drifterra). Topics = liste de project_name de posts ("Drifterra", "tech", "veille"...). Un abonné
// ne reçoit que les articles dont le project_name figure dans Topics. Topics VIDE = il veut TOUT.
//
// Phase 1 (ici) = CAPTURE seule : on collecte les abonnés dès maintenant. L'ENVOI des emails à la
// publication (et le double opt-in via Confirmed, et la désinscription via UnsubscribeToken) = phase 2.
type BlogSubscriber struct {
	BaseModel
	Email            string         `gorm:"uniqueIndex;not null" json:"email"`
	Topics           pq.StringArray `gorm:"type:text[]" json:"topics"`      // project_names suivis ; vide = tous
	Confirmed        bool           `gorm:"default:false" json:"confirmed"` // double opt-in (phase 2)
	UnsubscribeToken string         `gorm:"uniqueIndex" json:"-"`           // jeton du lien de désinscription (phase 2)
}

// TableName fixe le nom de table (cohérent avec l'entrée AutoMigrate "blog_subscribers").
func (BlogSubscriber) TableName() string { return "blog_subscribers" }
