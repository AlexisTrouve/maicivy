package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// maxMailboxBodyBytes plafonne le corps stocké par mail — défense contre un mail-bombe qui gonflerait
// la DB (200KB couvre très large un email texte de notification plateforme).
const maxMailboxBodyBytes = 200 * 1024

// maxForwardRetryBatch borne le nombre de transferts en échec retentés par cycle de poll — évite qu'un
// incident du relai SMTP fasse gonfler indéfiniment le travail d'un seul PollOnce.
const maxForwardRetryBatch = 50

// defaultMailboxKey identifie la (seule, pour l'instant) boîte suivie dans MailboxCursor.
const defaultMailboxKey = "primary"

// MailboxRelaySender envoie un message RFC822 déjà composé (signature de SendViaRelay) — injecté dans
// MailboxService pour permettre un fake en test (mailbox_service_test.go), sans réseau réel.
type MailboxRelaySender func(addr, from, to string, msg []byte) error

// MailboxService — ingestion IMAP (Malt, puis autres plateformes) + dispatch automatique vers une
// adresse tierce fixe. Toute la logique métier (curseur, allowlist, dédup, retry) est ici ; la
// connexion réseau est déléguée à ImapFetcher et l'envoi SMTP à MailboxRelaySender — les deux
// injectables, donc testables sans réseau réel.
type MailboxService struct {
	db        *gorm.DB
	fetcher   ImapFetcher
	allowlist map[string]string // domaine(lowercase) → platform
	relay     MailboxRelaySender
	relevance MailboxRelevanceEvaluator // nil = filtre désactivé, tout est transféré (comportement pré-filtre)

	smtpAddr  string
	forwardTo string
	fromEmail string
	fromName  string
}

// NewMailboxService construit le service. relay=nil → SendViaRelay (le vrai relai exim VPS57).
// relevance=nil → pas de filtre de pertinence (tout mail allowlisté est transféré, comme avant son
// introduction) — même convention que les autres dépendances optionnelles de maicivy.
func NewMailboxService(db *gorm.DB, fetcher ImapFetcher, allowlist map[string]string, smtpAddr, forwardTo, fromEmail, fromName string, relay MailboxRelaySender, relevance MailboxRelevanceEvaluator) *MailboxService {
	if relay == nil {
		relay = SendViaRelay
	}
	return &MailboxService{
		db:        db,
		fetcher:   fetcher,
		allowlist: allowlist,
		relay:     relay,
		relevance: relevance,
		smtpAddr:  smtpAddr,
		forwardTo: forwardTo,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// PollOnce est le point d'entrée appelé par le job ticker (cf. jobs/mailbox_poll.go).
//
// ORDRE (voir le plan) :
//  1. Retente d'abord les transferts en échec — mécanisme de retry complet, pas de backoff séparé.
//  2. Lit/crée le curseur. Premier démarrage ou changement d'UIDValidity → reset SANS jamais fetcher
//     (garantit qu'on n'ingère jamais l'historique existant de la boîte).
//  3. Sinon : fetch incrémental, filtre allowlist, dédup, insertion + avance du curseur PAR message,
//     puis tentative de transfert immédiate.
func (s *MailboxService) PollOnce(ctx context.Context) error {
	s.retryFailedForwards(ctx)

	uidValidity, uidNext, err := s.fetcher.MailboxStatus(ctx)
	if err != nil {
		return fmt.Errorf("mailbox status: %w", err)
	}

	var cursor models.MailboxCursor
	err = s.db.Where("mailbox_key = ?", defaultMailboxKey).First(&cursor).Error
	isNew := errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !isNew {
		return fmt.Errorf("load cursor: %w", err)
	}

	if isNew || cursor.UIDValidity != uidValidity {
		// Cold start OU boîte recréée (UIDValidity a changé, les anciens UID ne veulent plus rien
		// dire) : on seed le curseur à UIDNext-1 SANS FETCHER — l'historique existant n'est jamais
		// ingéré/transféré.
		var lastUID uint32
		if uidNext > 0 {
			lastUID = uidNext - 1
		}
		if isNew {
			cursor = models.MailboxCursor{MailboxKey: defaultMailboxKey}
		}
		cursor.UIDValidity = uidValidity
		cursor.LastUID = lastUID
		if err := s.db.Save(&cursor).Error; err != nil {
			return fmt.Errorf("seed cursor: %w", err)
		}
		log.Info().Bool("new", isNew).Uint32("uid_validity", uidValidity).Uint32("last_uid", lastUID).
			Msg("mailbox: curseur (ré)initialisé sans backfill")
		return nil
	}

	messages, err := s.fetcher.FetchSince(ctx, cursor.LastUID)
	if err != nil {
		return fmt.Errorf("fetch since: %w", err)
	}
	// Tri défensif : l'avance du curseur PAR message dépend de cet ordre pour une reprise propre
	// après crash (voir ImapFetcher.FetchSince).
	sort.Slice(messages, func(i, j int) bool { return messages[i].UID < messages[j].UID })

	for _, m := range messages {
		s.processMessage(ctx, &cursor, m)
	}
	return nil
}

// processMessage traite UN message inspecté : insertion si allowlisté (+ tentative de transfert),
// puis avance et persiste le curseur — TOUJOURS, même si le message a été ignoré, car le curseur suit
// tout ce qui a été INSPECTÉ, pas seulement ce qui a été retenu.
func (s *MailboxService) processMessage(ctx context.Context, cursor *models.MailboxCursor, m ImapMessage) {
	domain := EmailDomain(m.From)
	if platform, ok := MatchMailboxDomain(domain, s.allowlist); ok {
		s.ingest(ctx, m, domain, platform)
	}

	if m.UID > cursor.LastUID {
		cursor.LastUID = m.UID
		if err := s.db.Model(&models.MailboxCursor{}).Where("id = ?", cursor.ID).
			Update("last_uid", cursor.LastUID).Error; err != nil {
			log.Error().Err(err).Uint32("uid", m.UID).Msg("mailbox: échec avance curseur")
		}
	}
}

// ingest stocke un mail allowlisté (dédup par MessageID — filet de sécurité en cas de reprise après
// crash, cf. PollOnce) puis tente immédiatement son transfert.
func (s *MailboxService) ingest(ctx context.Context, m ImapMessage, domain, platform string) {
	messageID := m.MessageID
	if messageID == "" {
		// Garde-fou : un Message-ID est quasi toujours présent, mais un mail malformé sans en-tête
		// ne doit pas planter l'ingestion — on synthétise une clé de dédup stable sur l'UID.
		messageID = fmt.Sprintf("imap-uid-%d@synthetic.mailbox", m.UID)
	}

	var count int64
	s.db.Model(&models.MailboxEmail{}).Where("message_id = ?", messageID).Count(&count)
	if count > 0 {
		return // déjà stocké (rejeu après crash) — rien à refaire
	}

	body := m.BodyText
	if len(body) > maxMailboxBodyBytes {
		body = body[:maxMailboxBodyBytes]
	}

	email := models.MailboxEmail{
		MessageID:   messageID,
		ImapUID:     m.UID,
		FromAddress: m.From,
		FromDomain:  domain,
		Platform:    platform,
		Subject:     m.Subject,
		BodyText:    body,
		ReceivedAt:  m.ReceivedAt,
	}
	if err := s.db.Create(&email).Error; err != nil {
		log.Error().Err(err).Str("message_id", messageID).Msg("mailbox: échec stockage mail")
		return
	}

	s.judgeRelevance(ctx, &email)
	s.attemptForward(ctx, &email)
}

// judgeRelevance juge la pertinence d'une opportunité (si le filtre est configuré) et persiste le
// verdict. NE BLOQUE JAMAIS le transfert — décision produit : Tingting (qui postule pour Alexi) doit
// TOUT voir, le score sert à orienter l'action (cf. attemptForward → maltAction, injecté dans le
// sujet/corps du transfert), pas à décider à sa place. ForwardBlocked reste dans le modèle pour les
// mails bloqués manuellement/historiquement (cf. backfill) — juste plus jamais posé ICI.
//
// POURQUOI fail-open sur erreur LLM : une panne réseau/proxy ne doit JAMAIS coûter une opportunité —
// on transfère comme si le filtre n'existait pas plutôt que de bloquer sur une erreur technique.
// POURQUOI seul IsOpportunity=true est scoré : le score n'a de sens que pour une VRAIE proposition de
// mission — une newsletter/digest/notif de compte n'a rien à évaluer (RelevanceScore reste nil, pas
// 0, pour ne pas se lire comme "jugé et mauvais").
func (s *MailboxService) judgeRelevance(ctx context.Context, e *models.MailboxEmail) {
	if s.relevance == nil {
		return
	}
	verdict, err := s.relevance.Evaluate(ctx, e.Subject, e.BodyText)
	if err != nil {
		log.Warn().Err(err).Str("message_id", e.MessageID).
			Msg("mailbox: échec jugement pertinence — transfert quand même (fail-open)")
		return
	}

	e.IsOpportunity = verdict.IsOpportunity
	e.RelevanceReason = verdict.Reason
	e.RelevanceCoT = verdict.CoT
	e.RelevanceLink = verdict.Link
	updates := map[string]interface{}{
		"is_opportunity":   verdict.IsOpportunity,
		"relevance_reason": verdict.Reason,
		"relevance_cot":    verdict.CoT,
		"relevance_link":   verdict.Link,
	}
	if verdict.IsOpportunity {
		score := verdict.Score
		e.RelevanceScore = &score
		updates["relevance_score"] = score
	}
	s.db.Model(&models.MailboxEmail{}).Where("id = ?", e.ID).Updates(updates)
}

// maltAction retourne la consigne d'action (tag de sujet + bandeau de corps) à injecter dans le mail
// TRANSFÉRÉ. Les deux vides pour une non-opportunité (newsletter/digest/notif : rien à refuser) ou si
// le filtre n'est pas configuré. N'annule jamais le transfert (cf. judgeRelevance).
//
// POURQUOI une consigne sur TOUTE opportunité, y compris les bonnes : sur Malt, une offre laissée en
// attente BLOQUE les offres suivantes, et la plateforme n'a aucune API (cf. MaltReminderService) —
// seule une action manuelle la débloque. Toute opportunité doit donc finir postulée OU refusée. À
// l'envoi on ignore ce que la destinataire retiendra, donc chacune porte sa consigne : refuser sous
// le seuil, postuler-sinon-refuser au-dessus.
//
// POURQUOI ce n'est plus un avertissement : l'ancien texte (« probablement pas la peine de postuler »)
// décrivait un jugement et invitait à l'inaction — or ignorer une offre est précisément ce qui sature
// la file Malt. Le score reste affiché, mais il accompagne une action au lieu de la remplacer.
//
// POURQUOI EN ANGLAIS : la destinataire ne lit pas le français, même raison que renderMaltReminder().
// RelevanceReason est déjà produit en anglais par l'agent (cf. models.MailboxEmail) — rien à traduire.
//
// Score nil sur une opportunité ne devrait pas arriver (judgeRelevance pose toujours le score quand
// IsOpportunity=true), mais on ne prend pas ce risque en silence : la consigne tombe alors sur la
// branche « à examiner », sans score. Ne JAMAIS renvoyer "" ici pour une opportunité — ce serait
// perdre le refus et rebloquer la file.
func maltAction(e *models.MailboxEmail, threshold int) (tag, note string) {
	if !e.IsOpportunity {
		return "", ""
	}
	const pending = "Leaving an offer pending blocks the other offers on Malt, so it must be refused explicitly."

	if e.RelevanceScore != nil && *e.RelevanceScore < threshold {
		return "TO REFUSE", fmt.Sprintf(
			"REFUSE this offer on Malt. Low relevance (%d/100). Reason: %s\r\n%s",
			*e.RelevanceScore, e.RelevanceReason, pending)
	}
	if e.RelevanceScore != nil {
		return "TO REVIEW", fmt.Sprintf(
			"Relevance %d/100. Apply if it fits — otherwise REFUSE it on Malt.\r\n%s",
			*e.RelevanceScore, pending)
	}
	return "TO REVIEW", fmt.Sprintf("Not scored. Apply if it fits — otherwise REFUSE it on Malt.\r\n%s", pending)
}

// retryFailedForwards retente les transferts jamais réussis (forwarded_at NULL), plus vieux d'abord,
// bornés à maxForwardRetryBatch par cycle. C'est tout le mécanisme de retry — un mail non transféré
// est retenté à chaque cycle suivant jusqu'à succès, jamais silencieusement abandonné.
//
// EXCLUT les mails bloqués par le filtre de pertinence (forward_blocked) : ce blocage est une
// décision, pas un échec technique — seul un forçage manuel (RetryForward depuis le panel admin) doit
// pouvoir passer outre, jamais le retry automatique.
func (s *MailboxService) retryFailedForwards(ctx context.Context) {
	var pending []models.MailboxEmail
	if err := s.db.Where("forwarded_at IS NULL AND forward_blocked = ?", false).Order("received_at ASC").
		Limit(maxForwardRetryBatch).Find(&pending).Error; err != nil {
		log.Error().Err(err).Msg("mailbox: échec lecture transferts en attente")
		return
	}
	for i := range pending {
		s.attemptForward(ctx, &pending[i])
	}
}

// attemptForward envoie (ou retente) le transfert d'un mail déjà stocké et persiste le résultat.
// Injecte la consigne d'action Malt si le mail est une opportunité — jamais un blocage (cf. maltAction).
func (s *MailboxService) attemptForward(_ context.Context, e *models.MailboxEmail) {
	var actionTag, actionNote string
	if s.relevance != nil {
		actionTag, actionNote = maltAction(e, s.relevance.Threshold())
	}
	msg := RenderMailboxForward(s.fromEmail, s.fromName, s.forwardTo, MailboxForwardInput{
		FromAddress:    e.FromAddress,
		Subject:        e.Subject,
		BodyText:       e.BodyText,
		Platform:       e.Platform,
		ReceivedAt:     e.ReceivedAt,
		MaltActionTag:  actionTag,
		MaltActionNote: actionNote,
	})

	if err := s.relay(s.smtpAddr, s.fromEmail, s.forwardTo, msg); err != nil {
		e.ForwardError = err.Error()
		s.db.Model(&models.MailboxEmail{}).Where("id = ?", e.ID).
			Update("forward_error", e.ForwardError)
		log.Warn().Err(err).Str("message_id", e.MessageID).Msg("mailbox: échec transfert")
		return
	}

	now := time.Now()
	e.ForwardedAt = &now
	e.ForwardError = ""
	s.db.Model(&models.MailboxEmail{}).Where("id = ?", e.ID).
		Updates(map[string]interface{}{"forwarded_at": now, "forward_error": ""})
}

// RetryForward retente le transfert d'UN mail précis, identifié par son ID — utilisé par le retry
// manuel de l'admin panel (POST /admin/mailbox/:id/forward), par opposition au retry automatique en
// lot de retryFailedForwards. Si le mail était bloqué par le filtre de pertinence, ce forçage manuel
// passe outre (c'est le point d'échappement humain sur un faux négatif du LLM). Retourne l'erreur de
// transfert (si échec) ou nil (succès).
func (s *MailboxService) RetryForward(ctx context.Context, id uuid.UUID) error {
	var email models.MailboxEmail
	if err := s.db.First(&email, "id = ?", id).Error; err != nil {
		return fmt.Errorf("mail introuvable: %w", err)
	}
	if email.ForwardBlocked {
		email.ForwardBlocked = false
		s.db.Model(&models.MailboxEmail{}).Where("id = ?", email.ID).Update("forward_blocked", false)
	}
	s.attemptForward(ctx, &email)
	if email.ForwardError != "" {
		return errors.New(email.ForwardError)
	}
	return nil
}

// Close libère la connexion IMAP sous-jacente (appelé au shutdown, cf. main.go).
func (s *MailboxService) Close() error {
	if s.fetcher == nil {
		return nil
	}
	return s.fetcher.Close()
}
