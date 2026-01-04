package services

import (
	"fmt"
	"strings"

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

	template := `Tu es un expert en rédaction de lettres de motivation professionnelles.

PROFIL DU CANDIDAT:
- Nom: %s
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

TÂCHE:
Rédige une lettre de motivation professionnelle, convaincante et authentique pour postuler chez %s.

INSTRUCTIONS:
1. Structure classique (introduction, corps, conclusion)
2. Ton professionnel mais pas rigide
3. UTILISE des exemples CONCRETS du parcours du candidat (projets, achievements, métriques)
4. Mets en avant l'alignement entre les compétences du candidat et les besoins probables de l'entreprise
5. Montre un intérêt sincère pour l'entreprise (culture, projets, technologies)
6. Cite des réalisations spécifiques avec des chiffres quand disponibles
7. Longueur: 350-450 mots
8. Format: paragraphes bien structurés (pas de bullet points)

EXEMPLES DE BON STYLE:
- "Chez [entreprise précédente], j'ai [réalisation concrète avec métrique], ce qui m'a préparé à..."
- "Mon expérience en [technologie] où j'ai [achievement] correspond parfaitement à vos besoins en..."

N'invente PAS de faits sur l'entreprise. Utilise les informations du parcours du candidat.

Génère la lettre maintenant:`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
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

	template := `Tu es un humoriste spécialisé en rédaction de lettres d'anti-motivation créatives et absurdes.

PROFIL DU CANDIDAT (à détourner avec humour):
- Nom: %s
- Poste actuel: %s
- Années d'expérience: %d ans
- Compétences clés: %s

VRAI PARCOURS (à parodier):
%s

ENTREPRISE CIBLE:
- Nom: %s
- Secteur: %s
- Description: %s

TÂCHE:
Rédige une lettre d'ANTI-MOTIVATION humoristique expliquant pourquoi %s ne devrait SURTOUT PAS être embauché chez %s.

STYLE ET TON:
- Humour absurde et auto-dérision
- Deuxième degré évident (personne ne doit prendre ça au sérieux)
- DÉTOURNE les vraies compétences/expériences du candidat de manière comique
- Références pop culture, jeux de mots, exagérations comiques
- Ton léger, jamais méchant ou offensant envers l'entreprise

INSTRUCTIONS:
1. Structure libre (sois créatif !)
2. PARODIE les vraies expériences du candidat (ex: "J'ai réduit la latence de 70%%... en supprimant les features")
3. Transforme les achievements en "anti-achievements" hilarants
4. Fausses compétences inutiles basées sur les vraies
5. Anecdotes absurdes liées au vrai parcours
6. Conclusion ironique inversée
7. Longueur: 300-400 mots
8. Évite l'humour vulgaire ou offensant

EXEMPLES DE STYLE BASÉS SUR LE VRAI PARCOURS:
- "Mon expertise en 'high-performance REST APIs' signifie que je sais faire crasher 100K requêtes/jour avec style..."
- "J'ai 'mentoré 4 développeurs juniors'... dans l'art subtil de la procrastination professionnelle..."
- "Mon '99.9%% uptime SLA' cache les 0.1%% où j'ai paniqué devant mon écran..."

RAPPEL: C'est de l'humour ! Utilise le VRAI parcours pour créer des parodies personnalisées.

Génère la lettre maintenant:`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		experiencesSection,
		company.Name,
		company.Industry,
		company.Description,
		pb.userProfile.Name,
		company.Name,
	)
}
