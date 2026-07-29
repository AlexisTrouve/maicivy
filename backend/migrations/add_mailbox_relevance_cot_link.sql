-- Filtre de pertinence LLM agentic : ajoute le raisonnement (CoT) et le lien de mission extraits par
-- l'agent (cf. services.MailboxRelevanceService, reecriture agentic multi-tours).
--
-- mailbox_emails existe deja en prod ; RunAutoMigrations() ne migre JAMAIS les tables existantes
-- (skip-if-exists, cf. internal/database/migrations.go) -- ce script doit etre applique manuellement
-- AVANT de deployer le nouveau binaire backend, sinon toute requete sur mailbox_emails echoue
-- (colonnes absentes referencees par le modele Go).
ALTER TABLE mailbox_emails ADD COLUMN IF NOT EXISTS relevance_cot TEXT;
ALTER TABLE mailbox_emails ADD COLUMN IF NOT EXISTS relevance_link TEXT;
