package services

import (
	_ "embed"
	"encoding/json"
	"strings"
)

// dossier_tessera.go — le dossier du REJEU Tessera, tel que le serveur le connaît.
//
// QUOI : le scénario de démonstration (frontend/lib/dossier-vivant/tessera.ts) replié en dossier
// remis : visiteur, sections, blocs, durée et coût enregistrés. La route publique du PDF du rejeu
// imprime CETTE copie, jamais un dossier envoyé par le navigateur.
//
// POURQUOI une copie et pas le scénario lui-même : le scénario est du TypeScript (horloge, rejeu,
// animation) ; le serveur n'a besoin que de son résultat. dossier_tessera.json est écrit et
// verrouillé par le test jest lib/dossier-vivant/__tests__/tesseraRecord.test.ts : modifier le
// scénario sans régénérer la copie fait échouer le test du front.
//
// COMMENT les deux données du scénario qui dépendent du rendu (tessera.ts › ReplayContext) sont
// portées : par deux jetons, remplacés ici au moment d'imprimer —
//   - tesseraLocaleToken dans le lien du site (`/{locale}` → `/fr`) ;
//   - tesseraVideoToken à la place de la vidéo de MelodiCode : la première preuve vidéo de sa carte
//     du sommet (data.ts › replayContext) ; sans vidéo, la preuve n'a pas de lien de démo.

//go:embed dossier_tessera.json
var dossierTesseraSrc []byte

const (
	tesseraLocaleToken = "{locale}"
	tesseraVideoToken  = "{melodicode-video}"
)

// tesseraRecord — le dossier du rejeu pour la langue `lang` et la montagne courante. Décodé à chaque
// appel : la copie embarquée n'est jamais modifiée en place.
func tesseraRecord(lang string, m *Montagne) *DossierRecord {
	var rec DossierRecord
	if err := json.Unmarshal(dossierTesseraSrc, &rec); err != nil {
		panic("dossier pdf: dossier Tessera embarqué illisible : " + err.Error())
	}
	video := ""
	for _, card := range m.Summit {
		if card.ID != "melodicode" {
			continue
		}
		for _, p := range card.Proofs {
			if p.Kind == "video" {
				video = p.URL
				break
			}
		}
	}
	for i := range rec.Sections {
		blocks := rec.Sections[i].Blocks
		for j := range blocks {
			d := blocks[j].Demo
			switch {
			case d == nil:
			case d.URL == tesseraVideoToken && video == "":
				blocks[j].Demo = nil
			case d.URL == tesseraVideoToken:
				d.URL = video
			default:
				d.URL = strings.ReplaceAll(d.URL, tesseraLocaleToken, lang)
			}
		}
	}
	return &rec
}
