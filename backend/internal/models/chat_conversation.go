package models

import "time"

// ChatConversation — conversation PERSISTÉE de l'agent admin (mémoire durable owner).
// POURQUOI : le chat public tient l'historique côté front (perdu au refresh). L'agent admin doit
// pouvoir reprendre une discussion des jours/semaines après → on persiste le fil complet en base.
// COMMENT : Messages stocke le tableau de tours (user/assistant + tool_use/tool_result) sérialisé en
// JSONB. Le handler (de)sérialise vers []services.ChatMessage. Title = résumé court (1er message).
type ChatConversation struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	Title     string    `gorm:"type:varchar(200)" json:"title"`
	Messages  string    `gorm:"type:jsonb;default:'[]'" json:"-"` // JSON sérialisé de []ChatMessage
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName override le nom de table par défaut.
func (ChatConversation) TableName() string {
	return "chat_conversations"
}
