package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
)

// Recette du NOMBRE DE PAGES du CV taillé (POST /cv/generate, format pdf).
//
// CONTEXTE (mesuré sur la prod le 11/11/2026) : l'export standard tient en 2 pages pour les six
// thèmes en fr et en, mais sa page 2 est pleine à ~99 %. Le CV taillé y ajoute un résumé rédigé
// par le modèle (5 lignes contre 4 pour le pitch) et des accroches réécrites plus longues : le
// pied « maicivy.etheryale.com » basculait seul sur une page 3, vide, en fr comme en en.

// resumesTailles : résumés de la longueur réellement observée en prod, puis plus longs d'une
// phrase — la consigne en autorise trois. Chacun dans la langue du CV.
var resumesTailles = map[string]map[string]string{
	"fr": {
		"observe": "Je construis des systèmes d'agents qui livrent en production. Une équipe d'agents IA qui pousse " +
			"du code sous preuve E2E et test de mutation, des serveurs MCP, des orchestrateurs Claude en parallèle. " +
			"Python, Go, Postgres, Docker : l'agent comme brique d'architecture, avec l'évaluation pour le garder honnête.",
		"long": "Je construis des systèmes d'agents qui livrent en production. Une équipe d'agents IA qui pousse " +
			"du code sous preuve E2E et test de mutation, des serveurs MCP, des orchestrateurs Claude en parallèle. " +
			"Python, Go, Postgres, Docker : l'agent comme brique d'architecture, avec l'évaluation pour le garder honnête. " +
			"Près de 80 % des merge requests fusionnées de ma mission en cours sortent de cette équipe d'agents.",
	},
	"en": {
		"observe": "I build multi-agent systems that ship to production. A team of agents that delivers code with " +
			"E2E proofs and mutation testing, MCP servers in series (pgvector persistent memory, 23 trading tools), " +
			"Claude/OpenAI orchestration on Python, Go and Postgres backends. Tool use, evaluation and orchestration " +
			"are architecture building blocks to me, not demos.",
		"long": "I build multi-agent systems that ship to production. A team of agents that delivers code with " +
			"E2E proofs and mutation testing, MCP servers in series (pgvector persistent memory, 23 trading tools), " +
			"Claude/OpenAI orchestration on Python, Go and Postgres backends. Tool use, evaluation and orchestration " +
			"are architecture building blocks to me, not demos. Nearly 80% of the merged merge requests on my " +
			"current engagement come out of that agent team.",
	},
}

// accrochesTaillees : les trois accroches réécrites, longueur observée en prod (jusqu'à 80
// caractères, sur deux lignes de carte).
var accrochesTaillees = map[string][3]string{
	"fr": {
		"Équipe d'agents IA livrant du code sous preuve E2E et test de mutation",
		"Infrastructure de distribution d'API LLM multi-fournisseurs, Docker, Node.js",
		"Agents LLM, serveurs MCP et automatisations en production pour PME et startups",
	},
	"en": {
		"AI agents that ship code, validated by E2E proofs and mutation testing",
		"LLM API distribution infrastructure (Claude), Docker, Node.js",
		"LLM agents, MCP servers and production automations for SMBs and startups",
	},
}

// TestRecettePDF_CheminTailleTientEnDeuxPages rend le CV taillé par le MÊME chemin que la prod
// (RenderDynamicPDF : strip ATS, sélection des projets, htmlToPDF) sur le contenu réel du
// thème ai, habillé d'un résumé et d'accroches de la longueur observée, et exige 2 pages.
//
// Ignoré par défaut (Chromium requis) : il tourne si CV_RECETTE_DATA contient
// cv_ai_{fr,en}.json (réponses de `GET /api/v1/cv?theme=ai&lang=…`). CV_RECETTE_OUT, s'il est
// posé, reçoit les PDF pour inspection.
//
// ⚠️ La pagination dépend des MÉTRIQUES DE POLICE. Hors de l'image Docker, Chromium retombe sur
// Arial/Times (plus étroites) et la page 3 ne se reproduit pas — mesuré sur un hôte Windows le
// 11/11/2026. CV_RECETTE_POLICES (dossier avec DejaVuSans.ttf, DejaVuSans-Bold.ttf,
// CormorantGaramond-SemiBold.ttf) injecte alors les polices de l'image en @font-face : le rendu
// retrouve la pagination de prod. Dans l'image, ne pas le poser.
func TestRecettePDF_CheminTailleTientEnDeuxPages(t *testing.T) {
	dataDir := os.Getenv("CV_RECETTE_DATA")
	if dataDir == "" {
		t.Skip("CV_RECETTE_DATA non posé : recette PDF du CV taillé ignorée")
	}
	outDir := os.Getenv("CV_RECETTE_OUT")
	polices := policesDeLImage(t, os.Getenv("CV_RECETTE_POLICES"))
	gen := &CVGenerationService{l10n: NewLocalizationHelper(), pdfService: newPDFServiceFrom("../../templates/cv")}
	require.True(t, gen.pdfService.hasTemplates)

	for _, lang := range []string{"fr", "en"} {
		for _, variante := range []string{"observe", "long"} {
			t.Run(lang+"-"+variante, func(t *testing.T) {
				raw, err := os.ReadFile(filepath.Join(dataDir, "cv_ai_"+lang+".json"))
				require.NoError(t, err)
				var cv AdaptiveCVResponse
				require.NoError(t, json.Unmarshal(raw, &cv))

				// Habillage « taillé » : thème dynamique, résumé, accroches réécrites.
				cv.Theme = config.CVTheme{ID: "dynamic", Name: "Senior AI Engineer — Multi-agent systems & MCP"}
				cv.Summary = resumesTailles[lang][variante]
				for i, a := range accrochesTaillees[lang] {
					cv.Experiences[i].Catchphrase = a
				}

				// Même composition que RenderDynamicPDF → GenerateTailoredPDFEnPages (strip,
				// sélection plafonnée, ajustement aux pages), polices de l'image injectées en tête.
				stealth := stealthDynamique(&cv, lang)
				pdf, err := ajusterAuxPages(pagesCVTaille, func(plafond int) ([]byte, error) {
					html, err := gen.pdfService.renderCVHTMLPlafond(&cv, lang, stealth, plafond)
					if err != nil {
						return nil, err
					}
					return htmlToPDF(strings.Replace(html, "<head>", "<head>"+polices, 1))
				})
				require.NoError(t, err)
				if outDir != "" {
					require.NoError(t, os.MkdirAll(outDir, 0o755))
					require.NoError(t, os.WriteFile(filepath.Join(outDir, "taille-"+lang+"-"+variante+".pdf"), pdf, 0o644))
				}
				require.Equal(t, 2, compterPagesPDF(pdf), "le CV taillé doit tenir en 2 pages")
			})
		}
	}
}

// policesDeLImage rend un bloc <style> déclarant, en data URI, les polices de l'image Docker
// trouvées dans `dir` (vide → aucun bloc : rendu avec les polices de l'hôte).
func policesDeLImage(t *testing.T, dir string) string {
	t.Helper()
	if dir == "" {
		return ""
	}
	faces := []struct{ fichier, famille, graisse string }{
		{"DejaVuSans.ttf", "DejaVu Sans", "400"},
		{"DejaVuSans-Bold.ttf", "DejaVu Sans", "700"},
		{"CormorantGaramond-SemiBold.ttf", "Cormorant Garamond", "600"},
	}
	var b strings.Builder
	b.WriteString("<style>")
	for _, f := range faces {
		ttf, err := os.ReadFile(filepath.Join(dir, f.fichier))
		require.NoError(t, err)
		fmt.Fprintf(&b, "@font-face{font-family:'%s';font-weight:%s;src:url(data:font/ttf;base64,%s) format('truetype');}",
			f.famille, f.graisse, base64.StdEncoding.EncodeToString(ttf))
	}
	b.WriteString("</style>")
	return b.String()
}

// TestCompterPagesPDF_IgnoreLArbreDesPages : « /Type /Pages » (la racine de l'arbre) ne compte
// pas comme une page ; seuls les « /Type /Page » le font, quelle que soit l'espace qui suit.
func TestCompterPagesPDF_IgnoreLArbreDesPages(t *testing.T) {
	pdf := []byte("<</Type /Pages /Count 3>>\n<</Type /Page\n/Parent 2 0 R>><</Type /Page /Parent 2 0 R>><</Type/Page>>")
	require.Equal(t, 3, compterPagesPDF(pdf))
}

// fauxPDF : un « PDF » de n pages au sens de compterPagesPDF.
func fauxPDF(n int) []byte {
	return []byte("<</Type /Pages>>" + strings.Repeat("<</Type /Page>>", n))
}

// TestAjusterAuxPages_TientDuPremierCoupRenduStandard : un CV qui tient est rendu UNE fois, avec
// la grille standard (compactsMax) — l'ajustement ne change rien à ce qui tenait déjà.
func TestAjusterAuxPages_TientDuPremierCoupRenduStandard(t *testing.T) {
	var plafonds []int
	pdf, err := ajusterAuxPages(2, func(p int) ([]byte, error) {
		plafonds = append(plafonds, p)
		return fauxPDF(2), nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, compterPagesPDF(pdf))
	require.Equal(t, []int{compactsMax}, plafonds)
}

// TestAjusterAuxPages_RetireUneRangeeParDebordement : le cas mesuré en prod — le pied de page
// passe en page 3 ; retirer une rangée de la grille compacte suffit, et on s'arrête là.
func TestAjusterAuxPages_RetireUneRangeeParDebordement(t *testing.T) {
	var plafonds []int
	pdf, err := ajusterAuxPages(2, func(p int) ([]byte, error) {
		plafonds = append(plafonds, p)
		if p > compactsMax-rangeeCompacts {
			return fauxPDF(3), nil
		}
		return fauxPDF(2), nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, compterPagesPDF(pdf))
	require.Equal(t, []int{compactsMax, compactsMax - rangeeCompacts}, plafonds)
}

// TestAjusterAuxPages_SArreteSansGrille : un débordement que la grille ne résout pas (venu de la
// colonne des compétences, par exemple) termine à plafond 0 avec le dernier rendu — pas de
// boucle infinie, pas de CV perdu.
func TestAjusterAuxPages_SArreteSansGrille(t *testing.T) {
	var plafonds []int
	pdf, err := ajusterAuxPages(2, func(p int) ([]byte, error) {
		plafonds = append(plafonds, p)
		return fauxPDF(3), nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, compterPagesPDF(pdf))
	require.Equal(t, 0, plafonds[len(plafonds)-1])
	require.Len(t, plafonds, compactsMax/rangeeCompacts+1)
}

// TestSelectionProjets_PlafondReduitVersLesEcartes : un plafond réduit retire les compacts les
// moins bien notés (ils passent dans `ecartes`, donc vers le strip ATS) ; le plancher ne les
// remet pas ; les cartes détaillées ne bougent pas.
func TestSelectionProjets_PlafondReduitVersLesEcartes(t *testing.T) {
	projets := projetsScores(scoresDegressifs(20, 0.95, 0.45)...)
	dStd, cStd, _ := selectionnerProjets(projets)
	for _, plafond := range []int{2, 0} {
		d, c, e := selectionnerProjetsPlafond(projets, plafond)
		require.Equal(t, dStd, d)
		require.Len(t, c, plafond)
		require.Equal(t, cStd[:plafond], c, "on retire par la FIN (scores les plus bas)")
		require.Equal(t, len(projets), len(d)+len(c)+len(e))
	}
}

// TestRenderPlafond_CompactsRetiresGardentLeursTechnosDansLeStrip : la couche ATS ne perd pas les
// technos des projets retirés par l'ajustement aux pages.
func TestRenderPlafond_CompactsRetiresGardentLeursTechnosDansLeStrip(t *testing.T) {
	svc := newPDFServiceFrom("../../templates/cv")
	require.True(t, svc.hasTemplates)
	projets := projetsScores(scoresDegressifs(10, 0.95, 0.85)...)
	projets[projetsDetailles].Technologies = []string{"technoUniqueDuCompact"}
	cv := &AdaptiveCVResponse{Theme: config.CVTheme{ID: "dynamic", Name: "Tailored CV"}, Projects: projets}

	avec, err := svc.renderCVHTMLPlafond(cv, "en", "Go, Python.", compactsMax)
	require.NoError(t, err)
	sans, err := svc.renderCVHTMLPlafond(cv, "en", "Go, Python.", 0)
	require.NoError(t, err)

	require.Contains(t, avec, `class="mini-proj"`)
	require.NotContains(t, sans, `class="mini-proj"`)
	strip := regexp.MustCompile(`<span class="header-stealth">([^<]*)</span>`).FindStringSubmatch(sans)
	require.Len(t, strip, 2)
	require.Contains(t, strip[1], "technoUniqueDuCompact")
}
