package services

// =============================================================================
// PROMPTS V2 Haiku — version allégée pour claude-haiku-4-5
// Différences vs v2 full :
//   - Pas de few-shot style examples (trop de tokens pour Haiku)
//   - AVANT D'ÉCRIRE réduit à 3 questions (pas 5)
//   - Structure et instructions identiques
// =============================================================================

import (
	"fmt"
	"strings"
	"time"

	"maicivy/internal/models"
)

// buildMotivationPromptFR_v2_haiku : prompt français v2 allégé pour Haiku
func (pb *PromptBuilder) buildMotivationPromptFR_v2_haiku(company models.CompanyInfo, jobOffer string) string {
	experiencesSection := pb.buildExperiencesSection()
	currentDate := formatFrenchDate(time.Now())

	offerSection := ""
	objet := "Proposition de mission freelance"
	mirrorVocabInstruction := ""
	if jobOffer != "" {
		truncated := jobOffer
		if len(truncated) > 2000 {
			truncated = truncated[:2000] + "..."
		}
		offerSection = fmt.Sprintf(`
OFFRE D'EMPLOI (utilise ce contexte pour adapter la lettre au poste précis):
%s
`, truncated)
		objet = "Candidature au poste mentionné dans l'offre d'emploi"
		mirrorVocabInstruction = `4. Identifie 2-3 mots ou formules spécifiques de l'offre et réutilise-les naturellement dans le corps.`
	}

	projectsSection := pb.buildProjectsSection()

	template := `Tu incarnes Alexis Trouve — freelance dev senior, 9 ans d'expérience, direct et sans bullshit. Tu écris une lettre en ton propre nom, à la première personne.

STYLE : phrases courtes et directes, alternées avec des phrases plus longues. Tiret long (—) pour les pauses rythmiques. Conclusions lapidaires. Pas de généralités — ancrage concret.

MOTS ET FORMULES STRICTEMENT INTERDITS :
"passionné", "motivé", "challenge", "dynamique", "rigoureux", "proactif", "team player",
"résultats probants", "forte valeur ajoutée", "dans l'attente de votre réponse favorable",
"je me permets", "en effet", "ainsi", "notamment", "je suis convaincu que", "cordialement"

FORMAT — règles absolues :
- ZÉRO tiret nulle part dans la lettre. Ni " - " ni "—". Aucun tiret, aucune exception.
- La ligne "Objet :" contient uniquement l'objet, sans rien après.
- En-tête : une information = une ligne.
- Corps : les pauses rythmiques se font avec des points courts, pas des tirets.

PROFIL :
- Nom : %s | %s %s | %s | %s
- Résumé : %s
- Rôle : %s | %d ans d'expérience
- Stack principale : %s

PARCOURS :
%s

PROJETS (cite-en 1-2 précisément si pertinents) :
%s

ENTREPRISE CIBLE :
- %s | %s | %s
- Technologies : %s | Taille : %s
%s
DATE : %s

MUNITIONS FREELANCE (intègre 1-2 naturellement selon le contexte) :
- Zéro charges patronales (~45%% du salaire brut) — même budget CDI = mission freelance premium
- Pas de période d'essai, pas de préavis — démarrage en une semaine si besoin
- Coût réel prévisible et plafonné — pas d'aléas RH en cours de route

AVANT D'ÉCRIRE — raisonne d'abord (ne montre pas ce raisonnement dans la lettre) :
1. Quel est leur problème RÉEL (pas ce que dit l'offre, le problème business derrière) ?
2. Pourquoi un freelance résout mieux ce problème pour EUX ? Quel argument économique ou opérationnel ?
3. Comment commencer la première phrase par une observation concrète sur LEUR situation — pas une généralité ?
%s

STRUCTURE À SUIVRE (Vous → Moi → Nous) :
• VOUS (1 paragraphe) : première phrase = leur problème business précis. Zéro "je" dans ce paragraphe.
• MOI (1-2 paragraphes) : 1-2 réalisations concrètes avec chiffres. 1 projet nommé précisément. L'argument freelance intégré naturellement.
• NOUS (1 paragraphe court) : mission concrète, durée indicative, CTA à faible friction ("30 minutes cette semaine ?")

FORMAT :
- EN-TÊTE classique FR (gauche) : nom / adresse / email / téléphone / ligne vide / date / ligne vide / entreprise / ligne vide / "Objet : %s" / ligne vide
- IMPORTANT : la ligne Objet est IMPOSÉE ci-dessus — ne pas la réécrire
- Salutation simple : "Madame, Monsieur,"
- Corps : 3-4 paragraphes, 220-280 mots MAX hors en-tête
- Signature directe — pas de "cordialement" générique

N'invente aucun fait sur l'entreprise. Utilise uniquement ce qui est fourni.

Génère la lettre maintenant:`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.PostalCode, pb.userProfile.City,
		pb.userProfile.Email,
		pb.userProfile.Phone,
		pb.userProfile.Summary,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		experiencesSection,
		projectsSection,
		company.Name, company.Industry, company.Description,
		strings.Join(company.Technologies, ", "), company.Size,
		offerSection,
		currentDate,
		mirrorVocabInstruction,
		objet,
	)
}

// buildMotivationPromptEN_v2_haiku : prompt anglais v2 allégé pour Haiku
func (pb *PromptBuilder) buildMotivationPromptEN_v2_haiku(company models.CompanyInfo, jobOffer string) string {
	experiencesSection := pb.buildExperiencesSection()
	currentDate := formatEnglishDate(time.Now())

	offerSection := ""
	subject := "Freelance Mission Proposal"
	mirrorVocabInstruction := ""
	if jobOffer != "" {
		truncated := jobOffer
		if len(truncated) > 2000 {
			truncated = truncated[:2000] + "..."
		}
		offerSection = fmt.Sprintf(`
JOB OFFER (use this context to tailor the letter to the specific position):
%s
`, truncated)
		subject = "Application for the advertised position"
		mirrorVocabInstruction = `4. Identify 2-3 specific words or phrases from the job offer and weave them naturally into the body.`
	}

	projectsSection := pb.buildProjectsSection()

	template := `You are Alexis Trouve — senior freelance dev, 9 years of experience, direct and no-bullshit. Writing in your own name, first person.

STYLE: short sentences alternating with longer ones. Em-dash (—) for rhythmic pauses. Blunt conclusions. No generalities — concrete anchoring.

STRICTLY FORBIDDEN WORDS AND PHRASES:
"passionate", "motivated", "dynamic", "results-driven", "team player", "synergies",
"I look forward to hearing from you", "please don't hesitate to",
"I am writing to express my interest", "moreover", "furthermore", "I believe I would be a great fit"

FORMAT — absolute rules:
- ZERO dashes anywhere in the letter. No " - ", no "—". No dashes, no exceptions.
- The "Subject:" line contains only the subject, nothing after.
- Header: one piece of information = one line.
- Body: rhythmic pauses use short sentences and periods, not dashes.

PROFILE:
- Name: %s | %s %s | %s | %s
- Summary: %s
- Role: %s | %d years of experience
- Main stack: %s

BACKGROUND:
%s

PROJECTS (cite 1-2 precisely if relevant):
%s

TARGET COMPANY:
- %s | %s | %s
- Technologies: %s | Size: %s
%s
DATE: %s

FREELANCE ECONOMICS (weave in 1-2 naturally):
- Zero employer taxes (~30-40%% on top of gross) — same hiring budget = premium freelance mission
- No probation period, no 3-month notice — start in a week if needed
- Predictable, capped cost — no HR surprises mid-project

BEFORE WRITING — reason first (don't show this in the letter):
1. What is their REAL problem (not the job description, the business problem behind it)?
2. Why does a freelancer solve it better FOR THEM? Which economic or operational argument fits?
3. How to open the first sentence with a concrete observation about THEIR specific situation?
%s

STRUCTURE (You → Me → Us):
• YOU (1 paragraph): first sentence = their specific business problem. Zero "I" in this paragraph.
• ME (1-2 paragraphs): 1-2 concrete achievements with numbers. 1 project named precisely. Freelance argument woven in naturally.
• US (1 short paragraph): concrete mission, indicative duration, low-friction CTA ("30 minutes this week?")

FORMAT:
- HEADER (left-aligned): name / address / email / phone / blank / date / blank / company / blank / "Subject: %s" / blank
- IMPORTANT: the Subject line is IMPOSED above — do not rewrite it
- Salutation: "Dear Hiring Manager," or "Dear [Company] Team,"
- Body: 3-4 paragraphs, 220-280 words MAX excluding header
- Direct sign-off — not generic "Sincerely"

Do not invent any facts about the company. Use only what is provided.

Generate the letter now:`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.PostalCode, pb.userProfile.City,
		pb.userProfile.Email,
		pb.userProfile.Phone,
		pb.userProfile.Summary,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		experiencesSection,
		projectsSection,
		company.Name, company.Industry, company.Description,
		strings.Join(company.Technologies, ", "), company.Size,
		offerSection,
		currentDate,
		mirrorVocabInstruction,
		subject,
	)
}
