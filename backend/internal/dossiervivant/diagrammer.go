package dossiervivant

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// QUOI : le DESSINATEUR. Un agent à UN seul travail : le schéma d'architecture du dossier (nœuds,
// liens, embranchements, libellés, bande du journal). L'Architecte le lance par son outil
// `draw_schema` et lui transmet sa conception ; le Dessinateur rend une grille, le code la valide,
// l'Architecte la reçoit.
//
// POURQUOI un agent à part, sur un autre modèle (Options.SchemaModel, Sonnet par défaut, effort
// moyen) : le schéma est la figure qu'un décideur lit en premier, et c'est la seule sortie du run où
// le jugement de conception SE VOIT — dessiné par Haiku il restait une chaîne générique de boîtes.
// Un seul appel par run : le surcoût reste borné, et rien d'autre ne passe sur ce modèle — même la
// réécriture d'un libellé qui dépasse sa boîte (fit.go) part sur le modèle des agents.
//
// COMMENT il travaille sous les yeux : colonne propre (agent `diagrammer` du contrat), statut,
// appel d'outil, notes de ses décisions de dessin, compteur de tokens à son tarif ; il rend compte à
// l'Architecte (message), jamais à l'orchestrateur. Un schéma hors contrat de grille est REFUSÉ —
// note à l'écran puis erreur nommée (ErrSchemaRejected, raison `model`) — jamais affiché cassé.

// ErrSchemaRejected — le schéma rendu par le Dessinateur sort du contrat de la grille (cf. drawable).
var ErrSchemaRejected = errors.New("dossier vivant: schéma hors contrat")

const (
	schemaTask = "schema"
	// schemaEffort — l'effort « moyen » demandé à la passerelle (thinking adaptive + output_config).
	schemaEffort = "medium"
	// schemaMaxTokens — sortie RÉFLEXION COMPRISE : le JSON d'un schéma pèse ~800 tokens ; le reste
	// est la marge de réflexion d'un effort moyen.
	schemaMaxTokens = 8000
	// schemaTimeout — délai de CET appel (une passerelle muette au-delà est une panne, raison
	// `upstream`) : un appel qui réfléchit dépasse les 30 s réglés pour les appels courts.
	schemaTimeout  = 75 * time.Second
	maxSchemaNotes = 2
	// Bornes des libellés du schéma, en runes : la largeur d'une boîte (titre, sous-titre), de la
	// bande du journal, et la description pour lecteur d'écran.
	nodeTitleMaxRunes    = 16
	nodeSubtitleMaxRunes = 20
	journalMaxRunes      = 60
	labelMaxRunes        = 200
)

// schemaOrder — ce que l'Architecte transmet par `draw_schema`.
type schemaOrder struct {
	Need       readerOut // le besoin reformulé par le Lecteur (faits, contraintes, questions ouvertes)
	Stages     []string  // la chaîne conçue par l'Architecte, dans l'ordre
	Branches   []string  // où le flux quitte la voie principale
	Guardrails []string  // numérotés 1..n : chacun doit porter sa pastille sur une arête
	Lessons    []string  // « Projet : idée à réutiliser », tirées des briefs lus
}

// schemaOut — la réponse du Dessinateur.
type schemaOut struct {
	Notes   []string       `json:"notes"`
	Nodes   []PipelineNode `json:"nodes"`
	Edges   []PipelineEdge `json:"edges"`
	Journal string         `json:"journal"`
	Label   string         `json:"label"`
	Summary string         `json:"summary"`
}

func (r *run) diagrammer(ctx context.Context, o schemaOrder) (Block, error) {
	r.status(Diagrammer, "working", r.ph.get("diag.reading"))
	r.tool(Diagrammer, "draft_schema(need, design)", onDossier)
	var out schemaOut
	k := call{task: schemaTask, model: r.schemaModel, effort: schemaEffort, timeout: schemaTimeout,
		system: schemaPrompt(r.lang), prompt: o.prompt(r.brief), maxTokens: schemaMaxTokens}
	if err := r.complete(ctx, Diagrammer, k, &out); err != nil {
		return Block{}, err
	}

	r.status(Diagrammer, "working", r.ph.get("diag.checking"))
	r.tool(Diagrammer, "check_grid(nodes, edges)", onDossier)
	nodes, edges, err := drawable(out.Nodes, out.Edges, len(o.Guardrails))
	if err != nil {
		r.note(Diagrammer, "note", r.ph.get("diag.rejected"))
		return Block{}, err
	}
	// Libellés des boîtes, bande du journal, description, notes et compte rendu s'affichent tous
	// (schéma du dossier et du PDF, colonne, fil) : une seule réécriture pour ce qui dépasse.
	notes := firstLines(out.Notes, maxSchemaNotes)
	spans := append(itemSpans(notes, noteMaxRunes), span{text: &out.Summary, max: messageMaxRunes},
		span{text: &out.Journal, max: journalMaxRunes}, span{text: &out.Label, max: labelMaxRunes})
	for i := range nodes {
		spans = append(spans, span{text: &nodes[i].Title, max: nodeTitleMaxRunes}, span{text: &nodes[i].Subtitle, max: nodeSubtitleMaxRunes})
	}
	r.fit(ctx, Diagrammer, spans...)
	notes = filled(notes)
	for _, n := range notes {
		r.beat(ctx)
		r.note(Diagrammer, "note", n)
	}
	r.note(Diagrammer, "note", r.ph.get("diag.drawn", len(nodes), len(edges)))
	r.status(Diagrammer, "done", "")
	r.note(Diagrammer, "done", r.ph.get("diag.done"))
	r.say(string(Diagrammer), string(Architect), out.Summary, "")
	return Block{Type: "pipeline", Nodes: nodes, Edges: edges, Journal: out.Journal, Label: out.Label}, ctx.Err()
}

// prompt — le message du Dessinateur : le besoin tel que tapé, tel que reformulé, puis la
// conception de l'Architecte. Garde-fous NUMÉROTÉS : le numéro est celui de la pastille à poser.
func (o schemaOrder) prompt(b VisitorBrief) string {
	var sb strings.Builder
	sb.WriteString(needBlock(b))
	sb.WriteString("<restated-need>\n" + bullets(o.Need.Notes) + "</restated-need>\n")
	if len(o.Need.Gaps) > 0 {
		sb.WriteString("<open-questions>\n" + bullets(o.Need.Gaps) + "</open-questions>\n")
	}
	sb.WriteString("<architect-design>\n<stages>\n")
	for i, s := range o.Stages {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, s)
	}
	sb.WriteString("</stages>\n<branches>\n" + bullets(o.Branches) + "</branches>\n<guardrails>\n")
	for i, g := range o.Guardrails {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, g)
	}
	sb.WriteString("</guardrails>\n</architect-design>\n")
	if len(o.Lessons) > 0 {
		sb.WriteString("<lessons-from-alexis-projects>\n" + bullets(o.Lessons) + "</lessons-from-alexis-projects>\n")
	}
	return sb.String()
}

// schemaPrompt — la consigne du Dessinateur. Première ligne `dossier-vivant/schema` (journaux de
// la passerelle, faux serveur des tests), comme toutes les tâches du run. Les règles de voix et de
// chiffres sont celles de toute l'équipe (systemPrompt) ; s'y ajoutent ce qui rend un schéma digne
// d'être lu, et le contrat de grille que le code fait respecter (drawable).
//
// Les longueurs demandées (12 / 16 / 40) sont SOUS les bornes tenues par le code (16 / 20 / 60,
// cf. fit.go) : premier run réel, avec 16 / 20 demandés, Sonnet a rendu « Contrôle anomalies » et
// « Règles du cabinet » (18 et 17 caractères), qui ne tenaient pas dans leur boîte. Et chaque
// dépassement coûte une réécriture sur le chemin le plus long du run : la marge est large.
func schemaPrompt(lang string) string {
	return fmt.Sprintf(`dossier-vivant/%s
You are the diagram drafter in a team of AI agents writing Alexis Trouvé's PROPOSAL to a prospective client: a short design dossier for the system that Alexis — a freelance engineer who builds AI agent systems — proposes to build for them. The architect has designed that system; your only job is its architecture diagram, the one figure of the dossier. A decision-maker must grasp the proposed system from the diagram alone, in ten seconds.

What makes the diagram worth reading:
- Specific to THIS client: name each step with the client's own objects, teams and words, never with generic boxes such as "Input", "Processing", "Output", "Reception".
- Who acts at each step is visible: the subtitle names the mechanism — a language model, deterministic code (rules, database, schema check) or a human role. Where a model acts and where it must not is the heart of the design: make that boundary obvious.
- Where a human decides is visible (human: true), and what escalates to them is a dashed edge.
- Each numbered guardrail sits on the edge where it is enforced: that edge carries "step": its number.
- Merge or drop what adds nothing; 5 to 7 nodes. Follow the architect's design; improve its wording, not its substance.

Grid contract — the code draws the diagram and REJECTS anything outside it:
- "col" 0 to 5 = stage, left to right; "row" 0 = the main path, rows 1 and 2 = branches (exception, human review, feedback), placed under the step they leave from. Two nodes never share a cell.
- "id": short lowercase ASCII (a-z, 0-9, "-"), unique. Edges use node ids only, never link a node to itself; every node has at least one edge.
- "step": each guardrail number from 1 to the number of guardrails appears on exactly one edge; no other edge has a step.
- "title" at most 12 characters, "subtitle" at most 16 characters, spaces included — count them before answering and abbreviate rather than exceed: the box has no room for more.

Rules:
- Write every text value in %s. No marketing words.
- The diagram shows the system Alexis would build, not the agents writing this dossier; never write as if the client already owned it.
- Never state numbers about Alexis's projects.
- Answer with ONE JSON object of exactly this shape, nothing before or after it, no code fence:
{"notes": ["2 lines, <= 60 chars each: the design decisions your diagram makes visible"],
 "nodes": [{"id": "...", "title": "...", "subtitle": "...", "col": 0, "row": 0, "human": false}],
 "edges": [{"from": "node id", "to": "node id", "dashed": false, "step": 1}],
 "journal": "<= 40 chars: what the audit trail under the diagram records",
 "label": "<= 160 chars: plain-text description of the diagram for screen readers",
 "summary": "<= 60 chars: your report to the architect"}`, schemaTask, languageNames[lang])
}

// Bornes de la grille du schéma : la largeur du dossier tient six colonnes, trois rangées.
const (
	pipeMinNodes = 3
	pipeMaxNodes = 7
	pipeMaxCol   = 5
	pipeMaxRow   = 2
)

// nodeID — un identifiant de nœud tel que le contrat l'attend : court, ASCII, sans espace.
var nodeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)

// drawable — vérifie que le schéma rendu par le modèle se DESSINE et tient son contrat (cf. frontend
// pipeline.ts : nœuds sur une grille sans collision, arêtes entre nœuds existants). Les libellés
// sont ramenés à la largeur d'une boîte ensuite, par le Dessinateur (fit.go). `guardrails` = nombre de garde-fous : chacun doit porter sa
// pastille (1..n) sur exactement une arête — un garde-fou listé sous le schéma sans sa pastille ne
// renverrait à rien. Toute violation est une erreur NOMMÉE (ErrSchemaRejected) : le schéma n'est
// jamais affiché à moitié juste.
func drawable(nodes []PipelineNode, edges []PipelineEdge, guardrails int) ([]PipelineNode, []PipelineEdge, error) {
	reject := func(format string, args ...any) error {
		return fmt.Errorf("%w: %w: "+format, append([]any{ErrModelOutput, ErrSchemaRejected}, args...)...)
	}
	if len(nodes) < pipeMinNodes || len(nodes) > pipeMaxNodes {
		return nil, nil, reject("schéma à %d nœuds", len(nodes))
	}
	ids := map[string]bool{}
	cells := map[[2]int]bool{}
	out := make([]PipelineNode, len(nodes))
	for i, n := range nodes {
		cell := [2]int{n.Col, n.Row}
		switch {
		case !nodeID.MatchString(n.ID) || ids[n.ID]:
			return nil, nil, reject("nœud %q invalide ou en double", n.ID)
		case n.Col < 0 || n.Col > pipeMaxCol || n.Row < 0 || n.Row > pipeMaxRow:
			return nil, nil, reject("nœud %q hors grille (%d, %d)", n.ID, n.Col, n.Row)
		case cells[cell]:
			return nil, nil, reject("deux nœuds en (%d, %d)", n.Col, n.Row)
		case strings.TrimSpace(n.Title) == "":
			return nil, nil, reject("nœud %q sans titre", n.ID)
		}
		ids[n.ID], cells[cell] = true, true
		out[i] = n
	}
	var kept []PipelineEdge
	seen := map[[2]string]bool{}
	linked := map[string]bool{}
	steps := map[int]int{}
	for _, e := range edges {
		key := [2]string{e.From, e.To}
		if !ids[e.From] || !ids[e.To] || e.From == e.To {
			return nil, nil, reject("arête %s → %s hors schéma", e.From, e.To)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		if e.Step < 0 || e.Step > guardrails {
			return nil, nil, reject("pastille %d sur %s → %s (garde-fous : %d)", e.Step, e.From, e.To, guardrails)
		}
		if e.Step > 0 {
			steps[e.Step]++
		}
		linked[e.From], linked[e.To] = true, true
		kept = append(kept, e)
	}
	if len(kept) == 0 {
		return nil, nil, reject("schéma sans arête")
	}
	for _, n := range out {
		if !linked[n.ID] {
			return nil, nil, reject("nœud %q isolé", n.ID)
		}
	}
	for s := 1; s <= guardrails; s++ {
		if steps[s] != 1 {
			return nil, nil, reject("garde-fou %d porté par %d arêtes", s, steps[s])
		}
	}
	return out, kept, nil
}
