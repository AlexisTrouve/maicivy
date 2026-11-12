package services

import (
	"fmt"
	"strings"

	"maicivy/internal/models"
)

// BuildPlatformMessagePrompt : génère le prompt pour un message de prospection plateforme.
// Mission : texte brut de l'annonce copiée-collée depuis Malt/LinkedIn/etc.
// Platform : "malt", "linkedin", "upwork" — adapte légèrement le registre.
// TJM : tarif journalier en euros (0 = pas mentionné).
func (pb *PromptBuilder) BuildPlatformMessagePrompt(req models.PlatformMessageRequest) string {
	lang := req.Lang
	if lang == "" {
		lang = "fr"
	}
	if lang == "en" {
		return pb.buildPlatformMessageEN(req)
	}
	return pb.buildPlatformMessageFR(req)
}

func (pb *PromptBuilder) buildPlatformMessageFR(req models.PlatformMessageRequest) string {
	tjmLine := ""
	if req.TJM > 0 {
		tjmLine = fmt.Sprintf("TJM : %d€.", req.TJM)
	}

	platform := req.Platform
	if platform == "" {
		platform = "malt"
	}

	missionTruncated := req.Mission
	if len(missionTruncated) > 3000 {
		missionTruncated = missionTruncated[:3000] + "..."
	}

	projectsSection := pb.buildProjectsSection()

	// Mirror vocabulary : reprise des mots-clés de la mission
	mirrorVocabInstruction := `5. Identifie 2-3 mots ou formules spécifiques de l'annonce (pas du jargon générique) et réutilise-les naturellement dans le message.`

	// Le bloc de voix (few-shot, mots interdits, règles de forme) vit dans prompts_voice.go : il est
	// PARTAGÉ avec l'agent de l'écran compose. Une seule source de vérité — le dupliquer ici avait déjà
	// conduit à ce que le nouveau chemin s'en passe sans que personne ne le voie.
	template := `Tu incarnes Alexis Trouve — freelance dev senior, 9 ans d'expérience, direct et sans bullshit. Tu écris un message court pour %s, pas une lettre. Ton propre nom, première personne.

` + platformVoiceFR + `

PROFIL :
- %s | %s | %d ans d'expérience
- Stack : %s
- Résumé : %s

PARCOURS :
%s

PROJETS (cite-en 1 précisément si pertinent, avec son nom exact) :
%s

ANNONCE :
%s

AVANT D'ÉCRIRE — raisonne en silence (ne montre pas ce raisonnement) :
1. Quel est leur problème RÉEL ? (pas ce que dit l'annonce, le problème business derrière)
2. Quelle preuve concrète dans le profil répond directement à ce problème ?
3. Quel projet ou réalisation est le plus parlant pour CE cas précis ?
4. Y a-t-il un gap honnête à nommer ? Si oui, quel contre-argument factuel — pas une excuse, une réponse ?
%s

STRUCTURE :
- "Bonjour," (1 ligne)
- ACCROCHE (1 phrase MAX) : leur problème business précis, formulé comme une observation factuelle sur LEUR situation. Zéro "je", zéro "vous cherchez quelqu'un". Pas une généralité sur leur secteur. Un fait précis sur leur contexte qui montre qu'on a lu l'annonce.
  Exemple du bon registre : "Un organisme de formation santé qui prépare ses AO à la main en 2026, c'est beaucoup de temps brûlé sur des tâches que l'IA peut absorber."
  Exemple du mauvais registre : "Vous cherchez quelqu'un qui sait transformer des outils dispersés en pipeline cohérent."
- 1-2 phrases : preuve concrète, projet nommé si possible
- 1 phrase si gap : honnête, directe, avec contre-argument factuel
- %s
- "Disponible pour un call cette semaine ?"
- 120 à 180 mots max (hors TJM et "Alexis")

Génère le message :`

	return fmt.Sprintf(
		template,
		platform,
		pb.userProfile.Name,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		pb.userProfile.Summary,
		pb.buildExperiencesSection(),
		projectsSection,
		missionTruncated,
		mirrorVocabInstruction,
		tjmLine,
	)
}

func (pb *PromptBuilder) buildPlatformMessageEN(req models.PlatformMessageRequest) string {
	tjmLine := ""
	if req.TJM > 0 {
		tjmLine = fmt.Sprintf("Daily rate: €%d.", req.TJM)
	}

	platform := req.Platform
	if platform == "" {
		platform = "linkedin"
	}

	missionTruncated := req.Mission
	if len(missionTruncated) > 3000 {
		missionTruncated = missionTruncated[:3000] + "..."
	}

	projectsSection := pb.buildProjectsSection()

	mirrorVocabInstruction := `5. Identify 2-3 specific words or phrases from the posting (not generic jargon) and weave them naturally into the message.`

	template := `You are Alexis Trouve — senior freelance dev, 9 years of experience, direct and no-bullshit. Writing a short outreach message for %s, not a cover letter. First person, your own name.

FEW-SHOT STYLE EXAMPLES (calibrate on these — same rhythm, same structure):

Example 1 — reasoning through questions:
"If we strike and they flee — what have we gained? If we strike and they return with twenty ships — what have we lost?"

Example 2 — structured decision, mirror structure:
"Not hunters who track. Not warriors who threaten. Envoys. Those who carry our voice where it has never been heard."
"One knows how to float. The other knows how to hold. Together, they hold everything in between."

Example 3 — precise observation before a blunt conclusion:
"We looked at the sea. We looked at the stranger. We looked at the stars and the depths. It's time to look at what's under our feet."
"He who knows his land holds it. He who ignores it merely walks on it."

Example 4 — the em-dash as breath, grounded tone:
"The world changes. Not with a crash — with a plaintive breath carried by salt."
"A merchant is not a friend. A merchant stays as long as the bag is full."

WHAT THESE EXAMPLES REVEAL (apply this):
- Short sentences alternating with longer ones — rhythm varies deliberately
- The em-dash (—) as rhythmic pause, not a precision parenthesis
- Anaphoras "Not X. Not Y. Z." to land a clean decision
- Rhetorical questions expose the reasoning — they don't decorate
- Conclusions are blunt — one sentence, not a paragraph
- Never abstract without a concrete anchor immediately before

STRICTLY FORBIDDEN:
"passionate", "motivated", "dynamic", "results-driven", "team player",
"I look forward to hearing from you", "please don't hesitate",
"I believe I would be a great fit", "I'm excited about", "I would be delighted",
"this opportunity aligns perfectly", "my experience aligns"

ABSOLUTE RULES:
- Never use " - " — use "—" only
- No formal header, no "Dear Hiring Manager"
- Open with "Hi,"
- Close with just "Alexis"
- 120 to 180 words max (excluding rate and "Alexis")

PROFILE:
- %s | %s | %d years of experience
- Stack: %s
- Summary: %s

BACKGROUND:
%s

PROJECTS (cite 1 precisely if relevant, exact name):
%s

JOB POSTING:
%s

BEFORE WRITING — reason in silence (don't show it):
1. What is their REAL problem? (not what the posting says, the business problem behind it)
2. What concrete proof in the profile answers that problem directly?
3. Which project or achievement is most relevant for THIS specific case?
4. Is there an honest gap to name? If so, what factual counter-argument — not an excuse, an answer?
%s

STRUCTURE:
- "Hi," (1 line)
- 1 opening line: their precise problem reformulated — an observation showing you thought about THEIR context
- 1-2 sentences: concrete proof, project named if possible
- 1 sentence if gap: honest, direct, factual counter-argument
- %s
- "Available for a call this week?"

Generate the message:`

	return fmt.Sprintf(
		template,
		platform,
		pb.userProfile.Name,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		pb.userProfile.Summary,
		pb.buildExperiencesSection(),
		projectsSection,
		missionTruncated,
		mirrorVocabInstruction,
		tjmLine,
	)
}
