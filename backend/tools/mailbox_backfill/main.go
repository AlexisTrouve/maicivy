// Command mailbox_backfill importe des mails Malt historiques (identifiés par UID IMAP précis) dans
// mailbox_emails, avec jugement de pertinence RÉEL — pour rattraper ce qui existait déjà dans la
// boîte AVANT l'activation du curseur (cf. services.MailboxService : jamais de backfill auto au 1er
// boot, décision volontaire pour ne jamais transférer l'historique existant).
//
// QUOI    : fetch un ensemble d'UID précis (--uids), insère chaque mail + verdict de pertinence,
//
//	FORCE ForwardBlocked=true sur chaque ligne — jamais transféré automatiquement : c'est du
//	passé, pas la peine de spammer la queue de dispatch (Tingting) avec du périmé. Un
//	"Forcer l'envoi" manuel depuis le panel admin reste possible au cas par cas.
//
// POURQUOI: réutilisable pour un futur backfill (autre plage d'UID, ex: après avoir élargi
//
//	MAILBOX_ALLOWED_DOMAINS à une nouvelle plateforme) sans reconstruire ce script à la main.
//
// COMMENT : réutilise les VRAIS chemins de code prod (GmailImapFetcher, MailboxRelevanceService,
//
//	PortfolioService) — pas une réimplémentation. Connexion DB/IMAP/LLM via env vars (mêmes
//	noms que le service prod), cible via --uids.
//
// Usage (identifier les UID à backfiller au préalable via un script IMAP séparé, ex: recherche par
// expéditeur sur la boîte) :
//
//	BACKFILL_DB_DSN="host=... user=... password=... dbname=... sslmode=disable" \
//	MAILBOX_IMAP_USER=... MAILBOX_IMAP_APP_PASSWORD=... \
//	ANTHROPIC_BASE_URL=... ANTHROPIC_API_KEY=... \
//	go run ./tools/mailbox_backfill --uids=35385,35392,35396 --platform=malt
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

const maxBodyBytes = 200 * 1024

func main() {
	uidsFlag := flag.String("uids", "", "liste d'UID IMAP à backfiller, séparés par des virgules (requis)")
	platform := flag.String("platform", "malt", "label plateforme stocké sur chaque ligne backfillée")
	flag.Parse()

	uids, err := parseUIDs(*uidsFlag)
	if err != nil {
		log.Fatalf("--uids: %v", err)
	}
	if len(uids) == 0 {
		log.Fatal("--uids requis, ex: --uids=35385,35392,35396")
	}

	ctx := context.Background()

	dsn := os.Getenv("BACKFILL_DB_DSN")
	if dsn == "" {
		log.Fatal("BACKFILL_DB_DSN requis")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	fetcher, err := services.NewGmailImapFetcher(
		"imap.gmail.com", 993,
		os.Getenv("MAILBOX_IMAP_USER"), os.Getenv("MAILBOX_IMAP_APP_PASSWORD"), "INBOX",
	)
	if err != nil {
		log.Fatalf("imap connect: %v", err)
	}
	defer fetcher.Close()

	relevance := services.NewMailboxRelevanceService(
		os.Getenv("ANTHROPIC_BASE_URL"), os.Getenv("ANTHROPIC_API_KEY"),
		services.NewPortfolioService(), 50,
	)
	if relevance == nil {
		log.Fatal("relevance service non configuré — vérifier ANTHROPIC_BASE_URL/API_KEY")
	}

	messages, err := fetcher.FetchByUIDs(ctx, uids)
	if err != nil {
		log.Fatalf("fetch by uids: %v", err)
	}
	fmt.Printf("Fetched %d/%d messages\n", len(messages), len(uids))

	for _, m := range messages {
		domain := services.EmailDomain(m.From)
		messageID := m.MessageID
		if messageID == "" {
			messageID = fmt.Sprintf("imap-uid-%d@synthetic.mailbox", m.UID)
		}

		var count int64
		db.Model(&models.MailboxEmail{}).Where("message_id = ?", messageID).Count(&count)
		if count > 0 {
			fmt.Printf("UID %d: déjà présent, skip\n", m.UID)
			continue
		}

		body := m.BodyText
		if len(body) > maxBodyBytes {
			body = body[:maxBodyBytes]
		}

		email := models.MailboxEmail{
			MessageID:      messageID,
			ImapUID:        m.UID,
			FromAddress:    m.From,
			FromDomain:     domain,
			Platform:       *platform,
			Subject:        m.Subject,
			BodyText:       body,
			ReceivedAt:     m.ReceivedAt,
			ForwardBlocked: true, // backfill historique — jamais auto-transféré, cf. commentaire en tête
		}

		verdict, err := relevance.Evaluate(ctx, email.Subject, email.BodyText)
		if err != nil {
			fmt.Printf("UID %d: échec jugement pertinence: %v (stocké sans verdict)\n", m.UID, err)
		} else {
			email.IsOpportunity = verdict.IsOpportunity
			email.RelevanceReason = verdict.Reason
			email.RelevanceCoT = verdict.CoT
			email.RelevanceLink = verdict.Link
			if verdict.IsOpportunity {
				score := verdict.Score
				email.RelevanceScore = &score
			}
		}

		if err := db.Create(&email).Error; err != nil {
			fmt.Printf("UID %d: échec insertion: %v\n", m.UID, err)
			continue
		}
		scoreStr := "n/a"
		if email.RelevanceScore != nil {
			scoreStr = fmt.Sprintf("%d", *email.RelevanceScore)
		}
		fmt.Printf("UID %d: inséré — opportunity=%v score=%s subject=%q\n",
			m.UID, email.IsOpportunity, scoreStr, truncate(email.Subject, 60))
		time.Sleep(500 * time.Millisecond) // ménage le proxy Anthropic, pas de rafale
	}

	fmt.Println("Done.")
}

// parseUIDs parse une liste CSV d'UID ("35385,35392, 35396") en []uint32. CSV vide → slice vide,
// pas une erreur (c'est main() qui décide qu'--uids est requis).
func parseUIDs(csv string) ([]uint32, error) {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil, nil
	}
	parts := strings.Split(csv, ",")
	uids := make([]uint32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("uid invalide %q: %w", p, err)
		}
		uids = append(uids, uint32(n))
	}
	return uids, nil
}

// truncate coupe une string à n runes pour l'affichage console (sujets parfois très longs).
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
