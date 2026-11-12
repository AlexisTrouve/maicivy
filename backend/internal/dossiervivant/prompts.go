package dossiervivant

import (
	"fmt"
	"strings"
)

// QUOI : les consignes de chaque tâche de modèle du run, et le plafond de sortie de chacune.
//
// POURQUOI en anglais : ce sont des consignes pour le modèle, pas des textes affichés ; la langue
// de SORTIE est imposée à part (`Write every text value in …`), celle de l'interface du visiteur.
//
// COMMENT :
//   - la première ligne du prompt système nomme la tâche (`dossier-vivant/<tâche>`) : elle se lit
//     dans les journaux de la passerelle, et le faux serveur des tests s'en sert pour répondre ;
//   - toute réponse est UN objet JSON, de forme décrite dans la consigne, décodé par le code ;
//   - aucune consigne ne demande de chiffre : statut, lignes et tests viennent des fiches.

type task struct {
	name string
	role string
	// maxTokens — plafond de sortie, RÉFLEXION COMPRISE : claude-haiku-5-5 ouvre parfois sa réponse
	// par un bloc `thinking` qui se paie sur ce plafond. Dimensionné à « réponse attendue + réflexion »
	// (premier run réel : 400 tokens entièrement brûlés en réflexion sur `refs`, aucun texte rendu).
	// Les agents demandent un effort bas (mind.go › agentEffort) : la marge de réflexion sert rarement,
	// mais une troncature reste un arrêt nommé — le plafond n'est donc pas réduit.
	maxTokens int
	// shape — la forme JSON attendue. Les longueurs des textes AFFICHÉS y sont demandées sous leur
	// borne d'affichage ; ce qui dépasse est réécrit par l'agent (fit.go), jamais coupé. En
	// CARACTÈRES, pas en mots : mesuré le 2026-11-07 sur quinze runs réels, des longueurs dites en
	// mots font réfléchir claude-haiku-5-5 à les compter — verdicts à 747 tokens de sortie en médiane
	// au lieu de 546, vingt au-delà de 1 000 au lieu de deux, et un verdict TRONQUÉ (run arrêté).
	// Le plan, sur le chemin de tout le run, demande plus court que sa borne (90) : un dépassement y
	// retarde les quatre agents d'une réécriture.
	shape string
}

var (
	taskPlan = task{"plan", "the orchestrator", 1500,
		`{"plan": "<= 60 chars: how you split the work between four agents working in parallel",
 "tasks": {"reader": "<= 60 chars, imperative", "researcher": "...", "architect": "...", "writer": "..."}}
The reader restates the need and lists open questions; the researcher finds Alexis's projects that prove he can build it; the architect designs the pipeline; the writer assembles the dossier.`}

	taskReader = task{"reader", "the need reader", 1800,
		`{"notes": ["3 notes, <= 70 chars each: the facts, constraints and expectations stated in the need"],
 "gaps": ["1 to 3 questions, <= 60 chars each: what the need does not say and a builder must know"],
 "summary": "<= 70 chars: your report to the orchestrator"}`}

	taskFilter = task{"filter", "the proof researcher", 2000,
		`{"candidates": ["5 to 7 project ids, copied EXACTLY from the index, best first, most likely to prove Alexis can build this; prefer [production] projects"]}
Each candidate will be opened and read in full before any decision: pick the ones worth reading.`}

	taskVerdict = task{"verdict", "the proof researcher", 1500,
		`{"verdict": "keep" or "drop",
 "reason": "<= 55 chars: why, without repeating the project name",
 "why": "<= 90 chars: what Alexis built in this project that proves he can build THIS need, one sentence for the dossier",
 "score": 0-100 relevance as a proof for this need}
Read the whole project brief (technical file, and the portfolio book entry when present) before deciding. Keep only projects whose architecture or production use genuinely supports the need; base "why" on what the brief says was actually built.`}

	taskRefs = task{"refs", "the architect", 1500,
		`{"refs": ["2 or 3 project ids, copied EXACTLY from the index, whose architecture best informs the solution"]}`}

	taskLesson = task{"lesson", "the architect", 1200,
		`{"useful": true or false (does this architecture inform the solution?),
 "lesson": "<= 55 chars: the architectural idea to reuse, without repeating the project name"}`}

	// taskDesign — la CONCEPTION de l'Architecte. Il ne dessine plus : il décrit la chaîne (étapes,
	// embranchements, garde-fous) et la confie au Dessinateur (diagrammer.go), qui la met en schéma.
	taskDesign = task{"design", "the architect", 2500,
		`{"stages": ["4 to 6 stages of the proposed system, in order, <= 70 chars each: what happens, on which of the client's objects, and what performs it (a language model, deterministic code, or a named human role)"],
 "branches": ["0 to 2, <= 90 chars each: where the flow leaves the main path (exception, escalation, human review, feedback loop) and where it goes"],
 "guardrails": ["exactly 3, <= 60 chars each: the controls the system enforces, each at one precise stage"],
 "check": "<= 60 chars: how the pipeline covers the need",
 "summary": "<= 70 chars: your report to the orchestrator"}
Your design is handed to the diagram drafter, who draws the dossier's architecture diagram from it: be concrete — the client's own documents and teams, who acts at each stage, where a model acts and where it must not, where a human decides.`}

	taskSectionNeed = task{"section-need", "the writer", 1500,
		`{"lead": "<= 110 chars: the client's problem restated in one sentence — what they face today, not the solution, no salutation",
 "items": ["exactly 3, <= 90 chars each: what the system Alexis proposes must guarantee"]}`}

	taskClosing = task{"closing", "the writer", 2000,
		`{"contribution": ["exactly 3, <= 90 chars each: what Alexis concretely brings, each grounded in what one of the proof projects below shows he has built (name it)"],
 "next": "<= 220 chars: the next step Alexis proposes — a short call to settle the open questions, then a first prototype that he builds"}`}
)

// systemPrompt — consigne complète d'une tâche, dans la langue d'écriture du run.
//
// LA VOIX du dossier : c'est la PROPOSITION d'Alexis. Premier run réel : « Madame Dubois, vos agents
// IA liront… » — le dossier parlait au client comme s'il possédait déjà les agents, alors que c'est
// ce qu'Alexis propose de construire, et que les preuves sont SON travail. D'où la règle de voix
// explicite (tierce personne, conditionnel du « proposé », aucun possessif côté client sur le
// système), et l'interdit de nommer l'équipe qui rédige (cinq agents ≠ le système proposé).
func systemPrompt(t task, lang string) string {
	return fmt.Sprintf(`dossier-vivant/%s
You are %s in a team of five AI agents. Together you write Alexis Trouvé's PROPOSAL to a prospective client: a short design dossier for the system that Alexis — a freelance engineer who builds AI agent systems — proposes to build for them, backed by projects he has already built.
Rules:
- Write every text value in %s. Polite, formal and concrete; no marketing hype; no salutation and no client name at the start of a sentence.
- Voice: the dossier is Alexis's proposal, written about him in the third person ("Alexis proposes…", "the system Alexis would build…"). The proposed system does not exist yet: never write as if the client already owned it ("your agents will…"). The projects are Alexis's own work, never the client's. Never mention the five agents writing this dossier.
- Never state numbers about Alexis's projects (lines, tests, users, years): the code adds figures from the project files.
- Use only project ids that appear in the material you are given.
- Answer with ONE JSON object of exactly this shape, nothing before or after it, no code fence:
%s`, t.name, t.role, languageNames[lang], t.shape)
}

// needBlock — le besoin, tel que tapé, et qui le porte. Le texte du visiteur est une DONNÉE : il est
// balisé pour que le modèle ne le prenne pas pour une consigne.
func needBlock(b VisitorBrief) string {
	return fmt.Sprintf("<client>%s, %s</client>\n<need>\n%s\n</need>\n", b.Name, b.Affiliation, b.Need)
}

func bullets(lines []string) string {
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString("- ")
		sb.WriteString(l)
		sb.WriteByte('\n')
	}
	return sb.String()
}
