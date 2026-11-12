package config

import (
	"errors"
	"net"
	"os"
)

// SMTPRelayAddr rend l'adresse "hôte:port" du relai SMTP interne, partagé par les TROIS émetteurs de
// courrier du backend : le transfert des mails captés (services.MailboxService), le rappel de
// disponibilité Malt (services.MaltReminderService) et la newsletter du blog (api/newsletter.go).
//
// POURQUOI aucune valeur par défaut pour l'hôte. Jusqu'au 2026-08-30, les trois appelants repliaient
// chacun de leur côté sur une IP tailscale câblée en dur (celle de VPS57, qui hébergeait alors
// l'exim ; sa valeur exacte est conservée dans smtp_test.go, qui en interdit le retour).
// VPS57 s'est réenregistré dans le tailnet (nouvelle IP, l'ancien nœud restant « offline ») et son
// exim a été arrêté au profit du postfix de VPS142. L'adresse est devenue un trou noir : le
// transfert des offres Malt a échoué 13 jours d'affilée sans qu'aucun signal ne remonte, parce
// qu'un défaut mort est indiscernable d'une configuration valide. Aggravant : NEWSLETTER_SMTP_HOST
// n'était même pas câblée dans docker-compose.yml, donc le défaut était la SEULE valeur atteignable.
// Un hôte absent doit donc arrêter l'appelant, jamais le faire composer un numéro fantôme.
//
// COMMENT : l'hôte est obligatoire ; le port garde 25. L'asymétrie est voulue — se tromper de port
// produit un refus de connexion immédiat et bruyant, alors qu'un mauvais hôte peut absorber le
// trafic en silence pendant des semaines. Seul l'hôte est piégeux.
func SMTPRelayAddr() (string, error) {
	host := os.Getenv("NEWSLETTER_SMTP_HOST")
	if host == "" {
		return "", errors.New("NEWSLETTER_SMTP_HOST non configuré — le relai SMTP interne est requis " +
			"par tout envoi de courrier (transfert mailbox, rappel Malt, newsletter blog)")
	}

	port := os.Getenv("NEWSLETTER_SMTP_PORT")
	if port == "" {
		port = "25"
	}

	return net.JoinHostPort(host, port), nil
}
