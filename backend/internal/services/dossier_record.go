package services

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// dossier_record.go — la copie SERVEUR d'un dossier remis, et le document qu'on en imprime.
//
// QUOI : DossierRecord est ce que le serveur garde d'un dossier : le visiteur tel que le run l'a
// validé, les sections et les blocs tels que les agents les ont émis, la durée et le coût du run.
// PDFRequest en fait la requête de rendu (DossierPDFRequest) : libellés, titre et sous-titre dans la
// langue de l'interface, schéma mis en page à la largeur A4, liens de démo absolus.
//
// POURQUOI : les routes PUBLIQUES du PDF n'impriment que du contenu produit par le serveur. Le
// navigateur n'envoie plus qu'un identifiant et une langue ; tout le reste vient d'ici :
//   - un run réel : la copie enregistrée par dossiervivant.RecordingEmitter (dossier_store.go) ;
//   - le rejeu : la copie du scénario Tessera embarquée par le serveur (dossier_tessera.go).
// La route owner garde, elle, le contrat « corps = dossier » (DossierPDFRequest).
//
// COMMENT la fidélité au PDF d'avant est tenue : la requête construite ici est celle que le front
// construisait (pdfExport.ts › buildDossierPdfRequest + useDossierPdf) — mêmes libellés (copie des
// messages next-intl, verrouillée contre frontend/messages par test), même mise en page du schéma
// (dossier_print_layout.go), même formatage de la durée et du coût. Verrouillé par
// TestDossierRecord_MemeRenduQuAvant sur des requêtes produites par le front de 6f78f6c.

// DossierRecord — un dossier remis, gardé par le serveur (forme JSON stable : c'est aussi la forme
// stockée dans Redis et celle de dossier_tessera.json).
type DossierRecord struct {
	Visitor DossierRecordVisitor `json:"visitor"`
	// Agents — taille de l'équipe annoncée par run.started, orchestrateur compris (« par 6 agents »).
	Agents int `json:"agents"`
	// ElapsedSeconds — du premier agent au travail à la remise (runState.ts › elapsedAt).
	ElapsedSeconds float64 `json:"elapsedSeconds"`
	// CostUsd — somme des derniers compteurs de chaque agent (runState.ts › totalsAt).
	CostUsd  float64                `json:"costUsd"`
	Sections []DossierRecordSection `json:"sections"`
}

type DossierRecordVisitor struct {
	Name         string `json:"name"`
	Affiliation  string `json:"affiliation"`
	Organisation string `json:"organisation"`
}

type DossierRecordSection struct {
	ID     string               `json:"id"`
	Title  string               `json:"title"`
	Blocks []DossierRecordBlock `json:"blocks"`
}

// DossierRecordBlock — un bloc tel qu'émis (events.ts › DossierBlock) : le schéma y est une GRILLE
// (nœuds, arêtes), pas encore une géométrie.
type DossierRecordBlock struct {
	Type       string                `json:"type"` // lead | paragraph | item | pipeline | proof | also | contact
	Text       string                `json:"text,omitempty"`
	Nodes      []DossierPipelineNode `json:"nodes,omitempty"`
	Edges      []DossierPipelineEdge `json:"edges,omitempty"`
	Journal    string                `json:"journal,omitempty"`
	Label      string                `json:"label,omitempty"`
	ProjectID  string                `json:"projectId,omitempty"`
	Why        string                `json:"why,omitempty"`
	Demo       *DossierPDFDemo       `json:"demo,omitempty"`
	ProjectIDs []string              `json:"projectIds,omitempty"`
}

// DossierSectionIDs — les cinq sections, dans l'ordre de lecture (events.ts › SECTION_IDS).
var DossierSectionIDs = []string{"need", "architecture", "proofs", "contribution", "next"}

// --- Libellés du document (copie des messages next-intl) -----------------------------------------

// dossierPDFText — ce que le front tirait de next-intl pour le document : titre, gabarits de
// l'en-tête (`for`, `prepared`) et libellés. Langue = choix fermé de la requête.
type dossierPDFText struct {
	Title    string           `json:"title"`
	For      string           `json:"for"`
	Prepared string           `json:"prepared"`
	Labels   DossierPDFLabels `json:"labels"`
}

// dossier_pdf_labels.json — extrait de frontend/messages/{fr,en,de,it,zh}.json (FR = source).
// Régénéré et verrouillé par TestDossierPDFLabels_CopieDesMessages (DOSSIER_LABELS_WRITE=1).
//
//go:embed dossier_pdf_labels.json
var dossierPDFLabelsSrc []byte

var dossierPDFTexts = func() map[string]dossierPDFText {
	var t map[string]dossierPDFText
	if err := json.Unmarshal(dossierPDFLabelsSrc, &t); err != nil {
		panic("dossier pdf: libellés embarqués illisibles : " + err.Error())
	}
	return t
}()

// fillMessage — un gabarit ICU à arguments simples (`{name}`), rempli en une passe : une valeur qui
// contient elle-même `{affiliation}` n'est pas réinterprétée (comme next-intl).
func fillMessage(msg string, args map[string]string) string {
	pairs := make([]string, 0, 2*len(args))
	for k, v := range args {
		pairs = append(pairs, "{"+k+"}", v)
	}
	return strings.NewReplacer(pairs...).Replace(msg)
}

// formatElapsed — DossierVivant.tsx › fmtElapsed : m:ss, secondes tronquées.
func formatElapsed(s float64) string {
	return fmt.Sprintf("%d:%02d", int(math.Floor(s/60)), int(math.Floor(math.Mod(s, 60))))
}

// formatCost — Intl.NumberFormat(locale, { 3 décimales }) : virgule décimale en fr/de/it, point
// sinon ; milliers comme groupDigits.
func formatCost(v float64, lang string) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	whole, frac, _ := strings.Cut(s, ".")
	n, _ := strconv.Atoi(whole)
	dec := "."
	switch lang {
	case "fr", "de", "it":
		dec = ","
	}
	return groupDigits(n, lang) + dec + frac
}

// absoluteDemoURL — pdfExport.ts › absoluteDemoUrl : la vidéo est servie par maiProFiles, le site
// par l'origine publique. Un PDF n'a pas d'origine contre laquelle résoudre un chemin relatif.
func absoluteDemoURL(d DossierPDFDemo) string {
	if strings.HasPrefix(d.URL, "http://") || strings.HasPrefix(d.URL, "https://") {
		return d.URL
	}
	if d.Kind == "video" {
		return maiProFilesBaseURL + d.URL
	}
	return dossierSite + d.URL
}

// PDFRequest — le document de ce dossier dans la langue `lang` de l'interface. Langue hors liste :
// ErrDossierPDFInvalid (400). Le contenu (textes des agents, visiteur) reste celui du run.
func (r *DossierRecord) PDFRequest(lang string) (*DossierPDFRequest, error) {
	text, ok := dossierPDFTexts[lang]
	if !ok {
		return nil, invalid("lang %q", lang)
	}
	forLine := fillMessage(text.For, map[string]string{"name": r.Visitor.Name, "affiliation": r.Visitor.Affiliation})
	prepared := fillMessage(text.Prepared, map[string]string{
		"time": formatElapsed(r.ElapsedSeconds), "cost": formatCost(r.CostUsd, lang), "count": strconv.Itoa(r.Agents),
	})
	req := &DossierPDFRequest{Lang: lang, Title: text.Title, Subtitle: []string{forLine + ".", prepared}, Labels: text.Labels}
	for _, sec := range r.Sections {
		out := DossierPDFSection{ID: sec.ID, Title: sec.Title, Blocks: []DossierPDFBlock{}}
		for _, b := range sec.Blocks {
			switch b.Type {
			case "pipeline":
				if d := printDiagram(b.Nodes, b.Edges, b.Journal, b.Label); d != nil {
					out.Blocks = append(out.Blocks, DossierPDFBlock{Type: "diagram", Diagram: d})
				}
			case "proof":
				p := DossierPDFBlock{Type: "proof", ProjectID: b.ProjectID, Why: b.Why}
				if b.Demo != nil {
					p.Demo = &DossierPDFDemo{Kind: b.Demo.Kind, URL: absoluteDemoURL(*b.Demo)}
				}
				out.Blocks = append(out.Blocks, p)
			default:
				out.Blocks = append(out.Blocks, DossierPDFBlock{Type: b.Type, Text: b.Text, ProjectIDs: b.ProjectIDs})
			}
		}
		req.Sections = append(req.Sections, out)
	}
	return req, nil
}
