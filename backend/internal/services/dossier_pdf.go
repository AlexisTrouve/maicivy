package services

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// dossier_pdf.go — le PDF du « dossier de conception » du dossier vivant
// (frontend/app/[locale]/dossier-vivant).
//
// QUOI : met en page pour l'impression A4 (dossier_pdf_template.html) un dossier REMIS
// (DossierPDFRequest) et l'imprime avec le Chromium headless du CV (htmlToPDF). Aucun appel IA,
// aucune écriture.
//
// D'OÙ VIENT LE DOSSIER — la règle de sûreté : un PDF à l'en-tête d'Alexis servi par une route
// PUBLIQUE ne contient que du contenu produit par le serveur.
//   - routes publiques : la requête est construite ICI (dossier_record.go) à partir de la copie
//     serveur — le dossier gardé d'un run réel (dossier_store.go) ou celui du rejeu Tessera
//     (dossier_tessera.go). Le navigateur ne fournit qu'un identifiant et une langue. Avant, il
//     envoyait le dossier replié, et n'importe qui pouvait fabriquer avec curl un PDF signé Alexis
//     portant n'importe quel texte ;
//   - route owner : la requête complète arrive dans le corps (outil du propriétaire, RequireOwner).
//
// CE QUE NI LE NAVIGATEUR NI LE RUN NE FOURNISSENT, et que ce service relit lui-même :
//   - statut, lignes et tests des preuves : sur les fiches maiProFiles. Règle du dossier vivant :
//     aucun chiffre ne vient du flux ;
//   - noms et monogrammes des projets : sur la montagne ;
//   - l'adresse de contact : constante du serveur.
//
// SÛRETÉ (la route owner reçoit encore une requête externe) : chaque texte est borné et échappé par
// html/template ; le schéma arrive en géométrie, chaque nombre est borné et chaque tracé SVG
// réduit à une grammaire de commandes et de nombres ; les liens de démo sont http(s) seulement.
// Un seul Chromium à la fois pour ce document (`slots`) : la charge d'un visiteur insistant est
// bornée par construction, pas par un quota.

//go:embed dossier_pdf_template.html
var dossierPDFTemplateSrc string

var dossierPDFTemplate = template.Must(template.New("dossier").Parse(dossierPDFTemplateSrc))

// Identité imprimée sur chaque dossier — la même adresse que le profil (profile_builder.go) et
// le pied de page du site (frontend/lib/montagne/links.ts › CONTACT_EMAIL).
const (
	dossierAuthor = "Alexis Trouvé"
	dossierEmail  = "alexistrouve.pro@gmail.com"
	dossierSite   = "https://maicivy.etheryale.com"
)

// ErrDossierPDFInvalid — la requête ne respecte pas le contrat (400).
var ErrDossierPDFInvalid = errors.New("dossier pdf: requête invalide")

// ErrDossierPDFBusy — un autre dossier s'imprime et n'a pas libéré le moteur à temps (503).
var ErrDossierPDFBusy = errors.New("dossier pdf: moteur occupé")

// dossierPDFWait — attente maximale d'un tour d'impression avant de répondre « occupé ».
const dossierPDFWait = 20 * time.Second

// --- Contrat de la requête (miroir de pdfExport.ts › DossierPdfRequest) ---------------------------

type DossierPDFRequest struct {
	Lang     string              `json:"lang"`
	Title    string              `json:"title"`
	Subtitle []string            `json:"subtitle"`
	Sections []DossierPDFSection `json:"sections"`
	Labels   DossierPDFLabels    `json:"labels"`
}

type DossierPDFSection struct {
	ID     string            `json:"id"`
	Title  string            `json:"title"`
	Blocks []DossierPDFBlock `json:"blocks"`
}

// DossierPDFBlock — union discriminée par Type : lead | paragraph | item | diagram | proof | also | contact.
type DossierPDFBlock struct {
	Type       string             `json:"type"`
	Text       string             `json:"text,omitempty"`
	Diagram    *DossierPDFDiagram `json:"diagram,omitempty"`
	ProjectID  string             `json:"projectId,omitempty"`
	Why        string             `json:"why,omitempty"`
	Demo       *DossierPDFDemo    `json:"demo,omitempty"`
	ProjectIDs []string           `json:"projectIds,omitempty"`
}

type DossierPDFDemo struct {
	Kind string `json:"kind"` // video | site
	URL  string `json:"url"`
}

// DossierPDFDiagram — le schéma, déjà mis en page par le front (pipeline.ts) à la largeur A4.
type DossierPDFDiagram struct {
	Label    string                 `json:"label"`
	Journal  string                 `json:"journal"`
	Width    float64                `json:"width"`
	Height   float64                `json:"height"`
	Margin   float64                `json:"margin"`
	Scale    float64                `json:"scale"`
	JournalY float64                `json:"journalY"`
	JournalH float64                `json:"journalH"`
	BoxH     float64                `json:"boxH"`
	Boxes    []DossierPDFDiagramBox `json:"boxes"`
	Edges    []DossierPDFDiagramArc `json:"edges"`
	Steps    []DossierPDFDiagramDot `json:"steps"`
}

type DossierPDFDiagramBox struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	Title    string  `json:"title"`
	Subtitle string  `json:"subtitle"`
	Human    bool    `json:"human"`
}

type DossierPDFDiagramArc struct {
	D      string `json:"d"`
	Head   string `json:"head"`
	Dashed bool   `json:"dashed"`
}

type DossierPDFDiagramDot struct {
	N int     `json:"n"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DossierPDFLabels — libellés d'INTERFACE, résolus par next-intl dans la langue du visiteur.
// `Also` garde le paramètre `{names}`, rempli ici avec les noms lus sur la montagne.
type DossierPDFLabels struct {
	Project    string            `json:"project"`
	Status     string            `json:"status"`
	Lines      string            `json:"lines"`
	Tests      string            `json:"tests"`
	Demo       string            `json:"demo"`
	Video      string            `json:"video"`
	Site       string            `json:"site"`
	None       string            `json:"none"`
	Also       string            `json:"also"`
	Contact    string            `json:"contact"`
	ReplayNote string            `json:"replayNote"`
	Statuses   map[string]string `json:"statuses"`
	// Skills — la section « Compétences mobilisées » (dossier_skills.go). Titre vide = section non
	// imprimée : jamais un en-tête sans ses libellés.
	Skills DossierPDFSkillLabels `json:"skills"`
}

// DossierPDFSkillLabels — libellés de la section des compétences. `Levels` : clé de niveau du
// profil (expert, advanced…) → libellé affiché.
type DossierPDFSkillLabels struct {
	Title        string            `json:"title"`
	Lead         string            `json:"lead"`
	Skill        string            `json:"skill"`
	Level        string            `json:"level"`
	Source       string            `json:"source"`
	Need         string            `json:"need"`
	Technologies string            `json:"technologies"`
	Levels       map[string]string `json:"levels"`
}

// --- Validation (frontière système) ---------------------------------------------------------------

// Bornes : larges devant un vrai dossier (le run réel du 2026-10 : 5 sections, 21 blocs, 6 boîtes),
// étroites devant un abus. Un texte se compte en caractères, pas en octets (chinois).
const (
	dossierMaxLabel     = 300
	dossierMaxText      = 2000
	dossierMaxTotalText = 40000
	dossierMaxBlocks    = 60
	dossierMaxBoxes     = 60
	dossierMaxEdges     = 120
	dossierMaxPath      = 4000
	dossierMaxCoord     = 10000
	dossierMaxIDs       = 30
)

var (
	dossierSectionIDs = map[string]bool{"need": true, "architecture": true, "proofs": true, "contribution": true, "next": true}
	dossierLangs      = map[string]bool{"fr": true, "en": true, "de": true, "it": true, "zh": true}
	// Grammaire des tracés produits par pipeline.ts : commandes M L H V C Z (et minuscules), nombres.
	dossierPathRe  = regexp.MustCompile(`^[MmLlHhVvCcZz0-9.,\- ]+$`)
	dossierIDRe    = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,100}$`)
	dossierLabelRe = regexp.MustCompile(`^[^<>]*$`)
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrDossierPDFInvalid, fmt.Sprintf(format, args...))
}

// textBudget — compte les caractères d'une requête (pas les octets : chinois) et borne chaque texte.
type textBudget struct{ total int }

func (b *textBudget) text(field, s string, max int) error {
	n := utf8.RuneCountInString(s)
	if n > max {
		return invalid("%s : %d caractères (max %d)", field, n, max)
	}
	b.total += n
	return nil
}

// validateDossierPDF — tout ce qui sort de la requête est vérifié ici, une fois, avant le rendu.
func validateDossierPDF(r *DossierPDFRequest, replay bool) error {
	b := &textBudget{}
	if strings.TrimSpace(r.Title) == "" {
		return invalid("titre vide")
	}
	if err := b.text("title", r.Title, dossierMaxLabel); err != nil {
		return err
	}
	if len(r.Subtitle) > 3 {
		return invalid("sous-titre : %d lignes", len(r.Subtitle))
	}
	for _, l := range r.Subtitle {
		if err := b.text("subtitle", l, dossierMaxLabel); err != nil {
			return err
		}
	}
	if err := validateLabels(&r.Labels, replay); err != nil {
		return err
	}
	return validateSections(b, r.Lang, r.Sections)
}

// validateDossierSections — la langue et les sections seules : ce que lit la route des
// compétences (POST /dossier-vivant/skills), publique elle aussi.
func validateDossierSections(lang string, sections []DossierPDFSection) error {
	return validateSections(&textBudget{}, lang, sections)
}

func validateSections(b *textBudget, lang string, sections []DossierPDFSection) error {
	if !dossierLangs[lang] {
		return invalid("lang %q", lang)
	}
	if len(sections) == 0 || len(sections) > len(dossierSectionIDs) {
		return invalid("%d sections", len(sections))
	}
	seen := map[string]bool{}
	blocks := 0
	for _, sec := range sections {
		if !dossierSectionIDs[sec.ID] || seen[sec.ID] {
			return invalid("section %q", sec.ID)
		}
		seen[sec.ID] = true
		if err := b.text("section.title", sec.Title, dossierMaxLabel); err != nil {
			return err
		}
		blocks += len(sec.Blocks)
		if blocks > dossierMaxBlocks {
			return invalid("plus de %d blocs", dossierMaxBlocks)
		}
		for _, blk := range sec.Blocks {
			if err := validateBlock(&blk, b.text); err != nil {
				return err
			}
		}
	}
	if b.total > dossierMaxTotalText {
		return invalid("%d caractères au total (max %d)", b.total, dossierMaxTotalText)
	}
	return nil
}

func validateLabels(l *DossierPDFLabels, replay bool) error {
	k := l.Skills
	fields := map[string]string{
		"project": l.Project, "status": l.Status, "lines": l.Lines, "tests": l.Tests, "demo": l.Demo,
		"video": l.Video, "site": l.Site, "none": l.None, "also": l.Also, "contact": l.Contact, "replayNote": l.ReplayNote,
		"skills.title": k.Title, "skills.lead": k.Lead, "skills.skill": k.Skill, "skills.level": k.Level,
		"skills.source": k.Source, "skills.need": k.Need, "skills.technologies": k.Technologies,
	}
	for k, v := range fields {
		if utf8.RuneCountInString(v) > dossierMaxLabel {
			return invalid("libellé %s trop long", k)
		}
	}
	if !strings.Contains(l.Also, "{names}") {
		return invalid("libellé also sans {names}")
	}
	// Le rejeu se dit sur le document : la route publique n'imprime pas sans sa mention.
	if replay && strings.TrimSpace(l.ReplayNote) == "" {
		return invalid("mention du rejeu absente")
	}
	for name, m := range map[string]map[string]string{"statut": l.Statuses, "niveau": l.Skills.Levels} {
		if len(m) > 20 {
			return invalid("%d libellés de %s", len(m), name)
		}
		for k, v := range m {
			if utf8.RuneCountInString(k) > 40 || utf8.RuneCountInString(v) > 80 || !dossierLabelRe.MatchString(k) {
				return invalid("%s %q", name, k)
			}
		}
	}
	return nil
}

func validateBlock(b *DossierPDFBlock, text func(field, s string, max int) error) error {
	switch b.Type {
	case "lead", "paragraph", "item":
		return text(b.Type, b.Text, dossierMaxText)
	case "diagram":
		if b.Diagram == nil {
			return invalid("schéma vide")
		}
		return validateDiagram(b.Diagram, text)
	case "proof":
		if !dossierIDRe.MatchString(b.ProjectID) {
			return invalid("projet %q", b.ProjectID)
		}
		if b.Demo != nil {
			if b.Demo.Kind != "video" && b.Demo.Kind != "site" {
				return invalid("démo %q", b.Demo.Kind)
			}
			if len(b.Demo.URL) > 500 || !(strings.HasPrefix(b.Demo.URL, "https://") || strings.HasPrefix(b.Demo.URL, "http://")) {
				return invalid("lien de démo")
			}
		}
		return text("why", b.Why, dossierMaxText)
	case "also":
		if len(b.ProjectIDs) > dossierMaxIDs {
			return invalid("%d projets en « aussi »", len(b.ProjectIDs))
		}
		for _, id := range b.ProjectIDs {
			if !dossierIDRe.MatchString(id) {
				return invalid("projet %q", id)
			}
		}
		return nil
	case "contact":
		return nil
	default:
		return invalid("bloc %q", b.Type)
	}
}

func coordOK(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > dossierMaxCoord {
			return false
		}
	}
	return true
}

func validateDiagram(d *DossierPDFDiagram, text func(field, s string, max int) error) error {
	if d.Width <= 0 || d.Height <= 0 || d.Scale <= 0 || d.Scale > 1 || d.BoxH <= 0 || d.Margin < 0 ||
		!coordOK(d.Width, d.Height, d.Margin, d.JournalY, d.JournalH, d.BoxH) {
		return invalid("dimensions du schéma")
	}
	if len(d.Boxes) == 0 || len(d.Boxes) > dossierMaxBoxes || len(d.Edges) > dossierMaxEdges || len(d.Steps) > dossierMaxEdges {
		return invalid("schéma : %d boîtes, %d arêtes", len(d.Boxes), len(d.Edges))
	}
	if err := text("diagram.label", d.Label, dossierMaxText); err != nil {
		return err
	}
	if err := text("diagram.journal", d.Journal, dossierMaxLabel); err != nil {
		return err
	}
	for _, b := range d.Boxes {
		if b.W <= 0 || !coordOK(b.X, b.Y, b.W) {
			return invalid("boîte du schéma")
		}
		if err := text("box.title", b.Title, dossierMaxLabel); err != nil {
			return err
		}
		if err := text("box.subtitle", b.Subtitle, dossierMaxLabel); err != nil {
			return err
		}
	}
	for _, e := range d.Edges {
		for _, p := range []string{e.D, e.Head} {
			if p == "" || len(p) > dossierMaxPath || !dossierPathRe.MatchString(p) {
				return invalid("tracé du schéma")
			}
		}
	}
	for _, s := range d.Steps {
		if s.N < 0 || s.N > 999 || !coordOK(s.X, s.Y) {
			return invalid("pastille du schéma")
		}
	}
	return nil
}

// --- Faits relus par le serveur -------------------------------------------------------------------

// DossierFactsSource — d'où viennent les noms et les chiffres des projets cités.
type DossierFactsSource interface {
	Montagne(ctx context.Context, lang string) (*Montagne, error)
	ListProjects(ctx context.Context, lang string) ([]MPFProject, error)
	// Skills — le profil (GET /skills) : noms, niveaux. "fr" = noms canoniques, ceux des tags.
	Skills(ctx context.Context, lang string) ([]MPFSkill, error)
}

type liveDossierFacts struct {
	montagne *MontagneService
	profiles *MaiProFilesClient
}

// NewDossierFactsSource — la montagne (cache SWR) et le client maiProFiles (cache mémoire) du backend.
func NewDossierFactsSource(m *MontagneService, p *MaiProFilesClient) DossierFactsSource {
	return liveDossierFacts{montagne: m, profiles: p}
}

func (f liveDossierFacts) Montagne(ctx context.Context, lang string) (*Montagne, error) {
	return f.montagne.Get(ctx, lang)
}

func (f liveDossierFacts) ListProjects(ctx context.Context, lang string) ([]MPFProject, error) {
	return f.profiles.ListProjects(ctx, lang)
}

func (f liveDossierFacts) Skills(ctx context.Context, lang string) ([]MPFSkill, error) {
	return f.profiles.GetSkills(ctx, lang)
}

// --- Service --------------------------------------------------------------------------------------

type DossierPDFService struct {
	facts  DossierFactsSource
	icons  ProjectIconReader
	render func(html string) ([]byte, error)
	slots  chan struct{}
}

// ProjectIconReader — l'icône d'un projet (PNG réduit), si son dépôt en a une
// (services/project_icons.go). Le même service sert l'écran : un projet porte la même marque à
// l'écran et sur papier.
type ProjectIconReader interface {
	Icon(ctx context.Context, id string) (data []byte, version string, ok bool)
}

// WithIcons — branche les icônes des projets (sinon : monogrammes partout). Réglage séparé du
// constructeur : les icônes sont facultatives, le PDF se rend sans elles.
func (s *DossierPDFService) WithIcons(icons ProjectIconReader) *DossierPDFService {
	s.icons = icons
	return s
}

func NewDossierPDFService(facts DossierFactsSource) *DossierPDFService {
	return &DossierPDFService{facts: facts, render: htmlToPDF, slots: make(chan struct{}, 1)}
}

// WithRenderer — remplace l'impression Chromium (htmlToPDF). POURQUOI : les tests des routes
// doivent lire ce qui part à l'impression (le document HTML) sans dépendre d'un navigateur installé
// ni extraire du texte d'un flux PDF compressé.
func (s *DossierPDFService) WithRenderer(render func(html string) ([]byte, error)) *DossierPDFService {
	s.render = render
	return s
}

// ReplayRequest — le document du rejeu Tessera, tel que le serveur le connaît (dossier_tessera.go),
// dans la langue `lang` de l'interface. Rien ne vient du navigateur, à part ce choix de langue.
func (s *DossierPDFService) ReplayRequest(ctx context.Context, lang string) (*DossierPDFRequest, error) {
	if !dossierLangs[lang] {
		return nil, invalid("lang %q", lang)
	}
	m, err := s.facts.Montagne(ctx, lang)
	if err != nil {
		return nil, fmt.Errorf("dossier pdf: montagne : %w", err)
	}
	return tesseraRecord(lang, m).PDFRequest(lang)
}

// RenderReplay — le PDF du rejeu (route publique), avec sa mention.
func (s *DossierPDFService) RenderReplay(ctx context.Context, lang string) ([]byte, error) {
	req, err := s.ReplayRequest(ctx, lang)
	if err != nil {
		return nil, err
	}
	return s.Render(ctx, req, true)
}

// RenderRecord — le PDF d'un run réel gardé par le serveur (route publique du run), sans mention.
func (s *DossierPDFService) RenderRecord(ctx context.Context, rec *DossierRecord, lang string) ([]byte, error) {
	req, err := rec.PDFRequest(lang)
	if err != nil {
		return nil, err
	}
	return s.Render(ctx, req, false)
}

// Render — valide, met en page, imprime une requête complète. `replay` imprime la mention du rejeu.
// Appelé tel quel par la seule route OWNER (corps = dossier) ; les routes publiques passent par
// RenderReplay / RenderRecord, qui construisent la requête côté serveur.
func (s *DossierPDFService) Render(ctx context.Context, req *DossierPDFRequest, replay bool) ([]byte, error) {
	html, err := s.RenderHTML(ctx, req, replay)
	if err != nil {
		return nil, err
	}
	wait, cancel := context.WithTimeout(ctx, dossierPDFWait)
	defer cancel()
	select {
	case s.slots <- struct{}{}:
	case <-wait.Done():
		return nil, ErrDossierPDFBusy
	}
	defer func() { <-s.slots }()
	return s.render(html)
}

// RenderHTML — le document A4, prêt à imprimer. Séparé de Render : testable sans navigateur.
func (s *DossierPDFService) RenderHTML(ctx context.Context, req *DossierPDFRequest, replay bool) (string, error) {
	if err := validateDossierPDF(req, replay); err != nil {
		return "", err
	}
	montagne, err := s.facts.Montagne(ctx, req.Lang)
	if err != nil {
		return "", fmt.Errorf("dossier pdf: montagne : %w", err)
	}
	projects, err := s.facts.ListProjects(ctx, req.Lang)
	if err != nil {
		return "", fmt.Errorf("dossier pdf: fiches : %w", err)
	}
	var skills DossierSkills
	if req.Labels.Skills.Title != "" {
		if skills, err = s.skillsFor(ctx, req.Lang, req.Sections, projects); err != nil {
			return "", err
		}
	}
	view := buildDossierView(req, replay, montagne, projects, skills, s.proofIcons(ctx, req))
	var buf bytes.Buffer
	if err := dossierPDFTemplate.Execute(&buf, view); err != nil {
		return "", fmt.Errorf("dossier pdf: gabarit : %w", err)
	}
	return buf.String(), nil
}

// proofIcons — les icônes des projets cités en preuve, en data: URI. Embarquées dans le document :
// le Chromium qui imprime n'a ni réseau vers l'API ni jeton Gitea, et n'en a pas besoin.
func (s *DossierPDFService) proofIcons(ctx context.Context, req *DossierPDFRequest) map[string]template.URL {
	out := map[string]template.URL{}
	if s.icons == nil {
		return out
	}
	for _, sec := range req.Sections {
		for _, b := range sec.Blocks {
			if b.Type != "proof" || out[b.ProjectID] != "" {
				continue
			}
			if data, _, ok := s.icons.Icon(ctx, b.ProjectID); ok {
				// Sûr : des octets PNG que le backend vient lui-même d'encoder, en base64.
				out[b.ProjectID] = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(data))
			}
		}
	}
	return out
}

// --- Vue du gabarit -------------------------------------------------------------------------------

type dossierView struct {
	Lang       string
	Title      string
	Subtitle   []string
	ReplayNote string // vide hors rejeu
	Labels     DossierPDFLabels
	Sections   []dossierSectionView
	Author     string
	Email      string
	MailTo     template.URL
	Site       string
	SiteLabel  string
}

type dossierSectionView struct {
	ID     string
	Title  string
	Groups []dossierGroup
}

// dossierGroup — un bloc, ou une suite de blocs de même famille : les puces en liste, les preuves
// en tableau (même regroupement qu'à l'écran, DesignDossier.tsx).
type dossierGroup struct {
	Kind    string // lead | paragraph | items | diagram | proofs | also | skills | contact
	Text    string
	Items   []string
	Diagram *dossierDiagramView
	Proofs  []dossierProofView
	Skills  *dossierSkillsView
}

// dossierSkillsView — la section des compétences, prête à imprimer : noms des projets résolus,
// niveaux traduits, et la RÉSERVE marquée (au-delà de DossierSkillsCore : imprimée seulement si la
// mise en page en a besoin pour remplir la dernière page).
type dossierSkillsView struct {
	Rows  []dossierSkillRow
	Techs []dossierTechRow
}

type dossierSkillRow struct {
	Name, Level, Sources string
	Need, Reserve        bool
}

type dossierTechRow struct {
	Name, Sources string
}

type dossierProofView struct {
	Name, Monogram, Why, Status, Loc, Tests string
	// Icon — l'icône du projet (data: URI) ; vide = le monogramme.
	Icon      template.URL
	DemoLabel string
	DemoURL   string
}

type dossierDiagramView struct {
	Label, Journal string
	ViewBox        string
	PxW, PxH       int
	BoxH           float64
	JournalY       float64
	JournalH       float64
	JournalW       float64
	JournalCX      float64
	JournalTextY   float64
	Boxes          []dossierBoxView
	Edges          []DossierPDFDiagramArc
	Steps          []dossierDotView
}

type dossierBoxView struct {
	X, Y, W, CX, TitleY, SubY float64
	Title, Subtitle           string
	Human                     bool
}

type dossierDotView struct {
	N     int
	X, Y  float64
	TextY float64
}

func buildDossierView(req *DossierPDFRequest, replay bool, m *Montagne, projects []MPFProject, skills DossierSkills, icons map[string]template.URL) dossierView {
	nodes := make(map[string]MontagneNoeud, len(m.Nodes))
	for _, n := range m.Nodes {
		nodes[n.ID] = n
	}
	fiches := make(map[string]MPFProject, len(projects))
	for _, p := range projects {
		fiches[p.ID] = p
	}
	nameOf := func(id string) string {
		if n, ok := nodes[id]; ok && n.Name != "" {
			return n.Name
		}
		return id
	}

	v := dossierView{
		Lang: req.Lang, Title: req.Title, Subtitle: req.Subtitle, Labels: req.Labels,
		Author: dossierAuthor, Email: dossierEmail, MailTo: template.URL("mailto:" + dossierEmail),
		Site: dossierSite, SiteLabel: strings.TrimPrefix(dossierSite, "https://"),
	}
	if replay {
		v.ReplayNote = req.Labels.ReplayNote
	}
	hasContact := false
	for _, sec := range req.Sections {
		if len(sec.Blocks) == 0 {
			continue // section jamais écrite (run arrêté) : pas de titre orphelin
		}
		sv := dossierSectionView{ID: sec.ID, Title: sec.Title}
		for _, b := range sec.Blocks {
			last := len(sv.Groups) - 1
			switch b.Type {
			case "item":
				if last >= 0 && sv.Groups[last].Kind == "items" {
					sv.Groups[last].Items = append(sv.Groups[last].Items, b.Text)
				} else {
					sv.Groups = append(sv.Groups, dossierGroup{Kind: "items", Items: []string{b.Text}})
				}
			case "proof":
				p := proofView(b, nodes[b.ProjectID], fiches[b.ProjectID], nameOf(b.ProjectID), req)
				p.Icon = icons[b.ProjectID]
				if last >= 0 && sv.Groups[last].Kind == "proofs" {
					sv.Groups[last].Proofs = append(sv.Groups[last].Proofs, p)
				} else {
					sv.Groups = append(sv.Groups, dossierGroup{Kind: "proofs", Proofs: []dossierProofView{p}})
				}
			case "also":
				names := make([]string, len(b.ProjectIDs))
				for i, id := range b.ProjectIDs {
					names[i] = nameOf(id)
				}
				sv.Groups = append(sv.Groups, dossierGroup{Kind: "also", Text: strings.ReplaceAll(req.Labels.Also, "{names}", strings.Join(names, ", "))})
			case "diagram":
				sv.Groups = append(sv.Groups, dossierGroup{Kind: "diagram", Diagram: diagramView(b.Diagram)})
			case "contact":
				hasContact = true
				sv.Groups = append(sv.Groups, dossierGroup{Kind: "contact"})
			default: // lead | paragraph
				sv.Groups = append(sv.Groups, dossierGroup{Kind: b.Type, Text: b.Text})
			}
		}
		v.Sections = append(v.Sections, sv)
		// Les compétences suivent les preuves dont elles sont tirées.
		if sec.ID == "proofs" && !skills.Empty() {
			v.Sections = append(v.Sections, skillsSection(req.Labels.Skills, skills, nameOf))
		}
	}
	// Le contact figure sur tout dossier imprimé, même si les agents ne l'ont pas posé.
	if !hasContact && len(v.Sections) > 0 {
		lastSec := &v.Sections[len(v.Sections)-1]
		lastSec.Groups = append(lastSec.Groups, dossierGroup{Kind: "contact"})
	}
	return v
}

// skillsSection — la section « Compétences mobilisées » : une ligne par compétence (nom, niveau,
// preuves d'où elle vient), puis les technologies partagées.
func skillsSection(l DossierPDFSkillLabels, d DossierSkills, nameOf func(string) string) dossierSectionView {
	names := func(ids []string) string {
		out := make([]string, len(ids))
		for i, id := range ids {
			out[i] = nameOf(id)
		}
		return strings.Join(out, ", ")
	}
	sv := &dossierSkillsView{}
	for i, s := range d.Skills {
		level := s.Level
		if lbl, ok := l.Levels[s.Level]; ok {
			level = lbl
		}
		sv.Rows = append(sv.Rows, dossierSkillRow{Name: s.Name, Level: level, Sources: names(s.Projects), Need: s.Need, Reserve: i >= DossierSkillsCore})
	}
	for _, t := range d.Technologies {
		sv.Techs = append(sv.Techs, dossierTechRow{Name: t.Name, Sources: names(t.Projects)})
	}
	return dossierSectionView{ID: "skills", Title: l.Title, Groups: []dossierGroup{{Kind: "skills", Text: l.Lead, Skills: sv}}}
}

func proofView(b DossierPDFBlock, n MontagneNoeud, f MPFProject, name string, req *DossierPDFRequest) dossierProofView {
	p := dossierProofView{Name: name, Monogram: n.Monogram, Why: b.Why, Status: "—", Loc: "—", Tests: "—"}
	if f.ID != "" {
		p.Status = statusLabel(f.Status, req.Labels.Statuses)
		// Un zéro dans la fiche veut dire « non mesuré » (même règle qu'à l'écran, data.ts).
		if f.Stats.LOC > 0 {
			p.Loc = groupDigits(f.Stats.LOC, req.Lang)
		}
		if f.Stats.Tests > 0 {
			p.Tests = groupDigits(f.Stats.Tests, req.Lang)
		}
	}
	if b.Demo != nil {
		p.DemoURL = b.Demo.URL
		p.DemoLabel = req.Labels.Video
		if b.Demo.Kind == "site" {
			p.DemoLabel = req.Labels.Site
		}
	}
	return p
}

func statusLabel(status string, labels map[string]string) string {
	if l, ok := labels[status]; ok {
		return l
	}
	if l, ok := labels[strings.ToLower(status)]; ok {
		return l
	}
	return status
}

// groupDigits — séparateur de milliers de la langue (ce qu'Intl.NumberFormat affiche à l'écran) :
// espace fine insécable en français, point en allemand et italien, virgule sinon.
func groupDigits(n int, lang string) string {
	sep := ","
	switch lang {
	case "fr":
		sep = "\u202f"
	case "de", "it":
		sep = "."
	}
	s := strconv.Itoa(n)
	var out strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteString(sep)
		}
		out.WriteRune(r)
	}
	return out.String()
}

// diagramView — positions des textes et cadre du SVG. Mêmes décalages que PipelineDiagram.tsx
// (titre à +17, sous-titre à +32, chiffre de pastille à +4) : le PDF dessine ce que l'écran dessine.
func diagramView(d *DossierPDFDiagram) *dossierDiagramView {
	vbW := d.Width + 2*d.Margin
	vbH := d.Height + 2*d.Margin
	v := &dossierDiagramView{
		Label: d.Label, Journal: d.Journal,
		ViewBox: fmt.Sprintf("%g %g %g %g", -d.Margin, -d.Margin, vbW, vbH),
		PxW:     int(math.Round(vbW * d.Scale)), PxH: int(math.Round(vbH * d.Scale)),
		BoxH: d.BoxH, JournalY: d.JournalY, JournalH: d.JournalH, JournalW: d.Width,
		JournalCX: d.Width / 2, JournalTextY: d.JournalY + 16,
		Edges: d.Edges,
	}
	for _, b := range d.Boxes {
		v.Boxes = append(v.Boxes, dossierBoxView{X: b.X, Y: b.Y, W: b.W, CX: b.X + b.W/2, TitleY: b.Y + 17, SubY: b.Y + 32, Title: b.Title, Subtitle: b.Subtitle, Human: b.Human})
	}
	for _, s := range d.Steps {
		v.Steps = append(v.Steps, dossierDotView{N: s.N, X: s.X, Y: s.Y, TextY: s.Y + 4})
	}
	return v
}
