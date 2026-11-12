package services_test

import (
	"testing"

	"maicivy/internal/services"
)

func TestParseMailboxAllowlist(t *testing.T) {
	got := services.ParseMailboxAllowlist("malt.fr:malt,malt.com:malt, comet.co : comet,,bad-entry,empty-platform: ,:empty-domain")
	want := map[string]string{
		"malt.fr":  "malt",
		"malt.com": "malt",
		"comet.co": "comet",
	}
	if len(got) != len(want) {
		t.Fatalf("attendu %d entrées, got %d (%v)", len(want), len(got), got)
	}
	for domain, platform := range want {
		if got[domain] != platform {
			t.Errorf("domaine %q: attendu platform %q, got %q", domain, platform, got[domain])
		}
	}
}

func TestParseMailboxAllowlist_Empty(t *testing.T) {
	got := services.ParseMailboxAllowlist("")
	if len(got) != 0 {
		t.Fatalf("chaîne vide doit donner une allowlist vide, got %v", got)
	}
}

func TestMatchMailboxDomain_Exact(t *testing.T) {
	allow := services.ParseMailboxAllowlist("malt.fr:malt")
	platform, ok := services.MatchMailboxDomain("malt.fr", allow)
	if !ok || platform != "malt" {
		t.Fatalf("match exact attendu, got ok=%v platform=%q", ok, platform)
	}
}

func TestMatchMailboxDomain_Subdomain(t *testing.T) {
	allow := services.ParseMailboxAllowlist("malt.fr:malt")
	platform, ok := services.MatchMailboxDomain("notifications.malt.fr", allow)
	if !ok || platform != "malt" {
		t.Fatalf("sous-domaine doit matcher, got ok=%v platform=%q", ok, platform)
	}
}

// Bypass "evilmalt.fr" : un simple strings.HasSuffix(domain, allowed) sans frontière "." laisserait
// passer ce domaine (il se termine bien par "malt.fr" en tant que chaîne de caractères) — la règle
// exige une frontière "." stricte, donc ce domaine ne doit JAMAIS matcher.
func TestMatchMailboxDomain_RejectsSuffixBypass(t *testing.T) {
	allow := services.ParseMailboxAllowlist("malt.fr:malt")
	if platform, ok := services.MatchMailboxDomain("evilmalt.fr", allow); ok {
		t.Fatalf("evilmalt.fr ne doit PAS matcher malt.fr (bypass de frontière), got platform=%q", platform)
	}
}

func TestMatchMailboxDomain_Unlisted(t *testing.T) {
	allow := services.ParseMailboxAllowlist("malt.fr:malt")
	if _, ok := services.MatchMailboxDomain("gmail.com", allow); ok {
		t.Fatal("domaine non listé ne doit pas matcher")
	}
}

func TestMatchMailboxDomain_CaseInsensitive(t *testing.T) {
	allow := services.ParseMailboxAllowlist("malt.fr:malt")
	platform, ok := services.MatchMailboxDomain("Malt.FR", allow)
	if !ok || platform != "malt" {
		t.Fatalf("match insensible à la casse attendu, got ok=%v platform=%q", ok, platform)
	}
}

func TestEmailDomain(t *testing.T) {
	cases := map[string]string{
		"notifications@malt.fr": "malt.fr",
		"USER@Malt.FR":          "malt.fr",
		"no-at-sign":            "",
		"trailing@":             "",
		"":                      "",
	}
	for addr, want := range cases {
		if got := services.EmailDomain(addr); got != want {
			t.Errorf("EmailDomain(%q) = %q, attendu %q", addr, got, want)
		}
	}
}
