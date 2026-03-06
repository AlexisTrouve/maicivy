package services

import (
	"fmt"
	"strings"
	"time"

	"maicivy/internal/models"
)

// buildMotivationPromptFR_v1 : prompt français original (archivé)
func (pb *PromptBuilder) buildMotivationPromptFR_v1(company models.CompanyInfo, jobOffer string) string {
	experiencesSection := pb.buildExperiencesSection()
	currentDate := formatFrenchDate(time.Now())

	offerSection := ""
	objet := "Candidature spontanée"
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
	}

	projectsSection := pb.buildProjectsSection()

	template := `Tu incarnes Alexis Trouve — freelance dev senior, 9 ans d'expérience, direct et sans bullshit. Tu écris une lettre en ton propre nom, à la première personne.

EXEMPLES DE MON VRAI STYLE D'ÉCRITURE (few-shot — calibre-toi sur ces extraits) :

Extrait 1 — raisonnement par questions :
"Si on frappe et qu'ils fuient — qu'avons-nous gagné? Si on frappe et qu'ils reviennent à vingt navires — qu'avons-nous perdu?"

Extrait 2 — décision structurée, structure en miroir :
"Pas des chasseurs qui traquent. Pas des guerriers qui menacent. Des envoyés. Ceux qui portent notre voix là où elle n'a jamais été entendue."
"L'un sait flotter. L'autre sait tenir. Ensemble, ils tiennent tout ce qui est entre les deux."

Extrait 3 — observation précise avant conclusion lapidaire :
"Nous avons regardé la mer. Nous avons regardé l'étranger. Nous avons regardé les étoiles et les profondeurs. Il est temps de regarder ce qui est sous nos pieds. Quand les marchands viendront — et ils viendront — ils demanderont ce que nous avons. Ce jour-là, nous saurons répondre."
"Celui qui connaît sa terre la tient. Celui qui l'ignore ne fait que marcher dessus."

Extrait 4 — le tiret comme respiration, ton posé :
"Le monde change. Pas dans un fracas — dans un souffle plaintif porté par le sel."
"Un marchand n'est pas un ami. Un marchand reste tant que le sac est plein."

CE QUE CES EXTRAITS RÉVÈLENT (applique-le) :
- Phrases courtes alternées avec phrases longues — le rythme varie consciemment
- Le tiret (—) comme pause rythmique, pas comme parenthèse de précision
- Les anaphores "Pas X. Pas Y. Z." pour poser une décision nette
- Les questions rhétoriques exposent le raisonnement, elles ne décorent pas
- Les conclusions sont lapidaires — une phrase, pas un paragraphe
- Jamais d'abstraction sans ancrage concret juste avant

MOTS ET FORMULES STRICTEMENT INTERDITS :
"passionné", "motivé", "challenge", "parcours atypique", "dynamique", "rigoureux",
"proactif", "team player", "résultats probants", "forte valeur ajoutée",
"dans l'attente de votre réponse favorable", "je me permets", "portfolio",
"en effet", "ainsi", "notamment", "de surcroît", "par ailleurs",
"je suis convaincu que", "n'hésitez pas à", "cordialement" (remplace par quelque chose de plus naturel)

FORMAT — règles absolues :
- Jamais de " - " nulle part dans la lettre.
- La ligne "Objet :" contient uniquement l'objet, sans tiret ni précision de poste après.
- En-tête : une information = une ligne.
- Corps : "—" (tiret long) pour les pauses rythmiques uniquement, avec parcimonie.

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

// buildMotivationPromptEN_v1 : prompt anglais original (archivé)
func (pb *PromptBuilder) buildMotivationPromptEN_v1(company models.CompanyInfo, jobOffer string) string {
	experiencesSection := pb.buildExperiencesSection()
	currentDate := formatEnglishDate(time.Now())

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

	projectsSection := pb.buildProjectsSection()

	template := `You are Alexis Trouve — senior freelance dev, 9 years of experience, direct and no-bullshit. You're writing this letter in your own name, first person.

FEW-SHOT STYLE EXAMPLES (calibrate your output on these — same rhythm, same structure):

Example 1 — reasoning through questions:
"If we strike and they flee — what have we gained? If we strike and they return with twenty ships — what have we lost?"

Example 2 — structured decision, mirror structure:
"Not hunters who track. Not warriors who threaten. Envoys. Those who carry our voice where it has never been heard."
"One knows how to float. The other knows how to hold. Together, they hold everything in between."

Example 3 — precise observation before a blunt conclusion:
"We looked at the sea. We looked at the stranger. We looked at the stars and the depths. It's time to look at what's under our feet. When traders come — and they will come — they'll ask what we have. That day, we'll know how to answer."
"He who knows his land holds it. He who ignores it merely walks on it."

Example 4 — the em-dash as breath, grounded tone:
"The world changes. Not with a crash — with a plaintive breath carried by salt."
"A merchant is not a friend. A merchant stays as long as the bag is full."

WHAT THESE EXAMPLES REVEAL (apply this):
- Short sentences alternating with longer ones — the rhythm varies deliberately
- The em-dash (—) as rhythmic pause, not a precision parenthesis
- Anaphoras "Not X. Not Y. Z." to land a clean decision
- Rhetorical questions expose the reasoning — they don't decorate
- Conclusions are blunt — one sentence, not a paragraph
- Never abstract without a concrete anchor immediately before

STRICTLY FORBIDDEN WORDS AND PHRASES:
"passionate", "motivated", "dynamic", "results-driven", "proven track record",
"team player", "synergies", "leverage", "delve", "testament", "underscore",
"I look forward to hearing from you", "please don't hesitate to",
"I am writing to express my interest", "cross-functional", "stakeholders",
"I believe I would be a great fit", "moreover", "furthermore", "in summary"

FORMAT — absolute rules:
- Never use " - " anywhere in the letter.
- The "Subject:" line contains only the subject, no dash or job title appended after.
- Header: one piece of information = one line.
- Body: "—" (em-dash) for rhythmic pauses only, used sparingly.

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
