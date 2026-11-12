package dossiervivant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Panne réelle du 2026-11-07 (prod, reader) : le modèle a rendu un retour à la ligne BRUT dans une
// chaîne JSON — « invalid character '\n' in string literal » — et le run entier s'est arrêté.
// Un caractère de contrôle dans une chaîne n'a qu'une lecture possible : on l'échappe, on ne jette
// pas le run. Hors chaîne, un saut de ligne est un blanc légal et doit rester tel quel.
func TestDecodeJSON_ControleBrutDansUneChaine(t *testing.T) {
	var out struct {
		Notes []string `json:"notes"`
		Note  string   `json:"note"`
	}
	raw := "{\n  \"notes\": [\"ligne un\nligne deux\", \"tab\there\"],\n  \"note\": \"a\r\nb \\\"cité\\\" \\n déjà\"\n}"
	require.NoError(t, decodeJSON(raw, &out))
	assert.Equal(t, []string{"ligne un\nligne deux", "tab\there"}, out.Notes)
	assert.Equal(t, "a\r\nb \"cité\" \n déjà", out.Note)
}

// Un JSON vraiment cassé reste une erreur : la tolérance ne vise que les contrôles dans une chaîne.
func TestDecodeJSON_CasseResteUneErreur(t *testing.T) {
	var out map[string]any
	assert.Error(t, decodeJSON(`{"a": "ouvert`, &out))
	assert.Error(t, decodeJSON(`{"a": tru}`, &out))
}
