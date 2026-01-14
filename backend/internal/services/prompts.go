package services

import (
	"fmt"
	"strings"
	"time"

	"maicivy/internal/models"
)

type PromptBuilder struct {
	userProfile models.UserProfile
}

func NewPromptBuilder(profile models.UserProfile) *PromptBuilder {
	return &PromptBuilder{userProfile: profile}
}

// BuildMotivationPrompt : prompt pour lettre de motivation professionnelle
func (pb *PromptBuilder) BuildMotivationPrompt(company models.CompanyInfo) string {
	// Construire la section expériences détaillées
	experiencesSection := pb.buildExperiencesSection()

	// Formater la date en français (ex: "Tourtenay, le 5 janvier 2026")
	currentDate := formatFrenchDate(time.Now())

	template := `Tu es un expert en rédaction de lettres de motivation professionnelles.

PROFIL DU CANDIDAT:
- Nom: %s
- Adresse: %s
- Code postal, Ville: %s %s
- Email: %s
- Téléphone: %s
- Résumé: %s
- Poste actuel: %s
- Années d'expérience: %d ans
- Compétences clés: %s

PARCOURS PROFESSIONNEL DÉTAILLÉ:
%s

ENTREPRISE CIBLE:
- Nom: %s
- Secteur: %s
- Description: %s
- Technologies utilisées: %s
- Taille: %s

DATE DU JOUR (pour la lettre):
%s

TÂCHE:
Rédige une lettre de motivation professionnelle, convaincante et authentique pour postuler chez %s.

INSTRUCTIONS:
1. COMMENCE OBLIGATOIREMENT par l'en-tête complet au format français classique (aligné à gauche):
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
   - "Objet : Candidature spontanée"
   - Ligne vide

2. Structure classique ensuite (introduction, corps, conclusion)
3. Ton professionnel mais pas rigide
4. UTILISE des exemples CONCRETS du parcours du candidat (projets, achievements, métriques)
5. Mets en avant l'alignement entre les compétences du candidat et les besoins probables de l'entreprise
6. Montre un intérêt sincère pour l'entreprise (culture, projets, technologies)
7. Cite des réalisations spécifiques avec des chiffres quand disponibles
8. Longueur: 350-450 mots (sans compter l'en-tête)
9. Format: paragraphes bien structurés (pas de bullet points)
10. TERMINE par "Cordialement," suivi du nom du candidat

EXEMPLES DE BON STYLE:
- "Chez [entreprise précédente], j'ai [réalisation concrète avec métrique], ce qui m'a préparé à..."
- "Mon expérience en [technologie] où j'ai [achievement] correspond parfaitement à vos besoins en..."

N'invente PAS de faits sur l'entreprise. Utilise les informations du parcours du candidat.

Génère la lettre maintenant (AVEC l'en-tête complet):`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.Address,
		pb.userProfile.PostalCode, pb.userProfile.City,
		pb.userProfile.Email,
		pb.userProfile.Phone,
		pb.userProfile.Summary,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		experiencesSection,
		company.Name,
		company.Industry,
		company.Description,
		strings.Join(company.Technologies, ", "),
		company.Size,
		currentDate,
		company.Name,
	)
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

// BuildAntiMotivationPrompt : prompt pour lettre d'anti-motivation humoristique
func (pb *PromptBuilder) BuildAntiMotivationPrompt(company models.CompanyInfo) string {
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
