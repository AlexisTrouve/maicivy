package services

// =============================================================================
// PROMPTS V2 — améliorations basées sur les études cover letter 2024-2025
// Changements vs v1 :
//   1. Hook d'ouverture : 1ère phrase = problème business précis, pas générique
//   2. Mirror vocabulary : reprise explicite des mots-clés de l'offre
//   3. Freelance economics : munitions concrètes (charges, congés, risque embauche)
//   4. Empathie décideur : se mettre dans la tête de celui qui a peur de se planter
// =============================================================================

import (
	"fmt"
	"strings"
	"time"

	"maicivy/internal/models"
)

// buildMotivationPromptFR_v2 : prompt français v2
func (pb *PromptBuilder) buildMotivationPromptFR_v2(company models.CompanyInfo, jobOffer string) string {
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
		mirrorVocabInstruction = `5. Identifie 2-3 mots ou formules spécifiques de l'offre (pas du jargon générique) et réutilise-les naturellement dans le corps — le recruteur doit sentir que la lettre leur appartient.`
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

Extrait 4 — ton posé, conclusion factuelle :
"Le monde change. Pas dans un fracas. Dans un souffle plaintif porté par le sel."
"Un marchand n'est pas un ami. Un marchand reste tant que le sac est plein."

CE QUE CES EXTRAITS RÉVÈLENT (applique-le) :
- Phrases courtes alternées avec phrases longues. Le rythme varie consciemment.
- Les pauses rythmiques se font avec des points courts, pas des tirets
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

PROJETS (cite-en 1-2 précisément si pertinents, pas tous) :
%s

ENTREPRISE CIBLE :
- %s | %s | %s
- Technologies : %s | Taille : %s
%s
DATE : %s

MUNITIONS FREELANCE (à intégrer naturellement dans MOI ou NOUS — choisir 1-2 selon le contexte) :
- Zéro charges patronales (~45%% du salaire brut) — le même budget embauche CDI = mission freelance premium
- Zéro congés payés, arrêts maladie, mutuelle, intéressement
- Pas de période d'essai, pas de préavis de 3 mois — démarrage en une semaine si besoin
- Coût réel prévisible et plafonné — pas d'aléas RH en cours de route
- Montée en charge ou arrêt selon le besoin du projet — flexibilité pure
L'objectif : convertir une offre CDI en mission freelance. Chaque argument sert LEUR intérêt, pas le statut freelance.

AVANT D'ÉCRIRE — raisonne d'abord (ne montre pas ce raisonnement dans la lettre) :
1. Quel est leur problème RÉEL ? (pas celui décrit dans l'offre, le problème business derrière)
2. Pourquoi un freelance résout mieux ce problème qu'un CDI pour EUX ? Quel argument économique ou opérationnel est le plus pertinent ici ?
3. Quel projet ou réalisation d'Alexis est le plus parlant pour CE cas précis ?
4. Comment commencer la première phrase par une observation sur LEUR situation concrète — pas une généralité sur leur secteur, un fait précis sur leur contexte ?
5. Quel risque le décideur cherche-t-il à éviter ? (se planter dans un recrutement coûteux ? manquer une deadline critique ? un projet qui dérive ?)
%s

STRUCTURE À SUIVRE (Vous → Moi → Nous) :
• VOUS (1 paragraphe) : La PREMIÈRE PHRASE doit nommer leur problème business précis — une observation qui montre qu'on a réfléchi à LEUR contexte, pas une intro générique. Le reste développe leur situation dans leurs mots. Zéro "je" dans ce paragraphe.
• MOI (1-2 paragraphes) : 1-2 réalisations concrètes avec chiffres, directement liées à LEUR besoin. 1 projet pertinent nommé précisément. L'argument freelance le plus pertinent intégré naturellement — pas listé.
• NOUS (1 paragraphe court) : proposition concrète de mission, durée indicative, CTA à faible friction ("30 minutes cette semaine ?")

FORMAT :
- EN-TÊTE classique FR (gauche) : nom / adresse / email / téléphone / ligne vide / date / ligne vide / entreprise / ligne vide / "Objet : %s" / ligne vide
- IMPORTANT : la ligne Objet est IMPOSÉE ci-dessus — ne pas la réécrire, ne pas l'adapter
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
		mirrorVocabInstruction,
		objet,
	)
}

// buildMotivationPromptEN_v2 : prompt anglais v2
func (pb *PromptBuilder) buildMotivationPromptEN_v2(company models.CompanyInfo, jobOffer string) string {
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
		mirrorVocabInstruction = `5. Identify 2-3 specific words or phrases from the job offer (not generic jargon) and weave them naturally into the body — the recruiter should feel the letter was written for them specifically.`
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

PROJECTS (cite 1-2 precisely if relevant, not all):
%s

TARGET COMPANY:
- %s | %s | %s
- Technologies: %s | Size: %s
%s
DATE: %s

FREELANCE ECONOMICS (weave in 1-2 naturally in ME or US paragraph — choose what fits):
- Zero employer taxes (~30-40%% on top of gross salary) — same hiring budget = premium freelance mission
- No paid leave, sick days, health insurance, profit sharing to manage
- No probation period, no 3-month notice — start in a week if needed
- Predictable, capped cost — no HR surprises mid-project
- Scale up or stop based on project needs — pure flexibility
Goal: convert a job posting into a freelance mission. Every argument serves THEIR interest.

BEFORE WRITING — reason first (don't show this reasoning in the letter):
1. What is their REAL problem? (not what the job description says, the business problem behind it)
2. Why does a freelancer solve this better than a full-time hire FOR THEM? Which economic or operational argument fits best?
3. Which of Alexis's projects or achievements is most relevant for THIS specific case?
4. How to open the FIRST SENTENCE with a concrete observation about THEIR specific situation — not a generic industry statement?
5. What risk is the decision-maker trying to avoid? (costly mis-hire? missing a critical deadline? a project going off the rails?)
%s

STRUCTURE TO FOLLOW (You → Me → Us):
• YOU (1 paragraph): The FIRST SENTENCE must name their specific business problem — an observation showing you thought about THEIR context, not a generic intro. Zero "I" in this paragraph.
• ME (1-2 paragraphs): 1-2 concrete achievements with numbers, directly tied to THEIR need. 1 project named precisely. The most relevant freelance argument woven in naturally — not listed.
• US (1 short paragraph): concrete mission proposal, indicative duration, low-friction CTA ("30 minutes this week?")

FORMAT:
- HEADER classic format (left-aligned): name / address / email / phone / blank line / date / blank line / company / blank line / "Subject: %s" / blank line
- IMPORTANT: the Subject line is IMPOSED above — do not rewrite it, do not adapt it
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
		mirrorVocabInstruction,
		subject,
	)
}
