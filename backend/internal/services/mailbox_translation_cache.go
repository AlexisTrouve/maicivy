package services

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"maicivy/internal/models"
)

// ErrMailboxTranslationNotConfigured — remonté quand une traduction est demandée (allowTranslate=true)
// mais qu'aucun MailboxTranslator n'est configuré (credentials Anthropic absents).
var ErrMailboxTranslationNotConfigured = errors.New("mailbox translation service not configured")

// GetOrTranslateMailboxEmail sert la traduction en cache de (email, lang) si elle existe déjà — ZÉRO
// appel LLM dans ce cas, même mail relu 100 fois. Sinon :
//   - allowTranslate=false (check auto au chargement d'un mail dans le panel) → retourne (nil, nil),
//     jamais d'appel LLM juste en OUVRANT un mail.
//   - allowTranslate=true (clic explicite "Traduire") → traduit via translator, met en cache
//     (permanent, un seul appel par (mail, langue) pour toujours), puis retourne le résultat.
func GetOrTranslateMailboxEmail(ctx context.Context, db *gorm.DB, translator MailboxTranslator, email *models.MailboxEmail, lang string, allowTranslate bool) (*models.MailboxEmailTranslation, error) {
	var cached models.MailboxEmailTranslation
	err := db.Where("mailbox_email_id = ? AND lang = ?", email.ID, lang).First(&cached).Error
	if err == nil {
		return &cached, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("mailbox translation cache lookup: %w", err)
	}
	if !allowTranslate {
		return nil, nil // pas en cache, pas de traduction demandée — rien à faire
	}
	if translator == nil {
		return nil, ErrMailboxTranslationNotConfigured
	}

	subject, body, err := translator.Translate(ctx, email.Subject, email.BodyText, lang)
	if err != nil {
		return nil, fmt.Errorf("mailbox translation: %w", err)
	}

	t := models.MailboxEmailTranslation{
		MailboxEmailID: email.ID,
		Lang:           lang,
		Subject:        subject,
		Body:           body,
	}
	if err := db.Create(&t).Error; err != nil {
		return nil, fmt.Errorf("mailbox translation cache write: %w", err)
	}
	return &t, nil
}
