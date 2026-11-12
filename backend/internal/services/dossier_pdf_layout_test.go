package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : la garantie de mise en page du PDF du dossier, MESURÉE sur de vrais dossiers :
//   - le rejeu Tessera et le run réel du 2026-11 (Cabinet Lumen), tels que le navigateur les a
//     envoyés (testdata/dossier_pdf/*.request.json, requêtes capturées sur la page), avec les
//     données réelles du moment (facts-fr.json : montagne, fiches, profil) ;
//   - le même rejeu Tessera en chinois (tessera-zh.request.json) : libellés de zh.json, contenu
//     traduit à la main — un run réel en zh écrit son dossier en chinois, et c'est la langue dont
//     les glyphes manquent le plus vite (polices de l'image de prod : WenQuanYi, Noto CJK) ;
//   - imprimés par le Chromium du CV (htmlToPDF), puis relus par pdftotext -bbox : nombre de pages
//     et position du DERNIER mot de la dernière page. Règle : une seule page, ou une dernière page
//     remplie à FILL (60 %) au moins de sa hauteur utile ;
//   - et relus en texte : chaque texte de la requête (titres, puces, raisons des preuves) se
//     retrouve dans le PDF, et aucune police n'y est en Type3. C'est ce qui attrape un contenu
//     rogné par une page `overflow:hidden` ou un glyphe absent des polices (rendu en .notdef, sans
//     texte extractible) ;
//   - et un balayage : le même dossier, raccourci puis allongé puce par puce, mesuré dans le DOM
//     (`data-layout`, écrit par le script de pagination) — « quel que soit le contenu », pas
//     seulement pour deux exemples.
// Les requêtes ont été capturées sur la page (capture.mjs du banc). Une seule retouche, le chrono du
// run réel : rejoué d'un bloc, son flux faisait lire « 0:00 » au chronomètre ; la ligne porte la durée
// du run d'origine (1:14), comme son PDF d'alors.
// POURQUOI des fixtures et non le réseau : un test ne dépend pas de la production ; l'instantané dit
// sa date. Sans Chromium ou sans pdftotext (CI sans navigateur), le test est sauté — sauf si
// DOSSIER_PDF_REQUIRE_CHROME est posé (banc local, où il doit passer).
// DOSSIER_PDF_OUT=<dossier> : y écrit les PDF rendus (preuves à regarder). PDFTOTEXT : chemin du
// pdftotext de Poppler si celui du PATH n'en est pas un.
// ENVIRONNEMENT : la mesure n'a de valeur qu'avec les polices de la prod (étage runtime de
// backend/Dockerfile : DejaVu, Noto, Noto CJK, WenQuanYi). Mesurée le 2026-11-05 dans une réplique
// de cet étage (mêmes images de base, mêmes paquets, même headless-shell 123) : tessera 74 %,
// cabinet-lumen 77 %, tessera-zh 75 % (2 pages chacun, chinois en WenQuanYi TrueType, tout le texte
// relu) — aucune correction de gabarit nécessaire.

const (
	layoutFill = 0.6
	// Géométrie du gabarit : page A4 (841,89 pt), marge haute 16 mm, zone utile 296,5 − 32 mm.
	pdfTopPt     = 16 * 72 / 25.4
	pdfContentPt = (296.5 - 32) * 72 / 25.4
)

// fileFacts — les faits de l'instantané testdata/dossier_pdf/facts-fr.json.
type fileFacts struct {
	Nodes    []MontagneNoeud `json:"nodes"`
	Projects []MPFProject    `json:"projects"`
	Profile  []MPFSkill      `json:"skills"`
}

func (f fileFacts) Montagne(context.Context, string) (*Montagne, error) {
	return &Montagne{Nodes: f.Nodes}, nil
}
func (f fileFacts) ListProjects(context.Context, string) ([]MPFProject, error) {
	return f.Projects, nil
}
func (f fileFacts) Skills(context.Context, string) ([]MPFSkill, error) { return f.Profile, nil }

func loadJSON(t *testing.T, name string, out any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "dossier_pdf", name))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, out))
}

// realDossiers — les deux dossiers réels et la route qui les imprime (rejeu public ou owner).
// `cjk` : le dossier n'est mesurable qu'avec la police chinoise de la prod (voir hasProdCJKFont).
var realDossiers = []struct {
	name   string
	file   string
	replay bool
	cjk    bool
}{
	{"tessera", "tessera.request.json", true, false},
	{"cabinet-lumen", "cabinet-lumen.request.json", false, false},
	{"tessera-zh", "tessera-zh.request.json", true, true},
}

// hasProdCJKFont — la police chinoise de l'image de prod (WenQuanYi Micro Hei) est-elle installée ?
// POURQUOI : ailleurs, Chromium prend une autre police CJK (sur Windows, les Noto SC variables →
// Type3, chinois non extractible) : le résultat ne dirait rien de la prod. On saute alors le cas zh
// — même sous DOSSIER_PDF_REQUIRE_CHROME, car c'est l'environnement qui manque, pas l'outil.
func hasProdCJKFont() bool {
	out, err := exec.Command("fc-list", ":family=WenQuanYi Micro Hei").Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

// skipWithoutTools — saute (ou échoue, sur le banc) quand le moteur ou pdftotext manquent.
func skipWithoutTools(t *testing.T, what string, err error) {
	t.Helper()
	if os.Getenv("DOSSIER_PDF_REQUIRE_CHROME") != "" {
		require.NoError(t, err, what)
	}
	t.Skipf("%s indisponible sur cette machine : %v", what, err)
}

var (
	pdfPageRe = regexp.MustCompile(`<page width="[\d.]+" height="([\d.]+)">`)
	pdfWordRe = regexp.MustCompile(`<word xMin="[\d.]+" yMin="[\d.]+" xMax="[\d.]+" yMax="([\d.]+)">`)
)

// pdfLastPage — pages du PDF et bas du dernier mot de la dernière page (pt depuis le haut).
func pdfLastPage(t *testing.T, pdf []byte) (pages int, lastBottom float64) {
	t.Helper()
	// pdftotext ne lit pas l'entrée standard : le PDF passe par un fichier temporaire.
	path := filepath.Join(t.TempDir(), "dossier.pdf")
	require.NoError(t, os.WriteFile(path, pdf, 0o644))
	// -bbox est propre au pdftotext de Poppler (celui de Xpdf ne l'a pas) : PDFTOTEXT désigne le bon.
	bin := os.Getenv("PDFTOTEXT")
	if bin == "" {
		bin = "pdftotext"
	}
	out, err := exec.Command(bin, "-bbox", path, "-").Output()
	if err != nil {
		skipWithoutTools(t, "pdftotext", err)
	}
	chunks := pdfPageRe.Split(string(out), -1)[1:]
	for _, m := range pdfWordRe.FindAllStringSubmatch(chunks[len(chunks)-1], -1) {
		y, _ := strconv.ParseFloat(m[1], 64)
		lastBottom = max(lastBottom, y)
	}
	return len(chunks), lastBottom
}

// pdfPlainText — le texte du PDF tel qu'un lecteur le copie (pdftotext sans mise en page).
func pdfPlainText(t *testing.T, pdf []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dossier.pdf")
	require.NoError(t, os.WriteFile(path, pdf, 0o644))
	bin := os.Getenv("PDFTOTEXT")
	if bin == "" {
		bin = "pdftotext"
	}
	out, err := exec.Command(bin, "-enc", "UTF-8", path, "-").Output()
	if err != nil {
		skipWithoutTools(t, "pdftotext", err)
	}
	return string(out)
}

// squeeze — retire tous les blancs : un retour à la ligne de mise en page ne doit pas faire
// échouer la recherche d'une phrase (le chinois n'a pas d'espaces, le français si).
func squeeze(s string) string {
	return strings.Join(strings.Fields(s), "")
}

// requestTexts — les textes de la requête qui doivent tous figurer dans le PDF.
func requestTexts(req *DossierPDFRequest) []string {
	out := append([]string{req.Title}, req.Subtitle...)
	for _, sec := range req.Sections {
		if len(sec.Blocks) > 0 {
			out = append(out, sec.Title)
		}
		for _, b := range sec.Blocks {
			for _, s := range []string{b.Text, b.Why} {
				if s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func TestDossierPDF_DernierePageRemplieMesureeDansLePDF(t *testing.T) {
	var facts fileFacts
	loadJSON(t, "facts-fr.json", &facts)
	svc := NewDossierPDFService(facts)
	for _, d := range realDossiers {
		t.Run(d.name, func(t *testing.T) {
			if d.cjk && !hasProdCJKFont() {
				t.Skip("WenQuanYi Micro Hei absente : polices CJK différentes de la prod")
			}
			var req DossierPDFRequest
			loadJSON(t, d.file, &req)
			html, err := svc.RenderHTML(context.Background(), &req, d.replay)
			require.NoError(t, err)
			require.Contains(t, html, `data-section="skills"`, "la section des compétences est dans le dossier")

			pdf, err := htmlToPDF(html)
			if err != nil {
				skipWithoutTools(t, "Chromium", err)
			}
			if dir := os.Getenv("DOSSIER_PDF_OUT"); dir != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "dossier-"+d.name+".pdf"), pdf, 0o644))
			}
			pages, bottom := pdfLastPage(t, pdf)
			fill := (bottom - pdfTopPt) / pdfContentPt
			t.Logf("%s : %d page(s), dernière page remplie à %.0f %% (dernier mot à %.0f pt)", d.name, pages, fill*100, bottom)
			assert.True(t, pages == 1 || fill >= layoutFill, "%d pages, dernière remplie à %.0f %%", pages, fill*100)

			// Rien de rogné, aucun glyphe perdu : tout le texte demandé se relit dans le PDF.
			text := squeeze(pdfPlainText(t, pdf))
			for _, want := range requestTexts(&req) {
				assert.Contains(t, text, squeeze(want), "texte absent du PDF imprimé")
			}
			// Type3 = police CFF (Noto CJK) convertie par Skia : texte non extractible
			// (cf. TestPoliceCJK_TrueTypeAvantNoto).
			assert.NotContains(t, string(pdf), "/Type3", "police Type3 dans le PDF")
		})
	}
}

// layoutResult — ce que le script de pagination écrit dans `data-layout`.
type layoutResult struct {
	Pages   int     `json:"pages"`
	Fill    float64 `json:"fill"`
	Level   string  `json:"level"`
	Reserve int     `json:"reserve"`
}

// measureLayouts — charge chaque document dans UN Chromium (même injection que htmlToPDF) et lit
// la mise en page décidée par le script, avant toute impression.
func measureLayouts(t *testing.T, docs []string) []layoutResult {
	t.Helper()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.DisableGPU, chromedp.Headless)
	if p := os.Getenv("CHROME_PATH"); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	}
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	ctx, cancelT := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelT()
	var out []layoutResult
	for _, html := range docs {
		var raw string
		load := fmt.Sprintf(`(() => { const b = Uint8Array.from(atob('%s'), c => c.charCodeAt(0));
			document.open('text/html', 'replace'); document.write(new TextDecoder('utf-8').decode(b)); document.close(); })()`,
			base64.StdEncoding.EncodeToString([]byte(html)))
		read := `new Promise((ok) => { const tick = () => { const v = document.documentElement.getAttribute('data-layout'); v ? ok(v) : setTimeout(tick, 20); }; tick(); })`
		err := chromedp.Run(ctx, chromedp.Navigate("about:blank"), chromedp.Evaluate(load, nil),
			chromedp.Evaluate(read, &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }))
		if err != nil {
			skipWithoutTools(t, "Chromium", err)
		}
		var r layoutResult
		require.NoError(t, json.Unmarshal([]byte(raw), &r))
		out = append(out, r)
	}
	return out
}

func TestDossierPDF_BalayageDuContenuMesureDansLeDOM(t *testing.T) {
	var facts fileFacts
	loadJSON(t, "facts-fr.json", &facts)
	svc := NewDossierPDFService(facts)
	// Du dossier réel amputé de ses puces au même dossier allongé de 16 puces de contribution :
	// une plage qui traverse deux sauts de page.
	var docs []string
	var labels []string
	for extra := -6; extra <= 16; extra++ {
		var req DossierPDFRequest
		loadJSON(t, "cabinet-lumen.request.json", &req)
		toDrop := -extra // variantes négatives : les premières puces du dossier retirées, une par une
		for i := range req.Sections {
			sec := &req.Sections[i]
			var kept []DossierPDFBlock
			for _, b := range sec.Blocks {
				if b.Type == "item" && toDrop > 0 {
					toDrop--
					continue
				}
				kept = append(kept, b)
			}
			sec.Blocks = kept
			if extra > 0 && sec.ID == "contribution" {
				for k := 0; k < extra; k++ {
					sec.Blocks = append(sec.Blocks, DossierPDFBlock{Type: "item", Text: fmt.Sprintf("Point de contribution supplémentaire n° %d, rédigé sur une ligne et demie pour peser comme une vraie puce du Rédacteur.", k+1)})
				}
			}
		}
		html, err := svc.RenderHTML(context.Background(), &req, false)
		require.NoError(t, err)
		docs = append(docs, html)
		labels = append(labels, fmt.Sprintf("%+d", extra))
	}
	results := measureLayouts(t, docs)
	pages := map[int]bool{}
	for i, r := range results {
		pages[r.Pages] = true
		t.Logf("variante %s : %d page(s), remplissage %.0f %%, densité %s, réserve %d", labels[i], r.Pages, r.Fill*100, r.Level, r.Reserve)
		assert.True(t, r.Pages == 1 || r.Fill >= layoutFill, "variante %s : %d pages, dernière remplie à %.0f %%", labels[i], r.Pages, r.Fill*100)
	}
	assert.GreaterOrEqual(t, len(pages), 2, "le balayage traverse au moins un saut de page")
}
