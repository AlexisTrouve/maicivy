package services

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"unicode/utf16"
)

// dossier_print_layout.go — la mise en page IMPRIMÉE du schéma d'architecture d'un dossier.
//
// QUOI : port Go, au nombre près, de frontend/lib/dossier-vivant/pipeline.ts › layoutPipeline et de
// pdfExport.ts › printDiagram, à la largeur d'une page A4 (PRINT_FIT). Entrée : la grille que
// l'agent a produite (nœuds en colonne/rangée, arêtes) ; sortie : la géométrie que le gabarit dessine
// (boîtes, tracés SVG, pastilles, bande du journal).
//
// POURQUOI côté serveur : le PDF public ne doit imprimer que du contenu produit par le serveur. Une
// géométrie reçue du navigateur est un dessin arbitraire — des tracés M/L suffisent à écrire n'importe
// quel mot en traits. Le serveur part donc de SA copie du schéma (celle du run, celle du rejeu) et le
// met en page lui-même.
//
// COMMENT l'égalité avec le front est tenue : mêmes constantes, même ordre d'opérations en flottants
// IEEE (Go et JS calculent les mêmes doubles), longueurs de texte en unités UTF-16 (String.length),
// nombres écrits comme JS les écrit (jsNum). Verrouillé par une golden partagée :
// testdata/dossier_print_diagrams.json, écrite et relue par le test jest du front, relue ici.

// Constantes de pipeline.ts › PIPE (unités du viewBox).
const (
	pipeBoxH       = 40.0
	pipeRowStep    = 36.0
	pipeStackStep  = 52.0
	pipeColGap     = 32.0
	pipeLevelGap   = 28.0
	pipeTitleCh    = 8.2
	pipeSubCh      = 6.4
	pipePadX       = 22.0
	pipeMinW       = 62.0
	pipeJournalGap = 12.0
	pipeJournalH   = 24.0
	pipeStub       = 16.0
	pipeMargin     = 2.0
)

// pdfExport.ts › PRINT_FIT : largeur utile d'une page A4 (210 − 2 × 16 mm à 96 ppp), plancher 0,8.
const (
	printMaxWidth = 672.0
	printMinScale = 0.8
)

// DossierPipelineNode / DossierPipelineEdge — la grille du schéma telle que l'agent l'a écrite (forme
// JSON de events.ts › PipelineNode / PipelineEdge).
type DossierPipelineNode struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Col      int    `json:"col"`
	Row      int    `json:"row"`
	Human    bool   `json:"human,omitempty"`
}

type DossierPipelineEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Dashed bool   `json:"dashed,omitempty"`
	Step   int    `json:"step,omitempty"`
}

type pipeBox struct {
	x, y, w         float64
	title, subtitle string
	human           bool
}

type pipeDraw struct {
	arc  DossierPDFDiagramArc
	step *DossierPDFDiagramDot
}

type pipeGeometry struct {
	width, boxesBottom float64
	boxes              map[string]pipeBox
	draw               func(e DossierPipelineEdge) pipeDraw
}

// jsNum — un nombre tel que JS l'écrit dans un gabarit de chaîne (`${n}`) : plus courte écriture
// exacte, sans exposant dans la plage des coordonnées, et -0 écrit « 0 ».
func jsNum(v float64) string {
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// jsLen — String.length : des unités UTF-16, pas des runes (un emoji compte pour deux).
func jsLen(s string) float64 { return float64(len(utf16.Encode([]rune(s)))) }

func boxWidth(n DossierPipelineNode) float64 {
	return math.Max(pipeMinW, math.Ceil(math.Max(jsLen(n.Title)*pipeTitleCh, jsLen(n.Subtitle)*pipeSubCh)+pipePadX))
}

// tracks — largeur de chaque piste (colonne ou couloir) et abscisses cumulées.
func tracks(nodes []DossierPipelineNode, key func(DossierPipelineNode) int) (w, x []float64) {
	count := 0
	for _, n := range nodes {
		count = max(count, key(n)+1)
	}
	w = make([]float64, count)
	x = make([]float64, count)
	for i := range w {
		w[i] = pipeMinW
		for _, n := range nodes {
			if key(n) == i {
				w[i] = math.Max(w[i], boxWidth(n))
			}
		}
	}
	for i := range x {
		sum := 0.0
		for _, v := range w[:i] {
			sum += v
		}
		x[i] = sum + float64(i)*pipeColGap
	}
	return w, x
}

func maxOf(nodes []DossierPipelineNode, key func(DossierPipelineNode) int) int {
	m := key(nodes[0])
	for _, n := range nodes[1:] {
		m = max(m, key(n))
	}
	return m
}

func colOf(n DossierPipelineNode) int { return n.Col }
func rowOf(n DossierPipelineNode) int { return n.Row }

func stepAt(e DossierPipelineEdge, x, y float64) *DossierPDFDiagramDot {
	if e.Step == 0 {
		return nil
	}
	return &DossierPDFDiagramDot{N: e.Step, X: x, Y: y}
}

func pipeHorizontal(nodes []DossierPipelineNode, edges []DossierPipelineEdge) pipeGeometry {
	colW, colX := tracks(nodes, colOf)
	stacked := false
	for _, a := range nodes {
		for _, b := range nodes {
			if a.Col == b.Col && b.Row == a.Row+1 {
				stacked = true
			}
		}
	}
	rowStep := pipeRowStep
	if stacked {
		rowStep = pipeStackStep
	}
	rows := maxOf(nodes, rowOf) + 1
	node := map[string]DossierPipelineNode{}
	for _, n := range nodes {
		node[n.ID] = n
	}
	sameRowBack := func(e DossierPipelineEdge) bool {
		return node[e.To].Col < node[e.From].Col && node[e.To].Row == node[e.From].Row
	}
	top := 0.0
	for _, e := range edges {
		if sameRowBack(e) {
			top = pipeStub
		}
	}
	boxes := map[string]pipeBox{}
	for _, n := range nodes {
		boxes[n.ID] = pipeBox{x: colX[n.Col], y: top + float64(n.Row)*rowStep, w: colW[n.Col], title: n.Title, subtitle: n.Subtitle, human: n.Human}
	}
	f := jsNum
	draw := func(e DossierPipelineEdge) pipeDraw {
		s, t := boxes[e.From], boxes[e.To]
		sCy, tCy := s.y+pipeBoxH/2, t.y+pipeBoxH/2
		sc, tc := node[e.From].Col, node[e.To].Col
		if tc > sc {
			sx, tx := s.x+s.w, t.x
			sy := sCy
			if tCy != sCy {
				sy = sCy + 6
				if tCy < sCy {
					sy = sCy - 6
				}
			}
			mid := (sx + tx) / 2
			d := fmt.Sprintf("M%s,%sC%s,%s %s,%s %s,%s", f(sx), f(sy), f(mid), f(sy), f(mid), f(tCy), f(tx-6), f(tCy))
			if tCy == sCy {
				d = fmt.Sprintf("M%s,%sH%s", f(sx), f(sy), f(tx-6))
			}
			return pipeDraw{DossierPDFDiagramArc{D: d, Head: fmt.Sprintf("M%s,%sl-6,-3.5v7z", f(tx), f(tCy)), Dashed: e.Dashed}, stepAt(e, mid, (sy+tCy)/2)}
		}
		if tc == sc {
			// Même colonne : verticale de bord à bord, au milieu de la boîte.
			cx := s.x + s.w/2
			down := t.y > s.y
			sy, ty, end, head := s.y, t.y+pipeBoxH, t.y+pipeBoxH+6, "M%s,%sl-3.5,6h7z"
			if down {
				sy, ty, end, head = s.y+pipeBoxH, t.y, t.y-6, "M%s,%sl-3.5,-6h7z"
			}
			return pipeDraw{DossierPDFDiagramArc{D: fmt.Sprintf("M%s,%sV%s", f(cx), f(sy), f(end)), Head: fmt.Sprintf(head, f(cx), f(ty)), Dashed: e.Dashed}, stepAt(e, cx, (sy+ty)/2)}
		}
		sx, tcx := s.x+s.w/2, t.x+t.w/2
		if sameRowBack(e) {
			// Retour sur la même rangée : par le couloir du haut, jamais sur la flèche aller.
			lane := s.y - pipeStub
			return pipeDraw{DossierPDFDiagramArc{D: fmt.Sprintf("M%s,%sV%sH%sV%s", f(sx), f(s.y), f(lane), f(tcx), f(t.y-6)), Head: fmt.Sprintf("M%s,%sl-3.5,-6h7z", f(tcx), f(t.y)), Dashed: e.Dashed}, stepAt(e, (sx+tcx)/2, lane)}
		}
		if t.y < s.y {
			// Retour vers une rangée plus haute : du flanc gauche de la source, puis sur le bas de la cible.
			tb := t.y + pipeBoxH
			return pipeDraw{DossierPDFDiagramArc{D: fmt.Sprintf("M%s,%sH%sV%s", f(s.x), f(sCy), f(tcx), f(tb+6)), Head: fmt.Sprintf("M%s,%sl-3.5,6h7z", f(tcx), f(tb)), Dashed: e.Dashed}, stepAt(e, (s.x+tcx)/2, sCy)}
		}
		// Retour vers une rangée plus basse : sous la source, puis horizontal vers le flanc droit de la cible.
		sy, tx := s.y+pipeBoxH, t.x+t.w
		return pipeDraw{DossierPDFDiagramArc{D: fmt.Sprintf("M%s,%sV%sH%s", f(sx), f(sy), f(tCy), f(tx+6)), Head: fmt.Sprintf("M%s,%sl6,-3.5v7z", f(tx), f(tCy)), Dashed: e.Dashed}, stepAt(e, sx, (sy+tCy)/2)}
	}
	return pipeGeometry{width: colX[len(colX)-1] + colW[len(colW)-1], boxesBottom: top + float64(rows-1)*rowStep + pipeBoxH, boxes: boxes, draw: draw}
}

func pipeVertical(nodes []DossierPipelineNode, edges []DossierPipelineEdge) pipeGeometry {
	// Transposée : la rangée de la grille devient un couloir (abscisse), la colonne un étage.
	laneW, laneX := tracks(nodes, rowOf)
	levelStep := pipeBoxH + pipeLevelGap
	levels := maxOf(nodes, colOf) + 1
	boxes := map[string]pipeBox{}
	col := map[string]int{}
	for _, n := range nodes {
		boxes[n.ID] = pipeBox{x: laneX[n.Row], y: float64(n.Col) * levelStep, w: laneW[n.Row], title: n.Title, subtitle: n.Subtitle, human: n.Human}
		col[n.ID] = n.Col
	}
	lanesRight := laneX[len(laneX)-1] + laneW[len(laneW)-1]
	f := jsNum
	draw := func(e DossierPipelineEdge) pipeDraw {
		s, t := boxes[e.From], boxes[e.To]
		sc, tc := col[e.From], col[e.To]
		if tc > sc {
			// Vers un étage plus bas : du bord bas de la source au bord haut de la cible.
			sx, sy, tx, ty := s.x+s.w/2, s.y+pipeBoxH, t.x+t.w/2, t.y
			mid := (sy + ty) / 2
			d := fmt.Sprintf("M%s,%sC%s,%s %s,%s %s,%s", f(sx), f(sy), f(sx), f(mid), f(tx), f(mid), f(tx), f(ty-6))
			if sx == tx {
				d = fmt.Sprintf("M%s,%sV%s", f(sx), f(sy), f(ty-6))
			}
			return pipeDraw{DossierPDFDiagramArc{D: d, Head: fmt.Sprintf("M%s,%sl-3.5,-6h7z", f(tx), f(ty)), Dashed: e.Dashed}, stepAt(e, (sx+tx)/2, mid)}
		}
		sCy, tCy := s.y+pipeBoxH/2, t.y+pipeBoxH/2
		if tc == sc {
			// Même étage : horizontale de flanc à flanc.
			right := t.x > s.x
			sx, tx, end, head := s.x, t.x+t.w, t.x+t.w+6, "M%s,%sl6,-3.5v7z"
			if right {
				sx, tx, end, head = s.x+s.w, t.x, t.x-6, "M%s,%sl-6,-3.5v7z"
			}
			return pipeDraw{DossierPDFDiagramArc{D: fmt.Sprintf("M%s,%sH%s", f(sx), f(sCy), f(end)), Head: fmt.Sprintf(head, f(tx), f(tCy)), Dashed: e.Dashed}, stepAt(e, (sx+tx)/2, sCy)}
		}
		// Retour vers un étage supérieur : par le couloir libre à droite des deux boîtes.
		cx := math.Max(s.x+s.w, t.x+t.w) + pipeStub
		tx := t.x + t.w
		return pipeDraw{DossierPDFDiagramArc{D: fmt.Sprintf("M%s,%sH%sV%sH%s", f(s.x+s.w), f(sCy), f(cx), f(tCy), f(tx+6)), Head: fmt.Sprintf("M%s,%sl6,-3.5v7z", f(tx), f(tCy)), Dashed: e.Dashed}, stepAt(e, cx, (sCy+tCy)/2)}
	}
	// Un retour déborde à droite du dernier couloir : la largeur lui fait sa place.
	width := lanesRight
	for _, e := range edges {
		if col[e.To] < col[e.From] {
			width = lanesRight + pipeStub + 2
			break
		}
	}
	return pipeGeometry{width: width, boxesBottom: float64(levels-1)*levelStep + pipeBoxH, boxes: boxes, draw: draw}
}

// assemblePrint — l'ordre de dessin de pipeline.ts › assemble (colonne par colonne, chaque boîte avec
// les arêtes qui y arrivent depuis une boîte déjà dessinée, puis les retours), aplati comme le fait
// printDiagram : les arêtes dans cet ordre, les pastilles dans le même.
func assemblePrint(g pipeGeometry, nodes []DossierPipelineNode, edges []DossierPipelineEdge, scale float64, journal, label string) *DossierPDFDiagram {
	order := append([]DossierPipelineNode(nil), nodes...)
	sort.SliceStable(order, func(i, j int) bool {
		if order[i].Col != order[j].Col {
			return order[i].Col < order[j].Col
		}
		return order[i].Row < order[j].Row
	})
	drawn := map[string]bool{}
	used := make([]bool, len(edges))
	var drawnEdges []pipeDraw
	for _, n := range order {
		for i, e := range edges {
			if e.To == n.ID && drawn[e.From] {
				used[i] = true
				drawnEdges = append(drawnEdges, g.draw(e))
			}
		}
		drawn[n.ID] = true
	}
	for i, e := range edges {
		if !used[i] {
			drawnEdges = append(drawnEdges, g.draw(e))
		}
	}
	journalY := g.boxesBottom + pipeJournalGap
	d := &DossierPDFDiagram{
		Label: label, Journal: journal, Width: g.width, Height: journalY + pipeJournalH, Margin: pipeMargin,
		Scale: scale, JournalY: journalY, JournalH: pipeJournalH, BoxH: pipeBoxH,
		Boxes: []DossierPDFDiagramBox{}, Edges: []DossierPDFDiagramArc{}, Steps: []DossierPDFDiagramDot{},
	}
	// Les boîtes dans l'ordre d'insertion d'une Map JS : première apparition de l'id, dernière valeur.
	seen := map[string]bool{}
	for _, n := range nodes {
		if seen[n.ID] {
			continue
		}
		seen[n.ID] = true
		b := g.boxes[n.ID]
		d.Boxes = append(d.Boxes, DossierPDFDiagramBox{X: b.x, Y: b.y, W: b.w, Title: b.title, Subtitle: b.subtitle, Human: b.human})
	}
	for _, e := range drawnEdges {
		d.Edges = append(d.Edges, e.arc)
		if e.step != nil {
			d.Steps = append(d.Steps, *e.step)
		}
	}
	return d
}

func fitScale(width float64) float64 { return math.Min(1, printMaxWidth/(width+2*pipeMargin)) }

// printDiagram — le schéma à imprimer : horizontal tant qu'il tient à l'échelle plancher, sinon la
// transposée verticale si elle s'affiche plus grand (pipeline.ts › layoutPipeline avec PRINT_FIT).
// Une grille sans nœud n'a rien à dessiner : nil (le bloc n'est pas imprimé).
func printDiagram(nodes []DossierPipelineNode, edges []DossierPipelineEdge, journal, label string) *DossierPDFDiagram {
	if len(nodes) == 0 {
		return nil
	}
	h := pipeHorizontal(nodes, edges)
	hs := fitScale(h.width)
	if hs >= printMinScale {
		return assemblePrint(h, nodes, edges, hs, journal, label)
	}
	v := pipeVertical(nodes, edges)
	if vs := fitScale(v.width); vs > hs {
		return assemblePrint(v, nodes, edges, vs, journal, label)
	}
	return assemblePrint(h, nodes, edges, hs, journal, label)
}
