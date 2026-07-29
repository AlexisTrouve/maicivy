-- Filtre de pertinence LLM sur les mails Malt captes (cf. services.MailboxRelevanceEvaluator).
--
-- mailbox_emails existe deja en prod ; RunAutoMigrations() ne migre JAMAIS les tables existantes
-- (skip-if-exists, cf. internal/database/migrations.go) -- ce script doit etre applique manuellement
-- AVANT de deployer le nouveau binaire backend, sinon toute requete sur mailbox_emails echoue
-- (colonnes absentes referencees par le modele Go).
ALTER TABLE mailbox_emails ADD COLUMN IF NOT EXISTS is_opportunity BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE mailbox_emails ADD COLUMN IF NOT EXISTS relevance_score INTEGER;
ALTER TABLE mailbox_emails ADD COLUMN IF NOT EXISTS relevance_reason TEXT;
ALTER TABLE mailbox_emails ADD COLUMN IF NOT EXISTS forward_blocked BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_mailbox_emails_is_opportunity ON mailbox_emails (is_opportunity);
CREATE INDEX IF NOT EXISTS idx_mailbox_emails_forward_blocked ON mailbox_emails (forward_blocked);
