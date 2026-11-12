package services

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestIsBlockedFetchIP verrouille le classifieur anti-SSRF : toutes les plages internes/sensibles
// sont bloquées, les IP publiques passent.
func TestIsBlockedFetchIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", // loopback
		"10.1.2.3", "192.168.1.1", "172.16.0.1", // RFC1918
		"fd00::1",         // ULA
		"169.254.169.254", // metadata cloud (link-local)
		// Tailscale (CGNAT 100.64.0.0/10). Adresse GÉNÉRIQUE volontairement : c'est la plage
		// entière qui est bloquée, pas une machine précise — mettre l'IP réelle d'un hôte du
		// tailnet ne renforce en rien le test et divulguerait la topologie interne dans le
		// miroir GitHub public (cf. .claude/SYNC_GITHUB.md).
		"100.64.0.1",
		"0.0.0.0", // unspecified
	}
	for _, s := range blocked {
		require.True(t, isBlockedFetchIP(net.ParseIP(s)), "%s doit être bloqué (interne)", s)
	}

	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34"} // IP publiques
	for _, s := range allowed {
		require.False(t, isBlockedFetchIP(net.ParseIP(s)), "%s doit être autorisé (public)", s)
	}

	require.True(t, isBlockedFetchIP(nil), "IP nil (résolution échouée) doit être bloquée")
}

// TestFetchURLContent_BlocksLoopback verrouille le fix SSRF : un `offer` = URL pointant vers une
// IP loopback/interne DOIT être refusé. httptest bind sur 127.0.0.1, donc fetcher son URL revient
// à atteindre le réseau interne (ici le loopback) — exactement ce qu'un attaquant ferait avec
// `offer=http://maicivy-redis:6379/` ou `http://127.0.0.1:8081/...`. Le fetch doit échouer et ne
// JAMAIS renvoyer le contenu interne.
func TestFetchURLContent_BlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("SECRET-INTERNAL-RESPONSE"))
	}))
	defer srv.Close()

	// fetchURLContent n'utilise que le client HTTP ; on construit le service minimal.
	s := &CVGenerationService{httpClient: &http.Client{Timeout: 15 * time.Second}}

	out, err := s.fetchURLContent(srv.URL) // srv.URL = http://127.0.0.1:<port>
	require.Error(t, err, "fetch d'une IP loopback doit être refusé (anti-SSRF)")
	require.NotContains(t, out, "SECRET-INTERNAL-RESPONSE", "le contenu interne ne doit jamais fuiter")
}
