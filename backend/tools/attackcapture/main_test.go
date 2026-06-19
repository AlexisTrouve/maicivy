package main

import "testing"

// Une ligne nginx combined typique du scanner doit se parser entièrement.
func TestParseLine_OK(t *testing.T) {
	line := `185.177.72.52 - - [06/Jun/2026:12:31:58 +0000] "GET /fr/blog/.env.local HTTP/1.1" 200 14172 "-" "curl/8.7.1"`
	ip, r, ok := parseLine(line)
	if !ok {
		t.Fatal("devrait parser")
	}
	if ip != "185.177.72.52" {
		t.Errorf("ip=%q", ip)
	}
	if r.M != "GET" || r.P != "/fr/blog/.env.local" {
		t.Errorf("method/path inattendus: %+v", r)
	}
	if r.St != 200 || r.B != 14172 {
		t.Errorf("st/b=%d/%d", r.St, r.B)
	}
	if r.UA != "curl/8.7.1" {
		t.Errorf("ua=%q", r.UA)
	}
}

// "-" comme nombre d'octets (réponse sans corps, ex: 444) → 0, pas d'erreur.
func TestParseLine_DashBytes(t *testing.T) {
	line := `1.2.3.4 - - [06/Jun/2026:12:00:00 +0000] "GET /app/.env HTTP/1.1" 444 - "-" "curl/8.7.1"`
	_, r, ok := parseLine(line)
	if !ok || r.St != 444 || r.B != 0 {
		t.Errorf("attendu st=444 b=0, got ok=%v %+v", ok, r)
	}
}

// stripQuery (anonymisation corpus légitime) retire bien le ?... et laisse le path nu intact.
func TestStripQuery(t *testing.T) {
	if got := stripQuery("/fr/projets?ref=email&q=secret"); got != "/fr/projets" {
		t.Errorf("query non retirée: %q", got)
	}
	if got := stripQuery("/fr/projets"); got != "/fr/projets" {
		t.Errorf("path nu altéré: %q", got)
	}
}

// Ligne malformée (handshake TLS sur le port HTTP) → rejetée proprement, pas de panic.
func TestParseLine_Malformed(t *testing.T) {
	if _, _, ok := parseLine(`1.2.3.4 - - [06/Jun/2026:12:00:00 +0000] "\x16\x03\x01" 400 0 "-" "-"`); ok {
		t.Error("une requête sans path devrait être rejetée")
	}
	if _, _, ok := parseLine("garbage line"); ok {
		t.Error("une ligne hors-format devrait être rejetée")
	}
}
