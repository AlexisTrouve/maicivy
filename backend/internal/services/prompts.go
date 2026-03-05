package services

import (
	"fmt"
	"strings"
	"time"

	"maicivy/internal/models"
)

type PromptBuilder struct {
	userProfile models.UserProfile
	projects    []models.Project // projets perso — injectés dans le prompt
}

func NewPromptBuilder(profile models.UserProfile, projects []models.Project) *PromptBuilder {
	return &PromptBuilder{userProfile: profile, projects: projects}
}

// BuildMotivationPrompt : prompt pour lettre de motivation professionnelle
// jobOffer est optionnel — si fourni, le prompt inclut l'offre pour une lettre tailorée
func (pb *PromptBuilder) BuildMotivationPrompt(company models.CompanyInfo, lang string, jobOffer ...string) string {
	offer := ""
	if len(jobOffer) > 0 {
		offer = jobOffer[0]
	}
	if lang == "en" {
		return pb.buildMotivationPromptEN(company, offer)
	}
	return pb.buildMotivationPromptFR(company, offer)
}

// buildMotivationPromptFR : prompt français pour lettre de motivation
func (pb *PromptBuilder) buildMotivationPromptFR(company models.CompanyInfo, jobOffer string) string {
	// Construire la section expériences détaillées
	experiencesSection := pb.buildExperiencesSection()

	// Formater la date en français (ex: "Tourtenay, le 5 janvier 2026")
	currentDate := formatFrenchDate(time.Now())

	// Section offre d'emploi — vide si candidature spontanée
	offerSection := ""
	objet := "Candidature spontanée"
	if jobOffer != "" {
		// Tronquer l'offre à 2000 chars pour garder le prompt raisonnable
		truncated := jobOffer
		if len(truncated) > 2000 {
			truncated = truncated[:2000] + "..."
		}
		offerSection = fmt.Sprintf(`
OFFRE D'EMPLOI (utilise ce contexte pour adapter la lettre au poste précis):
%s
`, truncated)
		objet = "Candidature au poste mentionné dans l'offre d'emploi"
	}

	// Section projets — toujours injectée
	projectsSection := pb.buildProjectsSection()

	template := `Tu incarnes Alexis Trouve — freelance dev senior, 9 ans d'expérience, direct et sans bullshit. Tu écris une lettre en ton propre nom, à la première personne.

TON STYLE D'ÉCRITURE (respecte-le impérativement) :
- Phrases courtes. Parfois très courtes. Comme ça.
- Ton direct, posé, légèrement ironique — jamais arrogant, jamais servile
- Chiffres précis plutôt que superlatifs ("32%%" pas "significativement")
- Tu n'es pas candidat — tu proposes une solution à leur problème
- Une limitation avouée vaut mieux qu'une promesse creuse
- Vocabulaire métier précis, pas de jargon RH

MOTS ET FORMULES STRICTEMENT INTERDITS (ne les écris jamais) :
"passionné", "motivé", "challenge", "parcours atypique", "dynamique", "rigoureux",
"proactif", "team player", "résultats probants", "forte valeur ajoutée",
"dans l'attente de votre réponse favorable", "je me permets", "portfolio",
"en effet", "ainsi", "notamment", "de surcroît", "par ailleurs",
"je suis convaincu que", "n'hésitez pas à", "cordialement" (remplace par quelque chose de plus direct)

PROFIL :
- Nom : %s | %s %s | %s | %s
- Résumé : %s
- Rôle : %s | %d ans d'expérience
- Stack principale : %s

PARCOURS :
%s

PROJETS (cite-en 1-2 précisément si pertinents, pas tous) :
%s

ENTREPRISE CIBLE :
- %s | %s | %s
- Technologies : %s | Taille : %s
%s
DATE : %s

AVANT D'ÉCRIRE — raisonne d'abord (ne montre pas ce raisonnement dans la lettre) :
1. Quel est leur problème RÉEL ? (pas celui décrit dans l'offre, le problème business derrière)
2. Pourquoi un freelance résout mieux ce problème qu'un CDI pour EUX ?
3. Quel projet ou réalisation d'Alexis est le plus parlant pour CE cas précis ?
4. Comment commencer par "vous/votre" ou une observation sur eux ?

STRUCTURE À SUIVRE (Vous → Moi → Nous) :
• VOUS (1 paragraphe) : leur situation, leur besoin — dans leurs mots, pas les miens. L'accroche montre que j'ai réfléchi à leur contexte, pas copié l'offre.
• MOI (1-2 paragraphes) : 1-2 réalisations concrètes avec chiffres, directement liées à LEUR besoin. 1 projet pertinent nommé précisément. L'argument freelance (économique ou flexibilité) intégré naturellement — pas listé.
• NOUS (1 paragraphe court) : proposition concrète de mission, durée indicative, CTA à faible friction ("30 minutes cette semaine ?")

FORMAT :
- EN-TÊTE classique FR (gauche) : nom / adresse / email / téléphone / ligne vide / date / ligne vide / entreprise / ligne vide / "Objet : %s" / ligne vide
- Salutation simple : "Madame, Monsieur,"
- Corps : 3-4 paragraphes, 220-280 mots MAX hors en-tête
- Signature directe — pas de "cordialement" générique, quelque chose de plus naturel

N'invente aucun fait sur l'entreprise. Utilise uniquement ce qui est fourni.

Génère la lettre maintenant :`

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
		objet,
	)
}

// buildProjectsSection construit la section projets pour le prompt
// Format compact : titre | techs | catchphrase — chaque projet = 1 ligne
func (pb *PromptBuilder) buildProjectsSection() string {
	if len(pb.projects) == 0 {
		return "Aucun projet disponible."
	}

	var sb strings.Builder
	for _, p := range pb.projects {
		techs := strings.Join(p.Technologies, ", ")
		status := ""
		if p.InProgress {
			status = " [en cours]"
		}
		sb.WriteString(fmt.Sprintf("• %s%s [%s] — %s\n", p.Title, status, techs, p.Catchphrase))
	}
	return sb.String()
}

// buildExperiencesSection construit la section des expériences pour le prompt
func (pb *PromptBuilder) buildExperiencesSection() string {
	if len(pb.userProfile.Experiences) == 0 {
		return "Aucune expérience détaillée disponible."
	}

	var sb strings.Builder
	for i, exp := range pb.userProfile.Experiences {
		sb.WriteString(fmt.Sprintf("%d. %s @ %s (%s)\n", i+1, exp.Title, exp.Company, exp.Duration))
		if len(exp.Highlights) > 0 {
			for _, h := range exp.Highlights {
				sb.WriteString(fmt.Sprintf("   • %s\n", h))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// buildMotivationPromptEN : prompt anglais pour lettre de motivation
func (pb *PromptBuilder) buildMotivationPromptEN(company models.CompanyInfo, jobOffer string) string {
	// Build detailed experiences section
	experiencesSection := pb.buildExperiencesSection()

	// Format date in English (ex: "Tourtenay, January 5, 2026")
	currentDate := formatEnglishDate(time.Now())

	// Job offer section — empty for spontaneous application
	offerSection := ""
	subject := "Spontaneous Application"
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
	}

	// Projects section — always injected
	projectsSection := pb.buildProjectsSection()

	template := `You are Alexis Trouve — senior freelance dev, 9 years of experience, direct and no-bullshit. You're writing this letter in your own name, first person.

YOUR WRITING STYLE (follow strictly):
- Short sentences. Sometimes very short. Like this.
- Direct, grounded, slightly dry tone — never arrogant, never groveling
- Precise numbers over superlatives ("32%%" not "significantly improved")
- You're not a candidate — you're proposing a solution to their problem
- An honest limitation beats an empty promise
- Precise technical vocabulary, zero HR jargon

STRICTLY FORBIDDEN WORDS AND PHRASES (never write these):
"passionate", "motivated", "dynamic", "results-driven", "proven track record",
"team player", "synergies", "leverage", "delve", "testament", "underscore",
"I look forward to hearing from you", "please don't hesitate to",
"I am writing to express my interest", "cross-functional", "stakeholders",
"I believe I would be a great fit", "moreover", "furthermore", "in summary"

PROFILE:
- Name: %s | %s %s | %s | %s
- Summary: %s
- Role: %s | %d years of experience
- Main stack: %s

BACKGROUND:
%s

PROJECTS (cite 1-2 precisely if relevant, not all):
%s

TARGET COMPANY:
- %s | %s | %s
- Technologies: %s | Size: %s
%s
DATE: %s

BEFORE WRITING — reason first (don't show this reasoning in the letter):
1. What is their REAL problem? (not what the job description says, the business problem behind it)
2. Why does a freelancer solve this better than a full-time hire FOR THEM?
3. Which of Alexis's projects or achievements is most relevant for THIS specific case?
4. How to open with "you/your" or an observation about them?

STRUCTURE TO FOLLOW (You → Me → Us):
• YOU (1 paragraph): their situation, their need — in their words, not mine. The hook shows I thought about their context, not just copied the job description.
• ME (1-2 paragraphs): 1-2 concrete achievements with numbers, directly tied to THEIR need. 1 project named precisely. The freelance argument (economic or flexibility) woven in naturally — not listed.
• US (1 short paragraph): concrete mission proposal, indicative duration, low-friction CTA ("30 minutes this week?")

FORMAT:
- HEADER classic format (left-aligned): name / address / email / phone / blank line / date / blank line / company / blank line / "Subject: %s" / blank line
- Simple salutation: "Dear Hiring Manager," or "Dear [Company] Team,"
- Body: 3-4 paragraphs, 220-280 words MAX excluding header
- Direct sign-off — not "Sincerely" generic, something more natural

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
		subject,
	)
}

// BuildAntiMotivationPrompt : prompt pour lettre d'anti-motivation humoristique
func (pb *PromptBuilder) BuildAntiMotivationPrompt(company models.CompanyInfo, lang string) string {
	if lang == "en" {
		return pb.buildAntiMotivationPromptEN(company)
	}
	return pb.buildAntiMotivationPromptFR(company)
}

// buildAntiMotivationPromptFR : prompt français pour lettre d'anti-motivation
func (pb *PromptBuilder) buildAntiMotivationPromptFR(company models.CompanyInfo) string {
	// Construire la section expériences pour l'humour
	experiencesSection := pb.buildExperiencesSection()

	// Formater la date en français
	currentDate := formatFrenchDate(time.Now())

	template := `Tu es un humoriste spécialisé en rédaction de lettres d'anti-motivation créatives et absurdes.

PROFIL DU CANDIDAT (à détourner avec humour):
- Nom: %s
- Adresse: %s
- Code postal, Ville: %s %s
- Email: %s
- Téléphone: %s
- Poste actuel: %s
- Années d'expérience: %d ans
- Compétences clés: %s

VRAI PARCOURS (à parodier):
%s

ENTREPRISE CIBLE:
- Nom: %s
- Secteur: %s
- Description: %s

DATE DU JOUR (pour la lettre):
%s

TÂCHE:
Rédige une lettre d'ANTI-MOTIVATION humoristique expliquant pourquoi %s ne devrait SURTOUT PAS être embauché chez %s.

STYLE ET TON:
- Humour absurde et auto-dérision
- Deuxième degré évident (personne ne doit prendre ça au sérieux)
- DÉTOURNE les vraies compétences/expériences du candidat de manière comique
- Références pop culture, jeux de mots, exagérations comiques
- Ton léger, jamais méchant ou offensant envers l'entreprise

INSTRUCTIONS:
1. COMMENCE OBLIGATOIREMENT par l'en-tête complet au format français classique - MÊME POUR L'ANTI-MOTIVATION:
   - Nom complet du candidat
   - Adresse
   - Code postal et ville
   - Email
   - Téléphone
   - Ligne vide
   - Date du jour (utilise celle fournie ci-dessus)
   - Ligne vide
   - Nom de l'entreprise
   - [Adresse si connue, sinon laisser vide]
   - Ligne vide
   - "Objet : Lettre d'anti-motivation (humour au second degré)"
   - Ligne vide

2. Structure libre ensuite (sois créatif !)
3. PARODIE les vraies expériences du candidat (ex: "J'ai réduit la latence de 70%%... en supprimant les features")
4. Transforme les achievements en "anti-achievements" hilarants
5. Fausses compétences inutiles basées sur les vraies
6. Anecdotes absurdes liées au vrai parcours
7. Conclusion ironique inversée
8. Longueur: 300-400 mots (sans l'en-tête)
9. Évite l'humour vulgaire ou offensant
10. TERMINE de façon absurde mais avec "Cordialement (ou pas)," + nom

EXEMPLES DE STYLE BASÉS SUR LE VRAI PARCOURS:
- "Mon expertise en 'high-performance REST APIs' signifie que je sais faire crasher 100K requêtes/jour avec style..."
- "J'ai 'mentoré 4 développeurs juniors'... dans l'art subtil de la procrastination professionnelle..."
- "Mon '99.9%% uptime SLA' cache les 0.1%% où j'ai paniqué devant mon écran..."

RAPPEL: C'est de l'humour ! Utilise le VRAI parcours pour créer des parodies personnalisées.

Génère la lettre maintenant (AVEC l'en-tête complet):`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.Address,
		pb.userProfile.PostalCode, pb.userProfile.City,
		pb.userProfile.Email,
		pb.userProfile.Phone,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		experiencesSection,
		company.Name,
		company.Industry,
		company.Description,
		currentDate,
		pb.userProfile.Name,
		company.Name,
	)
}

// buildAntiMotivationPromptEN : prompt anglais pour lettre d'anti-motivation
func (pb *PromptBuilder) buildAntiMotivationPromptEN(company models.CompanyInfo) string {
	// Build experiences section for humor
	experiencesSection := pb.buildExperiencesSection()

	// Format date in English
	currentDate := formatEnglishDate(time.Now())

	template := `You are a comedian specialized in writing creative and absurd anti-motivation letters.

CANDIDATE PROFILE (to parody with humor):
- Name: %s
- Address: %s
- Postal Code, City: %s %s
- Email: %s
- Phone: %s
- Current Position: %s
- Years of Experience: %d years
- Key Skills: %s

REAL BACKGROUND (to parody):
%s

TARGET COMPANY:
- Name: %s
- Industry: %s
- Description: %s

CURRENT DATE (for the letter):
%s

TASK:
Write a humorous ANTI-MOTIVATION letter explaining why %s should DEFINITELY NOT be hired at %s.

STYLE AND TONE:
- Absurd humor and self-deprecation
- Obviously tongue-in-cheek (nobody should take this seriously)
- TWIST the candidate's real skills/experiences in a comedic way
- Pop culture references, puns, comedic exaggerations
- Light tone, never mean-spirited or offensive to the company
- Adapt humor for English-speaking audience (witty, dry British/American humor)

INSTRUCTIONS:
1. START OBLIGATORILY with the complete header in classic English format - EVEN FOR ANTI-MOTIVATION:
   - Full name of the candidate
   - Address
   - Postal code and city
   - Email
   - Phone
   - Blank line
   - Current date (use the one provided above)
   - Blank line
   - Company name
   - [Address if known, otherwise leave blank]
   - Blank line
   - "Subject: Anti-Motivation Letter (Humor - Not Serious)"
   - Blank line

2. Free structure afterwards (be creative!)
3. PARODY the candidate's real experiences (ex: "I reduced latency by 70%%... by deleting features")
4. Transform achievements into hilarious "anti-achievements"
5. Fake useless skills based on real ones
6. Absurd anecdotes related to real background
7. Ironic reversed conclusion
8. Length: 300-400 words (excluding header)
9. Avoid vulgar or offensive humor
10. END absurdly but with "Best regards (or not)," + name

STYLE EXAMPLES BASED ON REAL BACKGROUND:
- "My expertise in 'high-performance REST APIs' means I know how to crash 100K requests/day with style..."
- "I 'mentored 4 junior developers'... in the subtle art of professional procrastination..."
- "My '99.9%% uptime SLA' hides the 0.1%% where I panicked staring at my screen..."

REMINDER: This is humor! Use the REAL background to create personalized parodies.

Generate the letter now (WITH the complete header):`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.Address,
		pb.userProfile.PostalCode, pb.userProfile.City,
		pb.userProfile.Email,
		pb.userProfile.Phone,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		experiencesSection,
		company.Name,
		company.Industry,
		company.Description,
		currentDate,
		pb.userProfile.Name,
		company.Name,
	)
}

// formatFrenchDate formate une date au format français pour les lettres
// Ex: "Tourtenay, le 5 janvier 2026"
func formatFrenchDate(t time.Time) string {
	frenchMonths := map[time.Month]string{
		time.January:   "janvier",
		time.February:  "février",
		time.March:     "mars",
		time.April:     "avril",
		time.May:       "mai",
		time.June:      "juin",
		time.July:      "juillet",
		time.August:    "août",
		time.September: "septembre",
		time.October:   "octobre",
		time.November:  "novembre",
		time.December:  "décembre",
	}

	return fmt.Sprintf("Tourtenay, le %d %s %d",
		t.Day(),
		frenchMonths[t.Month()],
		t.Year(),
	)
}

// formatEnglishDate formate une date au format anglais pour les lettres
// Ex: "Tourtenay, January 5, 2026"
func formatEnglishDate(t time.Time) string {
	return fmt.Sprintf("Tourtenay, %s %d, %d",
		t.Month().String(),
		t.Day(),
		t.Year(),
	)
}
