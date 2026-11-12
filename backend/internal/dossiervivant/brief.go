package dossiervivant

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/rs/zerolog/log"

	"maicivy/internal/services"
)

// QUOI : le BRIEF d'un projet — tout ce qu'un agent a sous les yeux quand il TRANCHE sur ce projet :
//   - la fiche technique maiProFiles ENTIÈRE (statut, stack, chiffres, résumé, récit `portfolio`) ;
//   - l'entrée du livre (`book.yaml`, couche curée : récit, domaine, chiffres) quand le projet y est.
//
// POURQUOI : leçon payée sur ce projet (CLAUDE.md, « Ajouter du contexte peut RÉDUIRE
// l'investigation ») — un `mini` de 100 caractères remplaçait en silence une fiche de ~2 800. Ici la
// fiche était rognée à 1 600 caractères (la conclusion « valeur transférable » tombait), le livre
// n'était jamais lu, et un candidat non ouvert recevait un `drop` sans lecture. Un agent qui
// retient ou écarte un projet doit l'avoir lu en entier.
//
// COMMENT le CODE tient l'invariant (pas une consigne de prompt) :
//   - judgeOn est le SEUL appel qui décide sur un projet : il construit lui-même le prompt à partir
//     du brief, impossible de l'appeler sans ;
//   - il rend un `judged`, et emitVerdict n'accepte QUE ce jeton : aucun `project.verdict` ne peut
//     partir pour un projet qu'aucun appel n'a jugé sur son brief ;
//   - les preuves du dossier sont des keptProject, construits à partir d'un `judged` ;
//   - aucun rognage : un brief qui ne tient pas dans un appel (callsize.go) n'est PAS jugé — note
//     visible dans la colonne de l'agent, aucun verdict — jamais raccourci en silence.
// Verrouillé par TestRun_VerdictEtPreuveSurBriefComplet.

// projectBrief — le brief d'un projet, tel que le modèle le lit. Construit par readBrief seulement.
type projectBrief struct {
	ID, Name string
	Curated  bool   // le projet a une entrée dans le livre
	text     string // le brief rendu pour le modèle
}

// judged — preuve, pour le compilateur, qu'un appel de modèle a jugé ce projet SUR son brief.
// Seul judgeOn en fabrique.
type judged struct {
	brief projectBrief
}

// readBrief — lit la fiche de `id` (maiProFiles) et y joint l'entrée du livre si le projet est curé.
func (r *run) readBrief(ctx context.Context, id string) (projectBrief, error) {
	p, err := r.sources.Project(ctx, id, r.lang)
	if err != nil {
		return projectBrief{}, fmt.Errorf("%w: fiche %s : %w", ErrUpstream, id, err)
	}
	b := projectBrief{ID: id, Name: r.catalog.name(id)}
	var sb strings.Builder
	fmt.Fprintf(&sb, "id: %s\nname: %s\nstatus: %s\ncategory: %s\nstack: %s\nlines of code: %d\ntests: %d\nsummary: %s\n",
		id, b.Name, p.Status, p.Category, strings.Join(p.Stack, ", "), p.Stats.LOC, p.Stats.Tests, strings.TrimSpace(p.Description.Short))
	sb.WriteString("technical file:\n")
	sb.WriteString(strings.TrimSpace(p.Description.Portfolio))
	sb.WriteByte('\n')
	if e, ok := r.catalog.book[id]; ok {
		b.Curated = true
		sb.WriteString(bookText(e))
	}
	b.text = sb.String()
	r.reads.add(id, len([]rune(b.text)), b.Curated)
	return b, nil
}

// bookText — l'entrée curée du livre : domaine, stack affichée, chiffres mis en avant, récit.
func bookText(e services.MPFBookProject) string {
	var sb strings.Builder
	sb.WriteString("portfolio book entry (curated):\n")
	if e.IndexDomain != "" {
		fmt.Fprintf(&sb, "domain: %s\n", e.IndexDomain)
	}
	if e.IndexStack != "" {
		fmt.Fprintf(&sb, "stack shown: %s\n", e.IndexStack)
	}
	if len(e.Stats) > 0 {
		figures := make([]string, len(e.Stats))
		for i, s := range e.Stats {
			figures[i] = strings.TrimSpace(s.Value + s.Label)
		}
		fmt.Fprintf(&sb, "highlights: %s\n", strings.Join(figures, "; "))
	}
	for _, p := range e.Paragraphs {
		sb.WriteString(strings.TrimSpace(p))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// projectBlock — le brief balisé dans un prompt. `extra` (facultatif) s'ajoute dans la balise.
func projectBlock(b projectBrief, extra string) string {
	return "<project>\n" + b.text + extra + "</project>\n"
}

// judgeOn — `agent` décide sur le projet `b` (tâche `t`), le brief complet dans le prompt. `ok` faux
// = le brief ne tient pas dans un appel : aucun appel n'est fait, l'agent le dit dans sa colonne, et
// aucun verdict ne pourra partir (pas de `judged`).
func (r *run) judgeOn(ctx context.Context, agent AgentID, t task, b projectBrief, out any) (j judged, ok bool, err error) {
	prompt := needBlock(r.brief) + projectBlock(b, "")
	if !r.fits(t, prompt) {
		log.Warn().Str("project", b.ID).Int("tokens", estimateTokens(prompt)).Msg("dossier vivant: brief trop long pour un appel, non jugé")
		r.note(agent, "note", cutAtBoundary(r.ph.get("brief.tooLong", b.Name), noteMaxRunes))
		return judged{}, false, nil
	}
	if err := r.ask(ctx, agent, t, prompt, out); err != nil {
		return judged{}, false, err
	}
	return judged{brief: b}, true, nil
}

// emitVerdict — publie le verdict d'un projet jugé, en note « Nom : raison ». Le nom affiché est
// celui de la montagne, jamais celui que le modèle aurait écrit. `reason` tient déjà dans
// reasonRoom (l'appelant l'a passée par fit) : la note n'est plus rognée ici.
func (r *run) emitVerdict(agent AgentID, j judged, keep bool, reason string) {
	verdict := "drop"
	if keep {
		verdict = "keep"
	}
	note := j.brief.Name + r.ph.get("colon") + reason
	r.emit.Emit(ProjectVerdict{Agent: agent, ProjectID: j.brief.ID, Verdict: verdict, Note: note})
}

// reasonRoom — la place de la raison dans la note « Nom : raison » d'un verdict : une ligne de
// journal (noteMaxRunes) moins le nom. POURQUOI : la consigne borne la raison seule, le nom vient
// devant ; borner la note entière coupait la fin des raisons des projets aux noms longs (mesuré :
// « ClaudeOrchestrator : Pipeline asynchrone multi-agents avec validation humaine e… »). Un nom
// démesuré laisse au moins une demi-ligne à la raison : la note passe alors à la ligne, entière.
func (r *run) reasonRoom(name string) int {
	return max(noteMaxRunes-utf8.RuneCountInString(name+r.ph.get("colon")), noteMaxRunes/2)
}

// readLedger — ce qui a été lu pendant un run, pour la mesure (journal de fin de run).
type readLedger struct {
	mu      sync.Mutex
	reads   int            // briefs lus (un projet lu par deux agents compte deux fois)
	chars   map[string]int // taille du brief, par projet
	curated int            // projets distincts lus qui ont une entrée du livre
}

func newReadLedger() *readLedger { return &readLedger{chars: map[string]int{}} }

func (l *readLedger) add(id string, chars int, curated bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reads++
	if _, seen := l.chars[id]; !seen && curated {
		l.curated++
	}
	l.chars[id] = chars
}

// summary — « briefs lus, projets distincts, curés, caractères min/moyen/max par brief ».
func (l *readLedger) summary() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.chars) == 0 {
		return "aucun brief lu"
	}
	lo, hi, sum := -1, 0, 0
	for _, n := range l.chars {
		sum += n
		hi = max(hi, n)
		if lo < 0 || n < lo {
			lo = n
		}
	}
	return fmt.Sprintf("%d lectures, %d projets (%d curés), brief min/moy/max %d/%d/%d car.",
		l.reads, len(l.chars), l.curated, lo, sum/len(l.chars), hi)
}
