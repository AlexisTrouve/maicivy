package services

import (
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// Images jointes à un message de chat.
//
// POURQUOI : sur l'écran compose, l'opératrice doit pouvoir coller une CAPTURE — les questions de
// formulaire de la plateforme, ou une annonce dont le texte n'est pas sélectionnable. Chez Anthropic
// la vision n'est PAS un tool : il n'y a rien à apprendre au modèle, l'image entre directement dans le
// message à côté du texte et il la lit nativement.
//
// POURQUOI un rejet explicite plutôt qu'un filtrage silencieux : écarter une image en douce ferait
// répondre l'agent à côté sans que personne comprenne pourquoi — l'opératrice verrait une réponse hors
// sujet et croirait le modèle idiot. Doctrine du projet : échec franc plutôt que fallback.

// Codes d'erreur STABLES : le frontend les traduit dans la langue de l'opérateur (règle i18n — jamais
// de phrase en dur côté backend, qui ignore sa langue).
const (
	ImageErrUnsupportedType = "image_type_unsupported"
	ImageErrTooMany         = "image_too_many"
	ImageErrTooLarge        = "image_too_large"
	ImageErrEmpty           = "image_empty"
)

const (
	// maxImagesParMessage : au-delà, c'est un usage qu'on n'a pas prévu — et chaque image coûte des
	// tokens de vision. Quatre couvre largement un formulaire découpé en plusieurs captures.
	maxImagesParMessage = 4
	// maxImageBase64Octets : plafond par image, en octets de base64. ~6 Mo encodés ≈ 4,5 Mo réels,
	// soit largement une capture 4K, tout en gardant une marge confortable sous la limite de taille
	// de requête de l'API (32 Mo).
	maxImageBase64Octets = 6 * 1024 * 1024
)

// typesImageAcceptes : exactement ceux que l'API accepte. En refuser d'autres ICI donne un message
// lisible, au lieu d'une erreur générique renvoyée par Anthropic que l'opératrice ne saurait pas lire.
var typesImageAcceptes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

// ChatImage : une image jointe, en base64 NU (sans le préfixe `data:...;base64,`).
type ChatImage struct {
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// ValidateChatImages : refuse tout ce que l'API ou le bon sens n'accepte pas. Retourne une erreur dont
// le message est un CODE stable, jamais une phrase traduite.
func ValidateChatImages(images []ChatImage) error {
	if len(images) == 0 {
		return nil
	}
	if len(images) > maxImagesParMessage {
		return errors.New(ImageErrTooMany)
	}
	for _, img := range images {
		if !typesImageAcceptes[img.MediaType] {
			return errors.New(ImageErrUnsupportedType)
		}
		if strings.TrimSpace(img.Data) == "" {
			return errors.New(ImageErrEmpty)
		}
		if len(img.Data) > maxImageBase64Octets {
			return errors.New(ImageErrTooLarge)
		}
	}
	return nil
}

// buildUserBlocks : assemble un tour utilisateur en blocs de contenu.
//
// COMMENT : les images d'ABORD, le texte ensuite. C'est la convention de l'API, et c'est aussi ce qui
// fait que le modèle regarde la capture avant de lire la consigne qui s'y réfère. Sans image, on
// retombe sur un unique bloc de texte — le chat public ne voit aucun changement.
// COMMENT (suite) : un bloc de texte VIDE est refusé par l'API. Quand l'opérateur envoie une capture
// sans rien écrire — geste légitime : « regarde ça » — on n'ajoute donc pas de bloc texte du tout.
// Sans image, en revanche, on garde le bloc quoi qu'il arrive : le chat public passe toujours par là
// et son comportement doit rester strictement identique.
func buildUserBlocks(text string, images []ChatImage) []anthropic.ContentBlockParamUnion {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(images)+1)
	for _, img := range images {
		blocks = append(blocks, anthropic.NewImageBlockBase64(img.MediaType, img.Data))
	}
	if len(images) == 0 || strings.TrimSpace(text) != "" {
		blocks = append(blocks, anthropic.NewTextBlock(text))
	}
	return blocks
}
