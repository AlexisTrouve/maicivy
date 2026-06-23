package api

import "testing"

// Verrouille le fix de la boucle infinie d'escapeXML (flux RSS). L'ancienne implémentation
// remplaçait "&" par "&amp;" puis re-cherchait "&" dans le résultat → "&amp;" contient "&" → boucle
// INFINIE dès qu'un titre/résumé contenait un "&" (hang 60s → 504, goroutine qui spinne = mini-DoS).
// Avec le code buggé, ce test HANGAIT (timeout) ; il passe désormais et bloque toute régression.
func TestEscapeXML(t *testing.T) {
	cases := map[string]string{
		"Tom & Jerry":     "Tom &amp; Jerry",
		"a < b > c":       "a &lt; b &gt; c",
		`"q" 'a'`:         "&quot;q&quot; &apos;a&apos;",
		"R&D & AI & ML":   "R&amp;D &amp; AI &amp; ML", // plusieurs "&" : un seul passage, aucun re-scan
		"Full-Stack & AI": "Full-Stack &amp; AI",
		"no special":      "no special",
		"":                "",
	}
	for in, want := range cases {
		if got := escapeXML(in); got != want {
			t.Errorf("escapeXML(%q) = %q, want %q", in, got, want)
		}
	}
}
