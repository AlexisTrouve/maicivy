package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DiscordNotifier poste un message sur un webhook Discord, avec ou sans mention de l'owner.
//
// POURQUOI un type dédié plutôt qu'un http.Post à l'appel : la règle « ne me ping que si quelque
// chose casse » doit être garantie par le TRANSPORT, pas par la discipline de chaque appelant. Une
// charge utile sans ping déclare `allowed_mentions.parse: []`, ce qui interdit à Discord de résoudre
// la moindre mention — même si le texte en contenait une par accident (un sujet de mail hostile, un
// « @everyone » recopié depuis une offre). Le silence est ainsi structurel, pas espéré.
type DiscordNotifier struct {
	webhookURL string
	userID     string // ID Discord de l'owner ; vide = on poste sans jamais mentionner personne
	client     *http.Client
}

// NewDiscordNotifier rend nil si l'URL est vide — même convention que les autres services optionnels
// (Gitea/GitLab stats, mailbox) : pas de flag séparé, l'absence de secret désactive la fonction.
func NewDiscordNotifier(webhookURL, userID string) *DiscordNotifier {
	if webhookURL == "" {
		return nil
	}
	return &DiscordNotifier{
		webhookURL: webhookURL,
		userID:     userID,
		// Timeout court : on notifie À PROPOS d'un envoi de courrier, on ne doit jamais retarder
		// l'envoi lui-même parce que Discord rame.
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// mentionsAutorisees construit le bloc allowed_mentions. Deux formes seulement, jamais d'entre-deux :
// - ping refusé  → {"parse": []}        : Discord ne notifie personne, quoi qu'il y ait dans le texte
// - ping accordé → {"users": ["<id>"]}  : seul l'owner est notifiable, jamais @everyone/@here
func (n *DiscordNotifier) mentionsAutorisees(ping bool) map[string]any {
	if ping && n.userID != "" {
		return map[string]any{"users": []string{n.userID}}
	}
	return map[string]any{"parse": []string{}}
}

// Post envoie le message. `ping` place la mention de l'owner EN TÊTE du message : sur mobile, une
// mention noyée au milieu d'un pavé passe inaperçue.
//
// Sans userID configuré, un ping demandé dégénère en simple message : mieux vaut une notification
// sans mention qu'un « <@> » cassé dans le fil.
func (n *DiscordNotifier) Post(ctx context.Context, contenu string, ping bool) error {
	if ping && n.userID != "" {
		contenu = fmt.Sprintf("<@%s> %s", n.userID, contenu)
	}

	charge, err := json.Marshal(map[string]any{
		"content":          contenu,
		"allowed_mentions": n.mentionsAutorisees(ping),
	})
	if err != nil {
		return fmt.Errorf("discord: encodage: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(charge))
	if err != nil {
		return fmt.Errorf("discord: requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: envoi: %w", err)
	}
	defer resp.Body.Close()

	// Discord répond 204 sur un webhook accepté ; tout le reste est un échec qu'on remonte à
	// l'appelant, à charge pour lui de le journaliser sans interrompre son propre travail.
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("discord: statut %d", resp.StatusCode)
	}
	return nil
}
