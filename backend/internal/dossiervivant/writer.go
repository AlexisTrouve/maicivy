package dossiervivant

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// QUOI : le RÉDACTEUR. Ouvre le dossier dès le départ (plan en cinq sections), puis écrit chaque
// section dès que l'agent dont elle dépend a livré — relayé par l'orchestrateur :
//  1. le besoin (Lecteur)        — un appel : phrase d'ouverture + trois garanties ;
//  2. l'architecture (Architecte) — le schéma et les garde-fous de l'Architecte, tels quels ;
//  3. les preuves (Chercheur)    — une ligne par preuve, SANS chiffre (le front lit les fiches) ;
//  4. l'apport et 5. la suite    — un appel pour les deux, sur tout ce qui précède (dont le
//     brief complet de chaque preuve : l'apport cite ces projets).
//
// POURQUOI les sections 2 et 3 sans appel : schéma, garde-fous et raisons des preuves sont déjà
// écrits par l'agent qui les a établis ; les faire réécrire coûterait des tokens pour un risque de
// dérive (un garde-fou reformulé n'est plus celui qui a été vérifié).
//
// COMMENT le dossier se tape : chaque bloc part seul ; un paragraphe attend le temps de sa frappe
// avant le suivant (typePause), une puce le battement.
//
// COMMENT l'apport se rédige PENDANT la frappe : son appel ne dépend que du besoin, des garde-fous
// et des preuves. Il part dès que l'Architecte et le Chercheur ont livré, dans une goroutine, pendant
// que les sections 2 et 3 s'écrivent à l'écran — enchaîné après elles, il attendait leurs quelques
// secondes de frappe pour rien.
//
// COMMENT l'appel de l'apport tient sous le plafond (callsize.go) : il relit le brief COMPLET de
// chaque preuve. Les preuves entrent dans l'ordre du classement tant que leurs briefs tiennent ; une
// preuve qui ne tient plus passe dans « aussi retenus » — elle n'est jamais citée sans son brief.

type dossierOut struct {
	Proofs []keptProject
}

type sectionNeedOut struct {
	Lead  string   `json:"lead"`
	Items []string `json:"items"`
}

type closingOut struct {
	Contribution []string `json:"contribution"`
	Next         string   `json:"next"`
}

// Bornes des textes du dossier, en runes (cf. fit.go) : une puce, la phrase d'ouverture, le
// paragraphe de la suite.
const (
	itemMaxRunes = 110
	leadMaxRunes = 160
	nextMaxRunes = 280
)

func (r *run) writer(ctx context.Context, needIn *future[readerOut], archIn *future[archOut], proofsIn *future[researchOut]) (dossierOut, error) {
	r.status(Writer, "working", r.ph.get("writer.opening"))
	r.tool(Writer, fmt.Sprintf("create_dossier(%q)", r.brief.Organisation), onDossier)
	plan := []SectionPlan{
		{ID: SecNeed, WaitingFor: string(Reader)},
		{ID: SecArchitecture, WaitingFor: string(Architect)},
		{ID: SecProofs, WaitingFor: string(Researcher)},
		{ID: SecContribution, WaitingFor: string(Researcher)},
		{ID: SecNext, WaitingFor: "previous"},
	}
	for i := range plan {
		plan[i].Title = r.ph.get("section." + plan[i].ID)
	}
	r.emit.Emit(DossierOpened{Sections: plan})
	r.status(Writer, "waiting", r.ph.get("writer.wait.need"))
	r.note(Writer, "note", r.ph.get("writer.plan"))

	// 1. Le besoin.
	need, err := needIn.wait(ctx)
	if err != nil {
		return dossierOut{}, err
	}
	r.open(SecNeed, 1)
	var s1 sectionNeedOut
	prompt := needBlock(r.brief) + "<reader-notes>\n" + bullets(need.Notes) + "</reader-notes>\n"
	if err := r.ask(ctx, Writer, taskSectionNeed, prompt, &s1); err != nil {
		return dossierOut{}, err
	}
	items := firstLines(s1.Items, 3)
	r.fit(ctx, Writer, append(itemSpans(items, itemMaxRunes), span{text: &s1.Lead, max: leadMaxRunes})...)
	items = filled(items)
	r.block(ctx, SecNeed, Block{Type: "lead", Text: s1.Lead})
	for _, it := range items {
		r.block(ctx, SecNeed, Block{Type: "item", Text: it})
	}
	r.close(SecNeed, 1)

	// L'apport se rédige dès que ses entrées sont là, pendant que les sections 2 et 3 se tapent (cf.
	// en-tête). Deux goroutines sous un même `crew` : la première erreur arrête l'autre, et le
	// Rédacteur ne rend la main qu'une fois les deux finies.
	parts, drafted := newFuture[closingParts](), newFuture[closingOut]()
	work := newCrew(ctx)
	work.run(func(c context.Context) error {
		arch, err := archIn.wait(c)
		if err != nil {
			return err
		}
		research, err := proofsIn.wait(c)
		if err != nil {
			return err
		}
		p := r.closingFor(need, arch, research)
		parts.set(p)
		var cl closingOut
		if err := r.ask(c, Writer, taskClosing, p.prompt, &cl); err != nil {
			return err
		}
		// Réécrit ICI, pendant la frappe des sections 2 et 3, et non au moment d'afficher.
		cl.Contribution = firstLines(cl.Contribution, 3)
		r.fit(c, Writer, append(itemSpans(cl.Contribution, itemMaxRunes), span{text: &cl.Next, max: nextMaxRunes})...)
		cl.Contribution = filled(cl.Contribution)
		drafted.set(cl)
		return nil
	})
	var written dossierOut
	work.run(func(c context.Context) (err error) {
		written, err = r.writeRest(c, archIn, parts, drafted)
		return err
	})
	if err := work.wait(); err != nil {
		return dossierOut{}, err
	}
	return written, nil
}

// writeRest — les sections 2 à 5, dans l'ordre de lecture, pendant que l'apport se rédige.
func (r *run) writeRest(ctx context.Context, archIn *future[archOut], parts *future[closingParts], drafted *future[closingOut]) (dossierOut, error) {
	// 2. L'architecture.
	r.status(Writer, "waiting", r.ph.get("writer.wait.architecture"))
	arch, err := archIn.wait(ctx)
	if err != nil {
		return dossierOut{}, err
	}
	r.open(SecArchitecture, 2)
	r.block(ctx, SecArchitecture, arch.Pipeline)
	for _, g := range arch.Guardrails {
		r.block(ctx, SecArchitecture, Block{Type: "item", Text: g})
	}
	r.close(SecArchitecture, 2)

	// 3. Les preuves — celles que l'apport relit, dans l'ordre du classement.
	r.status(Writer, "waiting", r.ph.get("writer.wait.proofs"))
	p, err := parts.wait(ctx)
	if err != nil {
		return dossierOut{}, err
	}
	r.open(SecProofs, 3)
	for _, pr := range p.proofs {
		b := Block{Type: "proof", ProjectID: pr.ID, Why: pr.Why}
		if d, ok := r.catalog.demos[pr.ID]; ok {
			b.Demo = &d
		}
		r.block(ctx, SecProofs, b)
	}
	if len(p.also) > 0 {
		r.block(ctx, SecProofs, Block{Type: "also", ProjectIDs: p.also})
	}
	r.close(SecProofs, 3)

	// 4 et 5. L'apport, la suite.
	r.open(SecContribution, 4)
	cl, err := drafted.wait(ctx)
	if err != nil {
		return dossierOut{}, err
	}
	for _, it := range cl.Contribution {
		r.block(ctx, SecContribution, Block{Type: "item", Text: it})
	}
	r.close(SecContribution, 4)

	r.open(SecNext, 5)
	r.block(ctx, SecNext, Block{Type: "paragraph", Text: cl.Next})
	r.block(ctx, SecNext, Block{Type: "contact"})
	r.close(SecNext, 5)

	r.status(Writer, "done", "")
	r.note(Writer, "done", r.ph.get("writer.done"))
	return dossierOut{Proofs: p.proofs}, ctx.Err()
}

// closingParts — ce que l'apport relit, et ce que la section 3 affiche : les preuves dont le brief
// tient dans l'appel (dans l'ordre du classement), les autres retenus, et le message de l'appel.
type closingParts struct {
	proofs []keptProject
	also   []string
	prompt string
}

// closingFor — le message de l'apport et les preuves qu'il relit. Une preuve dont le brief ne tient
// plus sous le plafond d'appel (callsize.go) passe dans « aussi retenus » : citée sans son brief,
// elle ne serait plus une preuve jugée sur pièce.
func (r *run) closingFor(need readerOut, arch archOut, research researchOut) closingParts {
	head := needBlock(r.brief) +
		"<open-questions>\n" + bullets(need.Gaps) + "</open-questions>\n" +
		"<guardrails>\n" + bullets(arch.Guardrails) + "</guardrails>\n"
	prompt := func(proofs []keptProject) string { return head + "<proofs>\n" + proofBriefs(proofs) + "</proofs>\n" }
	var proofs []keptProject
	var overflow []string
	for _, p := range research.Proofs {
		if r.fits(taskClosing, prompt(append(proofs[:len(proofs):len(proofs)], p))) {
			proofs = append(proofs, p)
		} else {
			overflow = append(overflow, p.ID)
		}
	}
	if len(overflow) > 0 {
		log.Warn().Strs("ids", overflow).Msg("dossier vivant: preuves trop longues pour l'appel de l'apport, passées en « aussi retenus »")
	}
	return closingParts{
		proofs: proofs,
		also:   alsoKept(proofs, append(append(overflow, research.Also...), arch.Kept...)),
		prompt: prompt(proofs),
	}
}

// open — le Rédacteur se met à la section n.
func (r *run) open(section string, n int) {
	r.status(Writer, "working", r.ph.get("writer.write."+section))
	r.tool(Writer, fmt.Sprintf("write_section(%d, %s)", n, section), onDossier)
	r.emit.Emit(DossierSection{Section: section, Status: "writing"})
}

func (r *run) close(section string, n int) {
	r.emit.Emit(DossierSection{Section: section, Status: "done"})
	r.note(Writer, "done", r.ph.get("writer.sectionDone", n))
}

// block — un bloc du dossier, puis le temps que l'écran l'écrive.
func (r *run) block(ctx context.Context, section string, b Block) {
	r.emit.Emit(DossierBlock{Section: section, Block: b})
	switch b.Type {
	case "lead", "paragraph":
		r.typePause(ctx, b.Text)
	default:
		r.beat(ctx)
	}
}

// alsoKept — « Aussi retenus » : les autres retenus (`others`, dans leur ordre : preuves qui ne
// tenaient pas dans l'appel, retenus du Chercheur hors preuves, puis ceux de l'Architecte), sans
// doublon ni preuve affichée. Tous sont des cartes de la montagne (validés en amont).
func alsoKept(proofs []keptProject, others []string) []string {
	seen := map[string]bool{}
	for _, p := range proofs {
		seen[p.ID] = true
	}
	var out []string
	for _, id := range others {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// proofBriefs — les preuves telles que le Rédacteur les lit : le brief complet de chacune, suivi de
// la raison du Chercheur. POURQUOI le brief et pas « nom : raison » : « ce qu'Alexis apporte » cite
// ces projets ; une ligne de 90 caractères ne dit pas ce qu'il y a construit, le brief si.
func proofBriefs(proofs []keptProject) string {
	var sb strings.Builder
	for _, p := range proofs {
		sb.WriteString(projectBlock(p.brief, "why it proves the need: "+p.Why+"\n"))
	}
	return sb.String()
}
