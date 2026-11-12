package config

import "os"

// MailWatchConfig règle la surveillance du courrier sortant : où notifier, qui pinguer, et à partir
// de quand un mail non transféré cesse d'être « en attente » pour devenir « bloqué ».
//
// POURQUOI cette surveillance existe : du 13 au 30 août 2026, aucune offre Malt n'a été transférée.
// Les mails étaient captés, l'envoi échouait en boucle, et l'échec ne vivait que dans les logs du
// conteneur — personne ne les lit tant que rien n'attire l'attention. Il manquait un canal qui
// sorte de la machine.
type MailWatchConfig struct {
	WebhookURL    string // vide = surveillance désactivée (rien n'est posté, les compteurs restent)
	OwnerUserID   string // ID Discord à mentionner sur une panne ; vide = alerte sans mention
	StuckHours    int    // au-delà, un mail non transféré est signalé comme bloqué
	DigestHourUTC int    // heure UTC de publication du récapitulatif quotidien
}

// LoadMailWatchConfig lit l'environnement.
//
// Les défauts sont choisis pour ne pas être bavards : le récapitulatif part en fin de journée, et le
// seuil de blocage (6 h) est très au-dessus du cycle de poll (2 min) — un mail qui attend six heures
// n'attend pas son tour, il est coincé.
func LoadMailWatchConfig() MailWatchConfig {
	return MailWatchConfig{
		WebhookURL:    os.Getenv("DISCORD_WEBHOOK_URL"),
		OwnerUserID:   os.Getenv("DISCORD_OWNER_USER_ID"),
		StuckHours:    getEnvAsIntOrDefault("MAIL_WATCH_STUCK_HOURS", 6),
		DigestHourUTC: getEnvAsIntOrDefault("MAIL_DIGEST_HOUR_UTC", 21),
	}
}
