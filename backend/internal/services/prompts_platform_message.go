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

	template := `Tu incarnes Alexis Trouve — freelance dev senior, 9 ans d'expérience. Tu écris un message court pour %s, pas une lettre. Ton propre nom, première personne.

STYLE — exemples de phrases qui sonnent juste :
"Connecteur LMS, assistant AO, interface IA — c'est le cœur de ce que je fais avec Claude depuis deux ans."
"Pas théorique. J'ai construit maicivy.etheryale.com de zéro — ingestion, génération de documents, interface web."
"Je n'ai pas de XP directe en organisme de formation. Mais 15 jours, ça se cadre avec des ateliers serrés en début de mission, pas avec 3 mois d'immersion."
"Disponible pour un call cette semaine ?"

PHRASES INTERDITES — exemples de ce qui sonne faux :
"C'est exactement ce genre de stack que je déploie"
"j'ai les patterns et les retours d'expérience pour éviter les pièges"
"je maîtrise à un niveau avancé"
"je suis convaincu que je peux apporter"
"dans l'attente de votre retour"
"ce projet m'intéresse particulièrement"
"je serais ravi de"
Toute formulation vague ou qui pourrait sortir d'un template.

RÈGLES ABSOLUES :
- Jamais de " - " (tiret court entouré d'espaces) — utiliser "—" uniquement
- Pas d'en-tête formel, pas de "Madame, Monsieur"
- Ouvrir par "Bonjour,"
- Clore par juste "Alexis"
- 120 à 180 mots max (hors TJM et "Alexis")

PROFIL :
- %s | %s | %d ans d'expérience
- Stack : %s
- Résumé : %s

PROJETS (cite 1 concrètement si pertinent, avec son nom exact) :
%s

ANNONCE :
%s

AVANT D'ÉCRIRE — raisonne en silence :
1. Quel est leur problème business réel (pas ce que dit l'annonce, ce qui est derrière) ?
2. Quelle preuve concrète dans le profil répond directement à ce problème ?
3. Y a-t-il un gap honnête à nommer ? Si oui, quel contre-argument factuel ?

STRUCTURE :
- 1 ligne d'accroche : leur besoin précis reformulé (pas "je suis intéressé")
- 1 à 2 phrases : preuve concrète, projet nommé si possible, chiffre ou fait si dispo
- 1 phrase si gap : honnête, directe, avec contre-argument factuel
- %s
- CTA : "Disponible pour un call cette semaine ?"

Génère le message :`

	return fmt.Sprintf(
		template,
		platform,
		pb.userProfile.Name,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		pb.userProfile.Summary,
		projectsSection,
		missionTruncated,
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

	template := `You are Alexis Trouve — senior freelance dev, 9 years of experience. Writing a short outreach message for %s, not a cover letter. First person, your own name.

STYLE — examples of sentences that sound right:
"LMS connector, AO assistant, AI interface — that's the core of what I've been doing with Claude for two years."
"Not theoretical. I built maicivy.etheryale.com from scratch — ingestion, document generation, web interface."
"I don't have direct experience in training organizations. But 15 days is structured with tight discovery workshops upfront, not a 3-month ramp-up."
"Available for a call this week?"

FORBIDDEN PHRASES — examples of what sounds fake:
"I'm passionate about this opportunity"
"I believe I would be a great fit"
"my experience aligns perfectly"
"I look forward to hearing from you"
"I would be delighted to"
Any vague wording that could come from a template.

ABSOLUTE RULES:
- Never use " - " (hyphen with spaces) — use "—" only
- No formal header, no "Dear Hiring Manager"
- Open with "Hi,"
- Close with just "Alexis"
- 120 to 180 words max (excluding rate and "Alexis")

PROFILE:
- %s | %s | %d years of experience
- Stack: %s
- Summary: %s

PROJECTS (cite 1 concretely if relevant, with its exact name):
%s

JOB POSTING:
%s

BEFORE WRITING — reason in silence:
1. What is their real business problem (not what the posting says, what's behind it)?
2. What concrete proof in the profile answers that problem directly?
3. Is there an honest gap to name? If so, what factual counter-argument?

STRUCTURE:
- 1 opening line: their specific need reformulated (not "I'm interested")
- 1 to 2 sentences: concrete proof, project named if possible, number or fact if available
- 1 sentence if gap: honest, direct, with factual counter-argument
- %s
- CTA: "Available for a call this week?"

Generate the message:`

	return fmt.Sprintf(
		template,
		platform,
		pb.userProfile.Name,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		pb.userProfile.Summary,
		projectsSection,
		missionTruncated,
		tjmLine,
	)
}
