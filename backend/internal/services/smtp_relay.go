package services

import (
	"crypto/tls"
	"net"
	"net/smtp"
)

// SendViaRelay envoie un message RFC822 déjà composé via le relai SMTP interne (exim VPS57).
//
// POURQUOI pas smtp.SendMail : il fait un STARTTLS et VÉRIFIE le cert contre l'hôte (ici une IP
// Tailscale) ; l'exim présente un cert pour un hostname → échec "no IP SANs". COMMENT : on fait le
// STARTTLS à la main avec InsecureSkipVerify — c'est un relai INTERNE vers NOTRE propre exim, sur un
// lien Tailscale DÉJÀ chiffré (WireGuard) ; vérifier le cert d'un relai maison n'apporte rien.
//
// Extrait de internal/api/newsletter.go (partagé avec l'ingestion mailbox — même relai exim).
func SendViaRelay(addr, from, to string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.Hello("maicivy.etheryale.com"); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		host, _, _ := net.SplitHostPort(addr)
		if err := c.StartTLS(&tls.Config{ServerName: host, InsecureSkipVerify: true}); err != nil { // #nosec G402 — relai interne Tailscale
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
