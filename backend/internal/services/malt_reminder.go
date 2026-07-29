package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// MaltReminderService envoie un email de rappel — "pense à mettre à jour ta disponibilité sur Malt" —
// à jours/heure fixes chaque semaine. Malt n'a pas d'API pour lire/écrire la disponibilité, donc
// aucune automatisation possible côté produit : juste un rappel humain, pas d'agent, pas de logique
// métier — cf. décision explicite d'Alexi (pas de cloud agent pour ça).
type MaltReminderService struct {
	redis     *redis.Client
	relay     MailboxRelaySender // réutilise le même type que le forward mailbox — nil → SendViaRelay
	smtpAddr  string
	to        string
	fromEmail string
	fromName  string
	days      map[time.Weekday]bool
	hourUTC   int
}

// NewMaltReminderService — nil si to est vide (pas de destinataire configuré, feature désactivée).
// relay=nil → SendViaRelay (le vrai relai), même convention que MailboxService.
func NewMaltReminderService(redisClient *redis.Client, smtpAddr, to, fromEmail, fromName string, days []time.Weekday, hourUTC int, relay MailboxRelaySender) *MaltReminderService {
	if to == "" {
		return nil
	}
	if relay == nil {
		relay = SendViaRelay
	}
	dayset := make(map[time.Weekday]bool, len(days))
	for _, d := range days {
		dayset[d] = true
	}
	return &MaltReminderService{
		redis: redisClient, relay: relay, smtpAddr: smtpAddr, to: to,
		fromEmail: fromEmail, fromName: fromName, days: dayset, hourUTC: hourUTC,
	}
}

// CheckAndSend envoie le rappel si (1) `now` tombe un jour configuré, (2) l'heure UTC courante a
// atteint l'heure configurée, ET (3) rien n'a encore été envoyé avec succès aujourd'hui.
//
// COMMENT : le job appelant tourne toutes les heures (cf. jobs.MaltReminderJob) — le check Redis par
// date évite un envoi par heure de la fenêtre restante du jour. La clé n'est posée qu'APRÈS un envoi
// RÉUSSI (pas avant) : un échec SMTP transitoire est donc retenté au prochain cycle la même journée,
// au lieu d'être silencieusement perdu jusqu'au prochain jour configuré.
func (s *MaltReminderService) CheckAndSend(ctx context.Context, now time.Time) {
	now = now.UTC()
	if !s.days[now.Weekday()] || now.Hour() < s.hourUTC {
		return
	}

	key := fmt.Sprintf("maicivy:malt_reminder:sent:%s", now.Format("2006-01-02"))
	already, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		log.Error().Err(err).Msg("malt reminder: échec vérification Redis")
		return
	}
	if already > 0 {
		return // déjà envoyé aujourd'hui
	}

	msg := renderMaltReminder(s.fromEmail, s.fromName, s.to)
	if err := s.relay(s.smtpAddr, s.fromEmail, s.to, msg); err != nil {
		log.Error().Err(err).Msg("malt reminder: échec envoi — retenté au prochain cycle")
		return
	}

	// TTL 25h (pas 24h pile) : marge contre un léger décalage d'horloge/ticker qui ferait relire la
	// clé une seconde avant son expiration exacte à minuit.
	if err := s.redis.Set(ctx, key, "1", 25*time.Hour).Err(); err != nil {
		log.Error().Err(err).Msg("malt reminder: échec écriture Redis post-envoi")
	}
	log.Info().Msg("malt reminder: envoyé")
}

// renderMaltReminder compose le message RFC822 — texte fixe, aucun contenu dynamique (ce n'est pas
// une notification métier, juste un rappel récurrent identique). En ANGLAIS : le destinataire
// (Tingting, cf. MALT_REMINDER_TO) ne lit pas le français — même raison que la traduction des mails
// mailbox (cf. mailbox_translation.go).
func renderMaltReminder(fromEmail, fromName, to string) []byte {
	subject := "Reminder: Malt availability"
	text := "Automatic reminder (2x/week): remember to update your availability on Malt.\r\n\r\nhttps://www.malt.fr\r\n"

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s <%s>\r\n", MimeWord(fromName), fromEmail)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", MimeWord(subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@etheryale.com>\r\n", RandHex16())
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(text)
	return []byte(b.String())
}
