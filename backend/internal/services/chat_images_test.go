package services

import (
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : valide les images jointes à un message et leur transformation en blocs de contenu.
//
// POURQUOI cette fonctionnalité : sur l'écran compose, l'opératrice doit pouvoir coller une CAPTURE
// d'écran — les questions de formulaire de la plateforme, ou une annonce dont le texte n'est pas
// sélectionnable. La vision n'est pas un tool chez Anthropic : l'image entre directement dans le
// message, à côté du texte.
//
// POURQUOI un REJET explicite et non un filtrage silencieux : écarter une image en douce ferait
// répondre l'agent à côté sans que personne comprenne pourquoi — l'opératrice verrait une réponse
// hors sujet et croirait le modèle idiot. Doctrine du projet : échec franc plutôt que fallback.

func imageDeTaille(octets int) ChatImage {
	return ChatImage{MediaType: "image/png", Data: strings.Repeat("A", octets)}
}

func TestValidateChatImages_TypesAcceptes(t *testing.T) {
	for _, mt := range []string{"image/png", "image/jpeg", "image/gif", "image/webp"} {
		t.Run(mt, func(t *testing.T) {
			err := ValidateChatImages([]ChatImage{{MediaType: mt, Data: "AAAA"}})
			assert.NoError(t, err)
		})
	}
}

// Un type non supporté par l'API doit être refusé ICI, avec un code lisible, plutôt que de partir
// chez Anthropic et revenir en erreur générique incompréhensible pour l'opératrice.
func TestValidateChatImages_TypeRefuse(t *testing.T) {
	err := ValidateChatImages([]ChatImage{{MediaType: "image/bmp", Data: "AAAA"}})

	require.Error(t, err)
	assert.Equal(t, ImageErrUnsupportedType, err.Error())
}

func TestValidateChatImages_TropDImages(t *testing.T) {
	trop := make([]ChatImage, maxImagesParMessage+1)
	for i := range trop {
		trop[i] = imageDeTaille(10)
	}

	err := ValidateChatImages(trop)

	require.Error(t, err)
	assert.Equal(t, ImageErrTooMany, err.Error())
}

// Une capture 4K en base64 dépasse vite plusieurs mégaoctets. Sans plafond, une seule requête peut
// faire exploser la limite de taille de l'API — et le message d'erreur renvoyé serait illisible.
func TestValidateChatImages_TropLourde(t *testing.T) {
	err := ValidateChatImages([]ChatImage{imageDeTaille(maxImageBase64Octets + 1)})

	require.Error(t, err)
	assert.Equal(t, ImageErrTooLarge, err.Error())
}

func TestValidateChatImages_DonneesVides(t *testing.T) {
	err := ValidateChatImages([]ChatImage{{MediaType: "image/png", Data: "  "}})

	require.Error(t, err)
	assert.Equal(t, ImageErrEmpty, err.Error())
}

func TestValidateChatImages_AucuneImage(t *testing.T) {
	assert.NoError(t, ValidateChatImages(nil))
}

// Les images précèdent le texte dans le message : c'est la convention de l'API, et c'est aussi ce qui
// fait que le modèle lit la capture AVANT la consigne qui s'y réfère.
func TestBuildUserBlocks_ImagesAvantLeTexte(t *testing.T) {
	blocks := buildUserBlocks("Que dois-je mettre dans ces champs ?", []ChatImage{
		{MediaType: "image/png", Data: "AAAA"},
	})

	require.Len(t, blocks, 2)
	assert.NotNil(t, blocks[0].OfImage, "l'image doit venir en premier")
	require.NotNil(t, blocks[1].OfText, "le texte doit suivre l'image")
	assert.Equal(t, "Que dois-je mettre dans ces champs ?", blocks[1].OfText.Text)
}

// Image SEULE, sans texte : l'opératrice colle une capture et envoie sans rien écrire.
//
// POURQUOI ce cas mérite son test : l'API REFUSE un bloc de texte vide. Ajouter un bloc texte "" à
// côté de l'image ferait échouer tout l'appel — et l'erreur remontée serait incompréhensible pour
// quelqu'un qui a juste collé une capture.
func TestBuildUserBlocks_ImageSeuleSansTexte(t *testing.T) {
	blocks := buildUserBlocks("   ", []ChatImage{{MediaType: "image/png", Data: "AAAA"}})

	require.Len(t, blocks, 1, "un bloc de texte vide accompagne l'image et fait rejeter la requête")
	assert.NotNil(t, blocks[0].OfImage)
}

// Sans image, on retombe exactement sur le comportement d'avant : un seul bloc de texte. Le chat
// public ne doit rien voir changer.
func TestBuildUserBlocks_SansImage(t *testing.T) {
	blocks := buildUserBlocks("Bonjour", nil)

	require.Len(t, blocks, 1)
	require.NotNil(t, blocks[0].OfText)
	assert.Equal(t, "Bonjour", blocks[0].OfText.Text)
}

// Garde-fou de type : le SDK n'accepte que quatre types de média, la conversion doit les porter tels
// quels sans les réinventer.
func TestBuildUserBlocks_PorteLeTypeDeMedia(t *testing.T) {
	blocks := buildUserBlocks("x", []ChatImage{{MediaType: "image/webp", Data: "BBBB"}})

	require.NotNil(t, blocks[0].OfImage)
	assert.Equal(t, anthropic.Base64ImageSourceMediaType("image/webp"),
		blocks[0].OfImage.Source.OfBase64.MediaType)
}
