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
	template := `Tu es un expert en rédaction de lettres de motivation professionnelles.

CONTEXTE CANDIDAT:
- Nom: %s
- Poste actuel: %s
- Années d'expérience: %d ans
- Compétences clés: %s

CONTEXTE ENTREPRISE:
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
3. Mets en avant l'alignement entre les compétences du candidat et les besoins probables de l'entreprise
4. Montre un intérêt sincère pour l'entreprise (culture, projets, technologies)
5. Sois spécifique et concret (évite les généralités)
6. Longueur: 250-350 mots
7. Format: paragraphes bien structurés (pas de bullet points)

EXEMPLES DE BON STYLE:
- "Votre engagement dans [technologie/projet spécifique] résonne particulièrement avec mon expérience en..."
- "Ayant travaillé X années sur [compétence], je serais ravi de contribuer à [objectif entreprise]..."

N'invente PAS de faits sur l'entreprise. Utilise uniquement les informations fournies.

Génère la lettre maintenant (sans formule de politesse finale "Cordialement", etc.) :`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		company.Name,
		company.Industry,
		company.Description,
		strings.Join(company.Technologies, ", "),
		company.Size,
		company.Name,
	)
}

// BuildAntiMotivationPrompt : prompt pour lettre d'anti-motivation humoristique
func (pb *PromptBuilder) BuildAntiMotivationPrompt(company models.CompanyInfo) string {
	template := `Tu es un humoriste spécialisé en rédaction de lettres d'anti-motivation créatives et absurdes.

CONTEXTE CANDIDAT:
- Nom: %s
- Poste actuel: %s
- Années d'expérience: %d ans
- Compétences clés: %s

CONTEXTE ENTREPRISE:
- Nom: %s
- Secteur: %s
- Description: %s

TÂCHE:
Rédige une lettre d'ANTI-MOTIVATION humoristique expliquant pourquoi le candidat ne devrait SURTOUT PAS être embauché chez %s.

STYLE ET TON:
- Humour absurde et auto-dérision
- Deuxième degré évident (personne ne doit prendre ça au sérieux)
- Références pop culture, jeux de mots, exagérations comiques
- Ton léger, jamais méchant ou offensant envers l'entreprise
- Créatif et original

INSTRUCTIONS:
1. Structure libre (sois créatif !)
2. Liste de "défauts" hilarants et absurdes
3. Fausses compétences inutiles ("Expert en procrastination", "Champion de café froid", etc.)
4. Anecdotes inventées ridicules
5. Conclusion ironique inversée
6. Longueur: 200-300 mots
7. Évite l'humour vulgaire ou offensant

EXEMPLES DE STYLE:
- "Mes 10 ans d'expérience en débogage de code m'ont surtout appris à créer des bugs encore plus créatifs..."
- "Je maîtrise l'art ancestral de transformer un projet de 2 semaines en 6 mois..."
- "Mon CV ressemble à un README.md mal formaté, ce qui est ironique vu que je suis développeur..."

RAPPEL: C'est de l'humour ! Le but est de faire sourire tout en montrant créativité et auto-dérision.

Génère la lettre maintenant:`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		company.Name,
		company.Industry,
		company.Description,
		company.Name,
	)
}
