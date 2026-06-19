package services

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// maxFetchBytes borne la taille d'une URL d'offre fetchée (anti-DoS mémoire sur io.ReadAll :
// sans cap, une réponse géante saturerait la RAM du conteneur). 2 MiB suffit pour une page d'offre.
const maxFetchBytes = 2 << 20 // 2 MiB

// isBlockedFetchIP dit si une IP est INTERDITE pour un fetch d'URL fournie par l'utilisateur.
// QUOI : retourne true pour toute IP interne/sensible à ne jamais joindre depuis un offer=http://...
// POURQUOI : anti-SSRF. Un offer-URL ne doit jamais atteindre le loopback (le backend lui-même), le
// réseau Docker interne (redis/postgres), l'endpoint de métadonnées cloud (169.254.169.254) ni le
// réseau Tailscale (100.64.0.0/10, où vit projectmind) — sinon SSRF = scan/exfil du réseau interne.
// COMMENT : rejette loopback (127/8, ::1), link-local (couvre 169.254.x + fe80::), privées
// RFC1918/ULA (via IsPrivate), unspecified (0.0.0.0/::), multicast, et CGNAT/Tailscale 100.64/10.
func isBlockedFetchIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	// CGNAT 100.64.0.0/10 — plage utilisée par Tailscale (réseau interne d'Alexi).
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
		return true
	}
	return false
}

// newSSRFSafeClient construit un http.Client durci anti-SSRF pour fetcher des URLs utilisateur.
// QUOI : un client qui ne se connecte JAMAIS à une IP interne et ne suit AUCUNE redirection.
// POURQUOI : valider l'IP après un simple LookupIP puis dialer le host laisse une fenêtre de
// DNS-rebinding (IP publique à la validation, IP privée au dial). On valide donc à la connexion.
// COMMENT : net.Dialer.Control reçoit l'adresse DÉJÀ résolue (ip:port) juste avant le connect TCP,
// on y rejette les IP internes ; chaque hop de redirect re-dial (re-checké), mais on les refuse.
func newSSRFSafeClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || isBlockedFetchIP(ip) {
				return fmt.Errorf("fetch refusé: IP interne/privée non autorisée (%s)", host)
			}
			return nil
		},
	}
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return fmt.Errorf("redirections interdites pour le fetch d'offre (anti-SSRF)")
		},
		Transport: &http.Transport{DialContext: dialer.DialContext},
	}
}
