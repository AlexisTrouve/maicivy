package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : la photo embarquée dans le PDF du CV reste un JPEG léger.
// POURQUOI : elle pesait 1,85 Mo sur un PDF de 2,08 Mo (PNG 864×1184 affiché à 72 px, recompressé
// en Flate par Chromium — mesuré le 09/11). Un JPEG passe tel quel dans le PDF (DCTDecode) : 34 Ko.
// Remettre un PNG pleine définition referait gonfler CHAQUE CV envoyé, sans qu'aucun test de rendu
// ne le voie. Ce test vérifie l'invariant sans Chromium.
func TestCVPDF_PhotoEmbarqueeLegere(t *testing.T) {
	s := newPDFServiceFrom("../../templates/cv")
	require.True(t, s.hasTemplates)
	img := string(s.profileImg)
	require.NotEmpty(t, img, "photo de profil absente du gabarit")
	assert.True(t, strings.HasPrefix(img, "data:image/jpeg;base64,"), "JPEG attendu : Chromium l'embarque sans le recompresser")
	// base64 ≈ 4/3 de l'octet brut : 100 Ko de data URI ≈ 75 Ko d'image.
	assert.Less(t, len(img), 100_000, "photo de plus de ~75 Ko : elle redevient le poids du PDF")
}
