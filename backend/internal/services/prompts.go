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

	template := `Tu es un expert en conversion commerciale B2B freelance tech.

MISSION : Écrire une lettre qui CONVAINC le décideur de travailler avec Alexis en mission freelance — que l'offre soit pour un CDI ou un freelance, l'objectif est le même : convertir en mission.

PROFIL DU CANDIDAT:
- Nom: %s
- Adresse: %s %s, %s
- Email: %s
- Téléphone: %s
- Résumé: %s
- Poste actuel: %s
- Années d'expérience: %d ans
- Compétences clés: %s

PARCOURS PROFESSIONNEL:
%s

PROJETS PERSONNELS RÉALISÉS (preuves concrètes de compétence):
%s

ENTREPRISE CIBLE:
- Nom: %s
- Secteur: %s
- Description: %s
- Technologies: %s
- Taille: %s
%s
DATE DU JOUR: %s

ARGUMENTS FREELANCE À UTILISER (intègre-les naturellement, pas en liste):
• ÉCONOMIQUE : pas de charges patronales (~42%% du brut économisés), pas de mutuelle, pas de congés payés, pas d'onboarding à financer
• FLEXIBILITÉ : mission à durée définie, prolongeable ou arrêtable, charge adaptable aux phases du projet
• ZÉRO RISQUE RH : pas de période d'essai incertaine, pas de procédure de licenciement, pas d'engagement long terme
• VITESSE : disponibilité immédiate, pas de préavis à attendre, opérationnel dès le premier jour
• PERFORMANCE : un freelance senior livre vite et bien — pas de montée en compétence à gérer, résultats mesurables

EMPATHIE DÉCIDEUR — tu dois comprendre et verbaliser leur réalité :
• Recruter en CDI = 6-12 mois de risque financier (si ça ne matche pas, procédure longue et coûteuse)
• Budget RH contraint, pression des directions, projets qui évoluent
• Peur de se tromper sur un profil, de payer cher quelqu'un qui déçoit
• Besoin de résultats rapides, pas d'excuses ni de ramping
• La mission freelance résout tous ces problèmes — c'est ce que la lettre doit démontrer

TÂCHE:
Rédige une lettre qui positionne Alexis Trouve comme LA solution à leur problème, via une mission freelance. Pas une candidature classique — une proposition de valeur.

INSTRUCTIONS:
1. EN-TÊTE obligatoire au format français classique (aligné à gauche):
   - Nom complet
   - Adresse, code postal et ville
   - Email
   - Téléphone
   - Ligne vide
   - Date (utilise celle fournie)
   - Ligne vide
   - Nom de l'entreprise
   - Ligne vide
   - "Objet : %s"
   - Ligne vide

2. INTRODUCTION : accroche empathique — montre que tu comprends LEUR contexte et LEUR besoin. Pas "je vous écris pour..." — commence par eux, pas par toi.

3. CORPS : pitch de la mission concrète, pas du CV. Propose explicitement une mission freelance. Intègre 1-2 arguments économiques ou de flexibilité de manière fluide. Cite 2-3 projets/réalisations spécifiques qui prouvent la compétence sur leur cas précis.

4. Si une offre est fournie : adresse directement les missions demandées, réutilise leur vocabulaire, montre que tu as lu et compris.

5. CONCLUSION : appel à l'action direct — propose un échange rapide pour cadrer la mission. Pas de formule creuse.

6. TON : direct, confiant, empathique. Jamais servile ni corporate. Comme quelqu'un qui sait ce qu'il vaut et qui propose une solution — pas quelqu'un qui supplie.

7. Longueur : 350-450 mots hors en-tête. Paragraphes, pas de bullet points dans la lettre.

8. TERMINE par "Cordialement," suivi du nom.

N'invente PAS de faits sur l'entreprise. Utilise uniquement les infos fournies.

Génère la lettre maintenant (AVEC l'en-tête complet) :`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.PostalCode, pb.userProfile.City, pb.userProfile.Address,
		pb.userProfile.Email,
		pb.userProfile.Phone,
		pb.userProfile.Summary,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		experiencesSection,
		projectsSection,
		company.Name,
		company.Industry,
		company.Description,
		strings.Join(company.Technologies, ", "),
		company.Size,
		offerSection,
		currentDate,
		company.Name,
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

	template := `You are a B2B freelance tech conversion specialist.

MISSION: Write a letter that CONVINCES the decision-maker to work with Alexis as a freelance contractor — whether the offer is for a full-time role or freelance, the goal is the same: convert it into a mission.

CANDIDATE PROFILE:
- Name: %s
- Address: %s %s, %s
- Email: %s
- Phone: %s
- Summary: %s
- Current Position: %s
- Years of Experience: %d years
- Key Skills: %s

PROFESSIONAL BACKGROUND:
%s

PERSONAL PROJECTS (concrete proof of skills):
%s

TARGET COMPANY:
- Name: %s
- Industry: %s
- Description: %s
- Technologies: %s
- Size: %s
%s
CURRENT DATE: %s

FREELANCE ARGUMENTS TO USE (weave them naturally, not as a list):
• ECONOMIC: no employer payroll taxes (~30%% savings on total compensation), no benefits overhead, no onboarding costs
• FLEXIBILITY: fixed-duration mission, extendable or stoppable, workload adaptable to project phases
• ZERO HR RISK: no uncertain trial period, no termination procedure, no long-term commitment
• SPEED: immediate availability, no notice period, operational from day one
• PERFORMANCE: a senior freelancer delivers fast — no ramp-up, measurable results

DECISION-MAKER EMPATHY — understand and verbalize their reality:
• Hiring full-time = 6-12 months of financial risk (if it doesn't work, a long and costly process)
• Tight HR budget, shifting priorities, projects that evolve
• Fear of making a wrong hire, of paying a lot for someone who underdelivers
• Need for fast results, not excuses or onboarding delays
• A freelance mission solves all of this — that's what the letter must demonstrate

TASK:
Write a letter that positions Alexis Trouve as THE solution to their problem, via a freelance mission. Not a classic application — a value proposition.

INSTRUCTIONS:
1. HEADER — mandatory, classic English format (left-aligned):
   - Full name
   - Address, postal code and city
   - Email
   - Phone
   - Blank line
   - Date (use the one provided)
   - Blank line
   - Company name
   - Blank line
   - "Subject: %s"
   - Blank line

2. OPENING: empathetic hook — show you understand THEIR context and THEIR need. Don't start with "I am writing to..." — start with them, not yourself.

3. BODY: pitch the mission, not the CV. Explicitly propose a freelance mission. Weave in 1-2 economic or flexibility arguments naturally. Reference 2-3 specific projects/achievements that prove your competence for their exact case.

4. If a job offer is provided: directly address the stated responsibilities, reuse their vocabulary, show you read and understood it.

5. CLOSING: direct call to action — propose a quick conversation to scope the mission. No empty phrases.

6. TONE: direct, confident, empathetic. Never servile or corporate. Like someone who knows their value and is proposing a solution — not someone begging for a job.

7. Length: 350-450 words excluding header. Paragraphs, no bullet points in the letter itself.

8. END with "Sincerely," followed by the name.

Do NOT invent facts about the company. Use only the information provided.

Generate the letter now (WITH the complete header):`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.PostalCode, pb.userProfile.City, pb.userProfile.Address,
		pb.userProfile.Email,
		pb.userProfile.Phone,
		pb.userProfile.Summary,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		experiencesSection,
		projectsSection,
		company.Name,
		company.Industry,
		company.Description,
		strings.Join(company.Technologies, ", "),
		company.Size,
		offerSection,
		currentDate,
		company.Name,
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
