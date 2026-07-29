package config

import "os"

// MailboxConfig — ingestion IMAP de la boîte pro existante (où Malt notifie déjà) + dispatch
// automatique vers une adresse tierce fixe. ImapUser/ImapAppPassword/ForwardTo vides → service non
// configuré (cf. Configured) : main.go construit alors le service à nil et ne démarre pas le job de
// poll — même convention que les services Gitea/GitLab stats optionnels. Pas de flag MAILBOX_ENABLED
// séparé, la présence des credentials EST le flag.
type MailboxConfig struct {
	ImapHost            string
	ImapPort            int
	ImapUser            string
	ImapAppPassword     string
	ImapFolder          string
	AllowedDomains      string // "malt.fr:malt,malt.com:malt" — parsé par services.ParseMailboxAllowlist
	ForwardTo           string
	ForwardFromEmail    string
	ForwardFromName     string
	PollIntervalSeconds int
	RelevanceThreshold  int // score minimal (0-100) pour qu'une opportunité soit transférée auto
}

func LoadMailboxConfig() *MailboxConfig {
	return &MailboxConfig{
		ImapHost:            getEnvOrDefault("MAILBOX_IMAP_HOST", "imap.gmail.com"),
		ImapPort:            getEnvAsIntOrDefault("MAILBOX_IMAP_PORT", 993),
		ImapUser:            os.Getenv("MAILBOX_IMAP_USER"),
		ImapAppPassword:     os.Getenv("MAILBOX_IMAP_APP_PASSWORD"),
		ImapFolder:          getEnvOrDefault("MAILBOX_IMAP_FOLDER", "INBOX"),
		AllowedDomains:      getEnvOrDefault("MAILBOX_ALLOWED_DOMAINS", "malt.fr:malt,malt.com:malt"),
		ForwardTo:           os.Getenv("MAILBOX_FORWARD_TO"),
		ForwardFromEmail:    getEnvOrDefault("MAILBOX_FORWARD_FROM_EMAIL", "mailbox@etheryale.com"),
		ForwardFromName:     getEnvOrDefault("MAILBOX_FORWARD_FROM_NAME", "Mailbox Etheryale"),
		PollIntervalSeconds: getEnvAsIntOrDefault("MAILBOX_POLL_INTERVAL_SECONDS", 120),
		RelevanceThreshold:  getEnvAsIntOrDefault("MAILBOX_RELEVANCE_THRESHOLD", 50),
	}
}

// Configured dit si les credentials nécessaires au service d'ingestion sont présents.
func (c *MailboxConfig) Configured() bool {
	return c.ImapUser != "" && c.ImapAppPassword != "" && c.ForwardTo != ""
}
