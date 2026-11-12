package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// Lien « ✓ 标记为已处理 » + rappels des opportunités Malt restées ouvertes.
//
// POURQUOI : mesuré en octobre 2026, le délai moyen de réponse aux offres Malt était de ~40 h et la
// meilleure du mois (87/100) n'a jamais eu de réponse — non par refus, mais parce qu'elle s'est noyée
// dans la boîte. Sur Malt la première réponse rafle souvent la mission : un rappel ramène l'offre en
// haut de la boîte au moment où elle compte encore.
//
// LE PROBLÈME DUR : savoir qu'une offre a été traitée. Malt n'a aucune API (cf. MaltReminderService) ;
// sans signal, on relancerait aussi ce qui est fait, et le rappel redeviendrait du bruit. Le signal est
// donc un CLIC : chaque transfert d'opportunité porte un lien signé ; un clic marque l'offre traitée
// (HandledAt) et arrête les relances. Sans clic, la relance est BORNÉE (rappelsDelais) et s'arrête
// d'elle-même — un oubli de clic coûte au plus deux mails, jamais une boucle.

// rappelsDelais : âge du transfert (depuis ForwardedAt) à partir duquel part chaque rappel. Leur
// nombre est le plafond total de rappels par offre.
var rappelsDelais = []time.Duration{4 * time.Hour, 12 * time.Hour}

// rappelEcartMin : écart minimal entre deux rappels de la même offre. POURQUOI en plus des délais :
// au premier déploiement, une offre vieille de 20 h a déjà dépassé les DEUX délais — sans écart, les
// deux rappels partiraient à deux minutes d'intervalle (deux cycles de poll).
const rappelEcartMin = 4 * time.Hour

// rappelFenetre : au-delà de cet âge, on ne relance plus. POURQUOI : une offre Malt de plus de deux
// jours est en général pourvue ; et au déploiement, l'historique entier serait sinon relancé d'un coup.
const rappelFenetre = 48 * time.Hour

// ackSigPrefix sépare le domaine de signature. POURQUOI c'est indispensable et pas décoratif : le
// secret est le SESSION_SECRET, et le cookie de session est HMAC(uuid). L'id d'un mail EST un uuid :
// sans préfixe, la signature d'un lien « traité » serait un cookie de session valide, et l'inverse.
const ackSigPrefix = "mailbox-ack:"

// SetAckLink active le lien « traité » et les rappels. baseURL = origine publique du site (le lien
// pointe sur /api/v1/mailbox/ack/:id) ; secret = clé HMAC. secret vide → ni lien ni rappel : un
// rappel sans moyen de l'arrêter serait du harcèlement borné, on préfère ne rien envoyer.
func (s *MailboxService) SetAckLink(baseURL, secret string) {
	s.ackBaseURL = strings.TrimRight(baseURL, "/")
	s.ackSecret = secret
}

// MailboxAckSig signe l'id d'un mail pour le lien « traité » — base64url(HMAC-SHA256(préfixe+id)).
func MailboxAckSig(id, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ackSigPrefix + id))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyMailboxAck dit si sig est bien la signature de id. Comparaison constant-time ; un secret vide
// refuse tout (sinon n'importe qui calculerait la signature d'un secret vide).
func VerifyMailboxAck(id, sig, secret string) bool {
	if secret == "" || sig == "" {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(MailboxAckSig(id, secret)))
}

// MailboxAckURL construit le lien cliquable porté par le mail transféré.
func MailboxAckURL(baseURL, id, secret string) string {
	return strings.TrimRight(baseURL, "/") + "/api/v1/mailbox/ack/" + id + "?sig=" + MailboxAckSig(id, secret)
}

// ackURLPour : le lien du mail e, ou "" s'il n'en porte pas (non-opportunité : rien à traiter ;
// secret absent : feature désactivée).
func (s *MailboxService) ackURLPour(e *models.MailboxEmail) string {
	if s.ackSecret == "" || !e.IsOpportunity {
		return ""
	}
	return MailboxAckURL(s.ackBaseURL, e.ID.String(), s.ackSecret)
}

// SendReminders relance les opportunités au-dessus du seuil, transférées, non traitées, dans la
// fenêtre, dont le prochain délai est échu. Appelé à chaque PollOnce (toutes les ~2 min).
//
// COMMENT : la requête SQL sélectionne les candidates (bornes larges, indexées) ; le choix du rappel
// dû se fait en Go (délai n°RemindersSent + écart depuis le dernier). Le compteur n'avance qu'après un
// envoi RÉUSSI : un relai en panne fait retenter au cycle suivant, comme pour les transferts.
// Sous le seuil (« ✕ 建议拒绝 ») on ne relance pas : l'objet des rappels est de ne plus rater une offre
// qui VAUT une réponse, pas de rappeler chaque refus.
func (s *MailboxService) SendReminders(ctx context.Context, now time.Time) {
	if s.ackSecret == "" || s.relevance == nil {
		return
	}
	threshold := s.relevance.Threshold()

	var candidates []models.MailboxEmail
	if err := s.db.Where(
		"is_opportunity = ? AND handled_at IS NULL AND forwarded_at IS NOT NULL AND forwarded_at > ? AND reminders_sent < ? AND relevance_score >= ?",
		true, now.Add(-rappelFenetre), len(rappelsDelais), threshold,
	).Order("forwarded_at ASC").Find(&candidates).Error; err != nil {
		log.Error().Err(err).Msg("mailbox: lecture des rappels dus impossible")
		return
	}

	for i := range candidates {
		e := &candidates[i]
		if !rappelDu(e, now) {
			continue
		}
		n := e.RemindersSent + 1
		actionTag, actionNote := maltAction(e, threshold)
		msg := RenderMailboxForward(s.fromEmail, s.fromName, s.forwardTo, MailboxForwardInput{
			FromAddress:    e.FromAddress,
			Subject:        e.Subject,
			BodyText:       e.BodyText,
			Platform:       e.Platform,
			ReceivedAt:     e.ReceivedAt,
			MaltActionTag:  actionTag,
			MaltActionNote: actionNote,
			AckURL:         s.ackURLPour(e),
			Reminder:       n,
			ReminderTotal:  len(rappelsDelais),
		})
		if err := s.relay(s.smtpAddr, s.fromEmail, s.forwardTo, msg); err != nil {
			log.Warn().Err(err).Str("message_id", e.MessageID).Int("rappel", n).Msg("mailbox: échec rappel — retenté au prochain cycle")
			s.watch.Echec(ctx, OrigineMalt, err.Error(), now)
			continue
		}
		s.watch.Succes(ctx, OrigineMalt, now)
		s.db.Model(&models.MailboxEmail{}).Where("id = ?", e.ID).
			Updates(map[string]interface{}{"reminders_sent": n, "last_reminder_at": now})
		log.Info().Str("message_id", e.MessageID).Int("rappel", n).Msg("mailbox: rappel envoyé")
	}
}

// rappelDu dit si le prochain rappel de e est échu à now : son délai depuis le transfert est passé
// ET le précédent rappel date d'au moins rappelEcartMin.
func rappelDu(e *models.MailboxEmail, now time.Time) bool {
	if e.ForwardedAt == nil || e.RemindersSent >= len(rappelsDelais) {
		return false
	}
	if now.Sub(*e.ForwardedAt) < rappelsDelais[e.RemindersSent] {
		return false
	}
	return e.LastReminderAt == nil || now.Sub(*e.LastReminderAt) >= rappelEcartMin
}

// ErrMailboxNotFound : l'id ne désigne aucun mail capté.
var ErrMailboxNotFound = errors.New("mailbox email not found")

// MarkHandled marque le mail id comme traité (clic sur « ✓ 标记为已处理 »). Idempotent : un second clic
// garde la date du PREMIER (c'est elle qui mesure le délai de réponse). déjà=true si c'était fait.
//
// Fonction de paquet (et non méthode) : le handler doit pouvoir marquer même quand le service IMAP
// n'est pas monté (credentials absents) — la donnée est en base, le lien a déjà été envoyé.
func MarkHandled(db *gorm.DB, id uuid.UUID, now time.Time) (deja bool, err error) {
	res := db.Model(&models.MailboxEmail{}).Where("id = ? AND handled_at IS NULL", id).Update("handled_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 1 {
		return false, nil
	}
	var n int64
	if err := db.Model(&models.MailboxEmail{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return false, err
	}
	if n == 0 {
		return false, ErrMailboxNotFound
	}
	return true, nil
}
