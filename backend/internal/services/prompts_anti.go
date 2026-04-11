package services

import (
	"fmt"
	"strings"
	"time"

	"maicivy/internal/models"
)

// buildAntiMotivationPromptFR : prompt français pour lettre d'anti-motivation humoristique
func (pb *PromptBuilder) buildAntiMotivationPromptFR(company models.CompanyInfo) string {
	experiencesSection := pb.buildExperiencesSection()
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

// buildAntiMotivationPromptEN : prompt anglais pour lettre d'anti-motivation humoristique
func (pb *PromptBuilder) buildAntiMotivationPromptEN(company models.CompanyInfo) string {
	experiencesSection := pb.buildExperiencesSection()
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
