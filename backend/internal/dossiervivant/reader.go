package dossiervivant

import (
	"context"
	"fmt"
	"strings"
)

// QUOI : le LECTEUR DU BESOIN. Lit la demande (un appel de modèle), en relève les faits et
// contraintes (notes) et ce qui manque pour construire (questions), rend compte à l'orchestrateur.
// Son livrable nourrit la section 1 du dossier et la « prochaine étape » (les questions ouvertes).
//
// COMMENT : le curseur travaille à côté de la tuile du visiteur (focus `visitor`) ; les notes et
// les questions partent une à une, au battement, dans l'ordre où le modèle les a rendues.

type readerOut struct {
	Notes   []string `json:"notes"`
	Gaps    []string `json:"gaps"`
	Summary string   `json:"summary"`
}

const (
	noteMaxRunes    = 80 // largeur d'une ligne de journal dans une colonne d'agent
	messageMaxRunes = 90 // un message du fil (compte rendu, consigne de l'orchestrateur)
	maxNotes        = 3
	maxGaps         = 3
)

func (r *run) reader(ctx context.Context) (readerOut, error) {
	r.status(Reader, "working", r.ph.get("reading"))
	r.tool(Reader, fmt.Sprintf("read_request(%q)", slug(r.brief.Organisation)), onVisitor)

	var out readerOut
	if err := r.ask(ctx, Reader, taskReader, needBlock(r.brief), &out); err != nil {
		return readerOut{}, err
	}
	out.Notes = firstLines(out.Notes, maxNotes)
	out.Gaps = firstLines(out.Gaps, maxGaps)
	// Notes, questions et compte rendu s'affichent tous : une seule réécriture pour ce qui dépasse.
	r.fit(ctx, Reader, append(append(itemSpans(out.Notes, noteMaxRunes), itemSpans(out.Gaps, noteMaxRunes)...),
		span{text: &out.Summary, max: messageMaxRunes})...)
	out.Notes, out.Gaps = filled(out.Notes), filled(out.Gaps)
	for _, n := range out.Notes {
		r.note(Reader, "note", n)
		r.beat(ctx)
	}

	r.status(Reader, "working", r.ph.get("reader.gaps"))
	for _, g := range out.Gaps {
		r.note(Reader, "question", g)
		r.beat(ctx)
	}
	r.tool(Reader, `send("orchestrator", need)`, onVisitor)
	r.status(Reader, "done", "")
	r.note(Reader, "done", r.ph.get("reader.done"))
	return out, ctx.Err()
}

// slug — l'argument lisible d'un appel d'outil (« Tessera » → « tessera »).
func slug(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), "-"))
}
