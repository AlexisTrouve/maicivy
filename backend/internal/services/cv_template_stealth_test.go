package services

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// cvTemplateSource lit le template CV depuis la racine du dépôt.
// Les tests tournent avec le cwd = internal/services, alors que NewPDFService fait un
// ParseGlob relatif au cwd du binaire (backend/) — d'où le chemin remontant explicite.
func cvTemplateSource(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../../templates/cv/cv_base.html")
	require.NoError(t, err, "template CV introuvable")
	return string(raw)
}

// cssRule extrait le corps d'une règle CSS `selector { ... }` du template.
// Le sélecteur est ancré en début de ligne : sans ça, `.stealth` attraperait `.header-stealth`
// (et `.header` un commentaire qui le citerait) — le test lirait alors la mauvaise règle.
func cssRule(t *testing.T, source, selector string) string {
	t.Helper()
	re := regexp.MustCompile(`(?ms)^` + regexp.QuoteMeta(selector) + `\s*\{(.*?)\}`)
	m := re.FindStringSubmatch(source)
	require.NotNil(t, m, "règle CSS %q absente du template", selector)
	return m[1]
}

// cssHex lit la valeur hexadécimale (#rrggbb, en minuscules) d'une propriété dans un corps de règle.
// `(?m)^\s*` : `color` ne doit pas attraper `background-color` ni `border-left-color`.
func cssHex(t *testing.T, regle, propriete string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(propriete) + `:\s*(#[0-9a-fA-F]{6})\b`).FindStringSubmatch(regle)
	require.NotNil(t, m, "propriété %s (#rrggbb) absente de la règle", propriete)
	return strings.ToLower(m[1])
}

// cssPx lit une longueur en px (entier) d'une propriété dans un corps de règle.
func cssPx(t *testing.T, regle, propriete string) int {
	t.Helper()
	m := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(propriete) + `:\s*(\d+)px\s*;`).FindStringSubmatch(regle)
	require.NotNil(t, m, "propriété %s (en px) absente de la règle", propriete)
	n, err := strconv.Atoi(m[1])
	require.NoError(t, err)
	return n
}

// fondDeLaPage : fond du document, celui sur lequel reposent .main et toute couche qui
// s'échapperait d'une carte. Lu sur `body` ET exigé identique sur `html` : c'est la racine qui
// peint la page imprimée entière (page 2 comprise) ; deux valeurs différentes feraient mentir
// toute mesure « à une unité du fond ».
func fondDeLaPage(t *testing.T, src string) string {
	t.Helper()
	fond := cssHex(t, cssRule(t, src, "body"), "background")
	require.Equal(t, fond, cssHex(t, cssRule(t, src, "html"), "background"),
		"html et body doivent peindre le même fond de page")
	return fond
}

// fmtSscanf lit un octet hexadécimal (deux caractères).
func fmtSscanf(hex string, v *int) (int, error) { return fmt.Sscanf(hex, "%x", v) }

// ecartMaxCanal : plus grand écart par canal entre deux couleurs #rrggbb.
func ecartMaxCanal(a, b string) int {
	pire := 0
	for i := 1; i < 7; i += 2 {
		var x, y int
		_, _ = fmtSscanf(a[i:i+2], &x)
		_, _ = fmtSscanf(b[i:i+2], &y)
		d := x - y
		if d < 0 {
			d = -d
		}
		if d > pire {
			pire = d
		}
	}
	return pire
}

// requireCacheSurSonFond : l'invariant de tout texte caché du CV, quelle que soit la palette.
// Couleur ≠ fond (Chromium retire du flux PDF un texte de la couleur EXACTE du fond — règle payée
// sur `color:#ffffff`) et à au plus 2/255 par canal (au-delà, il se voit).
func requireCacheSurSonFond(t *testing.T, couche, couleur, fond string) {
	t.Helper()
	require.NotEqual(t, fond, couleur, "%s : couleur = fond %s → texte retiré du flux PDF", couche, fond)
	require.LessOrEqual(t, ecartMaxCanal(couleur, fond), 2, "%s : %s trop loin du fond %s → visible", couche, couleur, fond)
}

// TestHeaderStealth_EstBorneEtClippe verrouille le bug mesuré le 2026-09-07 : le strip ATS
// du header était `position:absolute` SANS offsets, dans un `.header` non positionné. Son
// bloc conteneur devenait donc l'ICB, sa position statique l'angle haut-gauche du content-box
// — exactement la photo de profil — et, étant positionné, il se peignait AU-DESSUS de l'<img>.
// Résultat : du texte sombre parfaitement lisible sur la photo, sur les PDF de prod
// (export standard ET /cv/generate).
//
// L'invariant testé n'est pas "telle couleur" mais "le strip est BORNÉ par le header,
// ROGNÉ, et tient ENTIER dans une bande de padding" : c'est ce qui rend impossible le
// recouvrement de la photo, d'un glyphe clair, ou de la crête.
func TestHeaderStealth_EstBorneEtClippe(t *testing.T) {
	src := cvTemplateSource(t)

	// 1. Le header doit être le bloc conteneur du strip, et rogner ce qui dépasse.
	header := cssRule(t, src, ".header")
	require.Contains(t, header, "position: relative",
		".header doit être positionné, sinon le strip ATS se cale sur l'ICB (donc sur la photo)")
	require.Contains(t, header, "overflow: hidden",
		".header doit rogner : un strip trop long déborderait sur la page")

	// 2. Le strip est épinglé (pas laissé en position statique), borné en hauteur, rogné.
	strip := cssRule(t, src, ".header-stealth")
	require.Contains(t, strip, "position: absolute")
	require.Contains(t, strip, "overflow: hidden", "le strip doit rogner son propre débordement")
	hauteur := cssPx(t, strip, "height")

	// 3. Il tient ENTIER dans la bande de padding du bord où il est épinglé : `align-items:
	//    center` n'y place aucun contenu, c'est la seule zone vraiment vide du bandeau.
	//    Padding du header en notation 3 valeurs (haut, côtés, bas).
	pad := regexp.MustCompile(`(?m)^\s*padding:\s*(\d+)px\s+\d+px\s+(\d+)px\s*;`).FindStringSubmatch(header)
	require.NotNil(t, pad, ".header doit déclarer son padding en `haut côtés bas` (px)")
	padHaut, _ := strconv.Atoi(pad[1])
	padBas, _ := strconv.Atoi(pad[2])

	epingleEnHaut := regexp.MustCompile(`(?m)^\s*top:`).MatchString(strip)
	epingleEnBas := regexp.MustCompile(`(?m)^\s*bottom:`).MatchString(strip)
	require.True(t, epingleEnHaut != epingleEnBas, "le strip doit être épinglé à UN bord du header (top ou bottom)")
	if epingleEnHaut {
		require.LessOrEqual(t, cssPx(t, strip, "top")+hauteur, padHaut,
			"le strip déborde de la bande de padding HAUTE : il recouvrirait la photo ou le nom")
	} else {
		require.LessOrEqual(t, cssPx(t, strip, "bottom")+hauteur, padBas,
			"le strip déborde de la bande de padding BASSE : il recouvrirait la photo ou le contenu")
	}

	// 4. La crête (décor peint dans la bande basse) ne partage jamais la bande du strip : posé
	//    sur un plan de montagne, le strip ne serait plus à une unité de SON fond.
	if strings.Contains(src, `class="ridge"`) {
		ridge := cssRule(t, src, ".ridge")
		require.True(t, epingleEnHaut, "la crête occupe la bande basse : le strip doit être en haut")
		require.Regexp(t, `(?m)^\s*bottom:\s*0`, ridge, "la crête doit rester au bas du bandeau")
		require.LessOrEqual(t, cssPx(t, ridge, "height"), padBas, "la crête doit tenir dans la bande basse")
	}

	// 5. Le markup doit passer par la classe — un style inline rouvrirait le bug en silence.
	require.Contains(t, src, `<span class="header-stealth">{{.StealthText}}</span>`,
		"le strip ATS doit utiliser .header-stealth, pas un style inline")
}

// TestHeaderStealth_CouleurNonIdentiqueAuFond : Chromium retire du flux texte PDF un texte
// dont la couleur est exactement celle du fond (règle déjà payée sur `color:#ffffff`).
// Un strip invisible mais ABSENT du stream ne sert plus à rien — l'invisibilité doit venir
// du contraste quasi nul, pas de l'égalité stricte. Et « quasi nul » se vérifie : à au plus
// 2/255 du fond du bandeau, sinon le strip se lit.
func TestHeaderStealth_CouleurNonIdentiqueAuFond(t *testing.T) {
	src := cvTemplateSource(t)
	fond := cssHex(t, cssRule(t, src, ".header"), "background")
	requireCacheSurSonFond(t, ".header-stealth", cssHex(t, cssRule(t, src, ".header-stealth"), "color"), fond)
}

// TestMiniProj_ContenuCouvreTouteLaCellule verrouille le second bug mesuré le 2026-09-07 :
// dans la grille des projets compacts, les cellules d'une même rangée s'étirent à la hauteur
// de la plus haute. Le contenu opaque, lui, gardait sa hauteur naturelle — laissant du
// vide sous lui. Or le .stealth couvre la carte entière : ses lignes du bas s'affichaient
// dans ce vide, coupées au milieu d'une phrase (PDF de prod, page 4, carte ProRequestManager).
//
// L'invariant : le contenu OPAQUE doit REMPLIR la cellule, sinon le stealth réapparaît.
func TestMiniProj_ContenuCouvreTouteLaCellule(t *testing.T) {
	src := cvTemplateSource(t)

	carte := cssRule(t, src, ".mini-proj")
	require.Contains(t, carte, "position: relative", "ancre du stealth absolu")
	require.Contains(t, carte, "overflow: hidden", "le stealth ne doit pas sortir de la carte")
	require.Contains(t, carte, "display: flex",
		".mini-proj doit être flex, sinon son contenu ne s'étire pas à la hauteur de la rangée")

	contenu := cssRule(t, src, ".mini-proj-content")
	// #rrggbb (6 chiffres) : ni canal alpha, ni rgba(), ni transparent — un fond translucide
	// laisserait voir la couche dessous.
	cssHex(t, contenu, "background")
	require.Regexp(t, `flex:\s*1`, contenu,
		"le contenu doit remplir la cellule étirée, sinon le stealth réapparaît dans le vide du bas")
}

// TestStealth_RetraitDuBordDeCarte verrouille le défaut mesuré le 2026-11-09 sur le rendu PDF
// (maquette sombre ET ancien gabarit clair) : avec `inset: 0`, la couche .stealth s'étendait
// jusqu'au bord de la carte. Au bas d'une mini-carte de hauteur fractionnaire, le bord
// anticrénelé du contenu opaque laissait transparaître une ligne de pixels de la couche
// (écart 79 à 90/255 sur fond clair) — du texte caché qui se voit au bord.
//
// L'invariant : la couche est RETIRÉE d'au moins 3px de tous les bords du conteneur, et chaque
// conteneur (.exp-wrap, .proj-wrap, .mini-proj) est bien son bloc conteneur et n'a pas de
// padding — sinon le retrait se mesurerait depuis une autre boîte que la carte.
func TestStealth_RetraitDuBordDeCarte(t *testing.T) {
	src := cvTemplateSource(t)

	stealth := cssRule(t, src, ".stealth")
	require.Contains(t, stealth, "position: absolute")
	require.Contains(t, stealth, "overflow: hidden", "le texte rogné reste dans le flux PDF, pas à l'écran")
	// Une seule valeur : les quatre bords. Une notation à 2-4 valeurs pourrait remettre un bord à 0.
	m := regexp.MustCompile(`(?m)^\s*inset:\s*(\d+)px\s*;`).FindStringSubmatch(stealth)
	require.NotNil(t, m, ".stealth doit déclarer `inset: Npx` (même retrait sur les quatre bords)")
	retrait, _ := strconv.Atoi(m[1])
	require.GreaterOrEqual(t, retrait, 3, "la couche touche le bord de la carte : le bord anticrénelé la laisse voir")
	for _, interdit := range []string{"top:", "right:", "bottom:", "left:"} {
		require.NotRegexp(t, `(?m)^\s*`+interdit, stealth, ".stealth : %s écraserait le retrait de inset", interdit)
	}

	for _, conteneur := range []string{".exp-wrap", ".proj-wrap", ".mini-proj"} {
		regle := cssRule(t, src, conteneur)
		require.Contains(t, regle, "position: relative", "%s doit être le bloc conteneur de la couche", conteneur)
		require.NotRegexp(t, `(?m)^\s*padding`, regle, "%s : un padding décalerait la carte par rapport au retrait", conteneur)
	}
}

// TestStealth_CouleurAUneUniteDuFondDeLaPage : sur le fond sombre, la couche .stealth est
// AUSSI à une unité du fond de la page. C'est la seconde protection du design sombre : si une
// carte venait à ne pas la couvrir (bug ProRequestManager, 2026-09-07), elle resterait
// invisible. Dans l'ancien gabarit clair, une couche exposée se lisait en sombre sur blanc.
func TestStealth_CouleurAUneUniteDuFondDeLaPage(t *testing.T) {
	src := cvTemplateSource(t)
	requireCacheSurSonFond(t, ".stealth", cssHex(t, cssRule(t, src, ".stealth"), "color"), fondDeLaPage(t, src))
}

// TestAtsAnchor_CouleurAccordeeAuFondDeSaColonne verrouille le bug mesuré le 2026-10-16 sur le
// PDF de prod : `.ats-anchor` portait UNE couleur, celle du fond de la SIDEBAR. Or l'ancre
// « Professional Experience. Work History. » vit dans `.main`, sur un AUTRE fond : elle
// s'affichait en gris pâle au-dessus du titre de section, sur tous les CV, toutes langues.
// Et l'ancre « Education » de la sidebar, elle, avait EXACTEMENT la couleur de son fond — ce que
// Chromium retire du flux texte.
//
// L'invariant, par colonne : la couleur de l'ancre DIFFÈRE du fond de SA colonne (sinon absente
// du PDF) et en reste à au plus 2/255 par canal (sinon visible). Les fonds sont lus dans le
// gabarit : `.main` n'a pas de fond propre, c'est celui de la page ; la sidebar a le sien.
func TestAtsAnchor_CouleurAccordeeAuFondDeSaColonne(t *testing.T) {
	src := cvTemplateSource(t)

	require.NotRegexp(t, `(?m)^\s*background`, cssRule(t, src, ".main"),
		".main a un fond propre : le lire ici au lieu de supposer celui de la page")
	fonds := map[string]string{
		".main":    fondDeLaPage(t, src),
		".sidebar": cssHex(t, cssRule(t, src, ".sidebar"), "background"),
	}
	require.NotEqual(t, fonds[".main"], fonds[".sidebar"],
		"fonds identiques : le test ne distinguerait plus les colonnes")

	for colonne, fond := range fonds {
		couleur := cssHex(t, cssRule(t, src, colonne+" .ats-anchor"), "color")
		requireCacheSurSonFond(t, colonne+" .ats-anchor", couleur, fond)
	}
}

// TestSectionLabel_JamaisSeulEnBasDePage : un titre de section ne termine jamais une page.
// POURQUOI : mesuré le 2026-10-16 sur les exports de prod, fullstack et backend en fr ET zh :
// « PROJETS » / « 项目 » tombait seul au pied de la page 1, ses cartes en page 2. Reproduit
// dans l'image de prod (headless-shell 123) en décalant la section de 0 à 720 px : 5 positions
// sur 61 laissaient le titre orphelin, 0 avec `break-after: avoid` (titre emmené en page 2).
// COMMENT : on vérifie la règle dans le bloc CSS de .section-label du VRAI gabarit.
func TestSectionLabel_JamaisSeulEnBasDePage(t *testing.T) {
	gabarit, err := os.ReadFile("../../templates/cv/cv_base.html")
	require.NoError(t, err)
	g := string(gabarit)
	i := strings.Index(g, ".section-label {")
	require.GreaterOrEqual(t, i, 0)
	bloc := g[i : i+strings.Index(g[i:], "}")]
	require.Contains(t, bloc, "break-after: avoid")
	require.Contains(t, bloc, "page-break-after: avoid")
}

// TestPoliceTitres_EmbarqueeDansLImage : le gabarit compose ses titres en Cormorant Garamond,
// qui n'a pas de paquet Debian. Sans le TTF installé dans l'image, Chromium retomberait en
// silence sur DejaVu Serif : le PDF de prod ne ressemblerait plus au design validé, et le
// budget du strip / la pagination n'auraient pas été mesurés sur ces métriques.
// L'invariant : toute famille nommée par le gabarit hors paquets Debian est versionnée dans
// backend/fonts/ (avec sa licence) ET copiée + indexée (fc-cache) par le Dockerfile.
func TestPoliceTitres_EmbarqueeDansLImage(t *testing.T) {
	src := cvTemplateSource(t)
	require.Contains(t, src, "'Cormorant Garamond'")

	for _, f := range []string{"../../fonts/CormorantGaramond-SemiBold.ttf", "../../fonts/OFL.txt"} {
		info, err := os.Stat(f)
		require.NoError(t, err, "%s absent du dépôt", f)
		require.Greater(t, info.Size(), int64(0))
	}
	licence, err := os.ReadFile("../../fonts/OFL.txt")
	require.NoError(t, err)
	require.Contains(t, string(licence), "SIL Open Font License")

	dockerfile, err := os.ReadFile("../../Dockerfile")
	require.NoError(t, err)
	d := string(dockerfile)
	copie := regexp.MustCompile(`(?m)^COPY fonts/ /usr/share/fonts/`).FindStringIndex(d)
	require.NotNil(t, copie, "le Dockerfile doit copier backend/fonts/ dans /usr/share/fonts/")
	cache := regexp.MustCompile(`(?m)^RUN fc-cache`).FindStringIndex(d[copie[1]:])
	require.NotNil(t, cache, "fc-cache doit tourner APRÈS la copie, sinon fontconfig ignore la police")
}
