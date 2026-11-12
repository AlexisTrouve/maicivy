package dossiervivant

import (
	"context"
	"strings"

	"maicivy/internal/services"
)

// QUOI : les OUTILS des agents — ce qu'ils lisent du monde réel.
//   - l'INDEX : les projets de la montagne (GET /api/v1/montagne côté service), un par ligne ;
//   - la FICHE d'un projet : maiProFiles via le client existant (récit + statut + chiffres) ;
//   - le LIVRE : la couche curée du portfolio (GET /book?web=true), lue une fois par run ; son
//     entrée complète la fiche dans le brief d'un projet curé (cf. brief.go) ;
//   - les DÉMOS : la vidéo d'une carte du sommet, si la montagne en porte une.
//
// POURQUOI la montagne comme index (et pas /projects/index de maiProFiles) : l'écran dessine la
// montagne. Un projet que l'agent retiendrait hors montagne s'allumerait… nulle part. L'index des
// agents est donc EXACTEMENT l'ensemble des cartes affichées, et c'est contre lui que le code
// valide chaque identifiant rendu par un modèle : un id inventé est écarté, jamais émis.
//
// COMMENT les chiffres restent hors du modèle : la fiche donne au modèle statut, lignes et tests
// pour qu'il juge sur du réel, mais aucun événement n'en porte — le tableau des preuves du front
// les relit sur la fiche (GET /api/v1/projects). Le modèle reçoit en outre la consigne de n'en
// citer aucun.

// Sources — les lectures réelles d'un run. Interface : les tests branchent une source en mémoire.
type Sources interface {
	Montagne(ctx context.Context, lang string) (*services.Montagne, error)
	Project(ctx context.Context, id, lang string) (*services.MPFProject, error)
	Book(ctx context.Context, lang string) (*services.MPFBook, error)
}

type liveSources struct {
	montagne *services.MontagneService
	profiles *services.MaiProFilesClient
}

// NewLiveSources — la montagne (cache SWR Redis) et le client maiProFiles du backend.
func NewLiveSources(m *services.MontagneService, p *services.MaiProFilesClient) Sources {
	return liveSources{montagne: m, profiles: p}
}

func (s liveSources) Montagne(ctx context.Context, lang string) (*services.Montagne, error) {
	return s.montagne.Get(ctx, lang)
}

func (s liveSources) Project(ctx context.Context, id, lang string) (*services.MPFProject, error) {
	return s.profiles.GetProject(ctx, id, lang)
}

// Book — l'édition du livre dans la langue du run (le client replie fr/en/zh, cf. langueDuLivre).
func (s liveSources) Book(ctx context.Context, lang string) (*services.MPFBook, error) {
	return s.profiles.GetPortfolioBook(ctx, lang)
}

type indexEntry struct {
	ID, Name, Mini, Status string
}

// catalog — l'index d'un run, figé à son début.
type catalog struct {
	entries []indexEntry
	byID    map[string]indexEntry
	demos   map[string]Demo
	book    map[string]services.MPFBookProject // entrées du livre des cartes curées, par id
}

func newCatalog(m *services.Montagne, b *services.MPFBook) *catalog {
	c := &catalog{byID: make(map[string]indexEntry, len(m.Nodes)), demos: map[string]Demo{}, book: map[string]services.MPFBookProject{}}
	for _, n := range m.Nodes {
		e := indexEntry{ID: n.ID, Name: n.Name, Mini: n.Mini, Status: n.Status}
		c.entries = append(c.entries, e)
		c.byID[n.ID] = e
	}
	for _, p := range b.Projects {
		if c.has(p.ID) {
			c.book[p.ID] = p
		}
	}
	// Démo d'un projet = la vidéo de SA carte du sommet (et non celle d'un projet qu'elle absorbe).
	for _, card := range m.Summit {
		for _, p := range card.Proofs {
			if p.Kind == "video" && (p.ProjectID == "" || p.ProjectID == card.ID) {
				c.demos[card.ID] = Demo{Kind: "video", URL: p.URL}
				break
			}
		}
	}
	return c
}

func (c *catalog) has(id string) bool { _, ok := c.byID[id]; return ok }

func (c *catalog) name(id string) string { return c.byID[id].Name }

// validate — garde les ids qui existent sur la montagne, dans l'ordre, sans doublon, au plus `max`.
// Les autres sont rendus à part : l'appelant les journalise, aucun n'atteint le flux.
func (c *catalog) validate(ids []string, max int) (valid, rejected []string) {
	seen := map[string]bool{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		switch {
		case seen[id]:
		case !c.has(id):
			rejected = append(rejected, id)
		case len(valid) < max:
			valid = append(valid, id)
		}
		seen[id] = true
	}
	return valid, rejected
}

// indexLines — l'index en contexte de modèle : `id | nom — mini [production]`, une ligne par projet
// (fin de ligne comprise). Même forme compacte que l'inventaire du chat (chat_projects_index.go) :
// le statut `production` est la seule preuve de réalité, les autres statuts ne sont pas rendus.
// En lignes et non en un bloc : indexPrompt n'en fait entrer que ce qui tient dans un appel.
func (c *catalog) indexLines() []string {
	lines := make([]string, len(c.entries))
	for i, e := range c.entries {
		var b strings.Builder
		b.WriteString(e.ID)
		b.WriteString(" | ")
		b.WriteString(e.Name)
		if e.Mini != "" {
			b.WriteString(" — ")
			b.WriteString(e.Mini)
		}
		if e.Status == "production" {
			b.WriteString(" [production]")
		}
		b.WriteByte('\n')
		lines[i] = b.String()
	}
	return lines
}
