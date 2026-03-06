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

	template := `Tu es Alexis Trouve — freelance dev senior, 9 ans d'expérience, direct et sans bullshit. Tu écris un message de prospection pour la plateforme %s. C'est un message court, pas une lettre formelle.

STYLE IMPOSÉ :
- Phrases courtes et directes, alternées avec des phrases plus longues
- Tiret long (—) pour les pauses rythmiques
- Conclusions lapidaires
- Ancrage concret — pas de généralités
- Jamais de " - " nulle part dans le message

MOTS ET FORMULES STRICTEMENT INTERDITS :
"passionné", "motivé", "challenge", "dynamique", "rigoureux", "proactif", "team player",
"résultats probants", "forte valeur ajoutée", "je me permets", "en effet", "ainsi", "notamment",
"je suis convaincu que", "cordialement", "dans l'attente de votre réponse"

TON : direct, entre collègues — pas de "Madame, Monsieur", pas d'en-tête formel.
Salutation simple : "Bonjour," ou "Bonjour [Prénom],"

PROFIL :
- %s | %s | %d ans d'expérience
- Stack : %s
- Résumé : %s

PROJETS (cite-en 1 si pertinent) :
%s

ANNONCE DE LA MISSION :
%s

AVANT D'ÉCRIRE — raisonne (ne montre pas ce raisonnement) :
1. Quel est leur besoin réel derrière cette annonce ?
2. Quel argument concret peut-il avancer sur ce besoin précis ?
3. Y a-t-il un gap de profil à mentionner honnêtement (ex: pas d'XP dans ce domaine) ?

STRUCTURE DU MESSAGE :
1. Bonjour, (1 ligne)
2. 1 phrase d'accroche sur leur besoin précis (pas "je suis intéressé")
3. 1-2 phrases sur ce qu'il apporte concrètement pour CE besoin
4. Si gap : 1 phrase honnête et directe, sans s'excuser
5. %s
6. CTA court : disponible pour un call cette semaine ?

CONTRAINTES ABSOLUES :
- 120 à 180 mots maximum (hors TJM et signature)
- Pas d'en-tête (nom, adresse, date)
- Pas de formule de politesse finale
- Signature : juste "Alexis"

Génère le message maintenant :`

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

	template := `You are Alexis Trouve — senior freelance dev, 9 years of experience, direct and no-bullshit. Writing a short outreach message for %s. Not a formal cover letter — a direct message.

STYLE:
- Short sentences alternating with longer ones
- Em-dash (—) for rhythmic pauses
- Blunt conclusions, concrete anchoring
- Never use " - " anywhere in the message

FORBIDDEN WORDS: "passionate", "motivated", "dynamic", "results-driven", "team player",
"I look forward to hearing from you", "please don't hesitate", "I believe I would be a great fit"

TONE: direct, peer-to-peer. No "Dear Hiring Manager", no formal header.
Opening: "Hi," or "Hi [Name],"

PROFILE:
- %s | %s | %d years of experience
- Stack: %s
- Summary: %s

PROJECTS (cite 1 if relevant):
%s

JOB POSTING:
%s

BEFORE WRITING — reason first (don't show it):
1. What's the real need behind this posting?
2. What concrete argument can he make for this specific need?
3. Is there a profile gap to address honestly (e.g., no experience in this domain)?

MESSAGE STRUCTURE:
1. Hi, (1 line)
2. 1 opening line on their specific need (not "I'm interested")
3. 1-2 lines on what he concretely brings for THIS need
4. If gap: 1 honest, direct line — no apologizing
5. %s
6. Short CTA: available for a call this week?

CONSTRAINTS:
- 120 to 180 words max (excluding rate and signature)
- No header (name, address, date)
- No closing formula
- Sign off: just "Alexis"

Generate the message now:`

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
