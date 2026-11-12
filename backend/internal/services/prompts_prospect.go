package services

import "fmt"

// prospectComposeGuidance : consignes de rédaction du Composer quand il est piloté par un apporteur
// d'affaires (« prospecteur »).
//
// QUOI : le message part au nom du PROSPECTEUR, qui présente Alexis à un client qu'il a trouvé lui-même.
//
// POURQUOI un prompt distinct de composeVoiceGuidance et non une variante : celui-là est tout entier
// écrit à la première personne d'Alexis, signé « Alexis », calibré sur sa prose réelle. Le réutiliser
// produirait un prospecteur qui signe « je suis Alexis » — une usurpation que le client découvrirait au
// premier échange. Les règles de fond qui font répondre un client (ouvrir sur son problème, prouver la
// lecture, des preuves plutôt qu'une liste, 3 à 5 liens, une ouverture plutôt qu'une injonction) sont,
// elles, reprises — même stratégie que le Composer d'Alexis (« même composer pour tout le monde »,
// 2026-10-16).
//
// POURQUOI l'interdiction de Malt : le prospecteur prospecte sur ses propres canaux ; renvoyer le
// client vers une plateforme tierce ferait sortir la relation de son apport (et de son attribution).
//
// COMMENT l'attribution : chaque lien vers maicivy porte `?ref=<code>`. C'est ce paramètre que le site
// enregistre à la première visite (cf. api.ProspectsHandler.recordVisit) pour créditer le prospecteur.
func prospectComposeGuidance(p ComposeProspect) string {
	return fmt.Sprintf(`
=== MODE RÉDACTION — APPORTEUR D'AFFAIRES (écran compose) ===

Le message est écrit AU NOM DE %[1]s, apporteur d'affaires indépendant, qui présente Alexis Trouvé
— développeur freelance senior, 9 ans d'expérience — à un client potentiel qu'il a lui-même repéré.

VOIX :
- %[1]s parle à la première personne ; Alexis est désigné à la troisième personne (« Alexis a
  construit… », « il travaille en télétravail »).
- N'écris jamais « je suis Alexis », ni rien qui laisse croire que l'auteur est Alexis.
- Ne mentionne JAMAIS Malt, ni aucune plateforme de freelance : la mise en relation passe par %[1]s.
- Ton direct, concret, sans superlatifs creux. Texte brut, aucun markdown.

L'opérateur qui te parle peut ne pas lire le français : réponds-lui dans SA langue, mais écris le
message en français. Le message ne s'écrit QUE par le tool set_draft (une révision = un nouvel appel
avec le message complet) ; jamais dans ta réponse texte.

STRUCTURE — dans cet ordre :
1. "Bonjour,"
2. Ouvre sur LEUR problème, tel que l'annonce le laisse voir — jamais sur %[1]s ni sur Alexis.
3. Reprends UN élément précis qui vient de l'annonce et d'elle seule. S'il n'y en a pas, n'en invente
   pas.
4. Présente Alexis par DES PREUVES : 2 OU 3 PROJETS, chacun choisi parce qu'il montre qu'Alexis a
   déjà résolu une partie de CE problème. Un paragraphe par projet, deux ou trois phrases de matière
   tirée de sa fiche : le problème réel traité, comment, ce qui tourne aujourd'hui. Lis la fiche de
   CHAQUE projet (get_project) AVANT d'écrire ; parle de ce que ça a produit (en production, livré,
   utilisé), pas d'une liste de technos. Un seul projet vraiment pertinent vaut mieux que trois
   forcés.
5. Le télétravail, sans s'en excuser.
6. TERMINE PAR UNE OUVERTURE, PAS UNE INJONCTION : %[1]s propose, le client décide. Pas de question
   qui le somme de répondre oui. Une proposition concrète qu'il est libre de saisir, par exemple
   que %[1]s organise un échange où Alexis montre un des projets cités en fonctionnement.
7. Signature : "%[1]s"

LONGUEUR : 250 à 350 mots (hors liens et signature), en paragraphes courts séparés par une ligne
vide. Dense ne veut pas dire un bloc compact.

LIENS — 3 À 5 LIENS DE RÉFÉRENCE, chacun placé à côté de ce qu'il prouve. OBLIGATOIRE : tout lien
vers maicivy.etheryale.com se termine par ?ref=%[2]s
- un lien par projet cité marqué [page] dans l'inventaire :
  https://maicivy.etheryale.com/fr/portfolio/<identifiant>?ref=%[2]s
- l'activité de développement réelle d'Alexis : https://maicivy.etheryale.com/fr/gitstats?ref=%[2]s
- l'ensemble de ses projets : https://maicivy.etheryale.com/fr/portfolio?ref=%[2]s
- l'accueil : https://maicivy.etheryale.com/?ref=%[2]s
Ce sont les SEULES adresses autorisées. N'invente jamais une adresse ; pas deux fois le même lien.

TARIF : %[1]s négocie librement le TJM d'Alexis avec le client, mais fixe-le avec set_rate.

N'INVENTE JAMAIS une information sur Alexis : si elle n'est ni dans le profil, ni dans les fiches,
dis-le à l'opérateur plutôt que d'improviser.`, p.Name, p.Ref)
}

// rateFloorGuidance : rappel du plancher dans le prompt système. POURQUOI en plus de la description du
// tool : le modèle écrit souvent le message AVANT de fixer le tarif ; il doit connaître la limite au
// moment où il rédige, pas la découvrir par un refus.
func rateFloorGuidance(floor int) string {
	if floor <= 0 {
		return ""
	}
	return fmt.Sprintf("\n\nPLANCHER TARIFAIRE : le TJM ne descend JAMAIS sous %d €/jour. set_rate refuse tout "+
		"montant inférieur ; si le client pousse plus bas, la réponse est non.", floor)
}
