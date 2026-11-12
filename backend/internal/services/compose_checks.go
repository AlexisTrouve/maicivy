package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Contrôles MÉCANIQUES d'un draft de message : tout ce qui est vérifiable sans LLM.
//
// POURQUOI ce fichier existe : l'opérateur de l'écran /admin/compose ne lit pas le français. Il ne peut
// pas juger le message qu'il s'apprête à envoyer à un client. Si la checklist affichée à l'écran était
// notée par le modèle qui vient d'écrire le message, elle cocherait ses propres cases — l'opérateur
// n'aurait aucun moyen d'attraper l'erreur, et la checklist deviendrait une machine à fabriquer de la
// confiance injustifiée. Pire que pas de checklist du tout : sans elle il sait qu'il ne sait pas.
//
// D'où la règle : ce qui est vérifiable en code l'est en code. Une case cochée ici est une PREUVE.
// Les contrôles qui demandent du jugement (ton, adéquation à l'annonce) vivent ailleurs et sont
// affichés séparément, pour que l'opérateur ne confonde pas les deux natures.

// CheckStatus : état d'un contrôle. Le tri-état est nécessaire — « non applicable » n'est pas un échec.
type CheckStatus string

const (
	CheckPass CheckStatus = "pass"
	CheckFail CheckStatus = "fail"
	// CheckNA : le contrôle ne s'applique pas à ce draft (ex : aucun tarif énoncé, donc rien à
	// contredire). Doit s'afficher différemment d'un échec, sinon l'opérateur cherche à corriger un
	// problème qui n'existe pas.
	CheckNA CheckStatus = "na"
)

// Identifiants stables des contrôles. STABLES parce que le frontend les traduit via next-intl : ce
// sont des clés, jamais des phrases (règle i18n du projet — aucune string visible en dur côté backend,
// qui ne connaît de toute façon pas la langue de l'opérateur).
const (
	CheckRemote = "remote"
	// CheckMaicivyLink : le message contient-il l'ADRESSE de maicivy, et pas seulement son nom ?
	//
	// POURQUOI c'est le lien qu'on contrôle et PAS la mention du nom : citer et lier ne
	// produisent pas le même effet. Le nom est une affirmation, le lien est une preuve qu'on ouvre
	// en un clic — et c'est le lien qui fait la différence sur une candidature (constat d'Alexi).
	// Les séparer permet à l'opératrice de voir LEQUEL des deux manque : « il faut ajouter le lien »
	// est actionnable, « le message n'est pas bon » ne l'est pas.
	CheckMaicivyLink = "maicivy_link"
	// CheckProjectLink : les liens vers la PAGE d'un projet (/portfolio/<id>) mènent-ils à une page
	// qui existe ? Cf. checkProjectLink.
	CheckProjectLink   = "project_link"
	CheckNoMarkdown    = "no_markdown"
	CheckSignature     = "signature"
	CheckRateCoherence = "rate_coherence"
	CheckLength        = "length"
)

// defaultMaxDraftRunes : plafond de longueur d'un message de prospection. Les plateformes tronquent
// silencieusement au-delà d'une certaine taille ; l'opérateur ne s'en apercevrait qu'après envoi.
// Relevé de 1500 à 3000 le 2026-10-16 : le Composer vise désormais 250 à 350 mots ET 3 à 5 liens
// (~60 caractères chacun), soit environ 2 400 caractères. À 1500, chaque message conforme au prompt
// aurait été marqué en échec.
const defaultMaxDraftRunes = 3000

// ComposeCheckInput : ce que le vérificateur observe. Volontairement un état plat et non l'artefact
// complet — ces contrôles doivent rester purs et testables sans monter tout le service.
type ComposeCheckInput struct {
	Draft     string // le message tel qu'il sera collé, en texte brut
	Signature string // nom attendu en dernière ligne (ex: "Alexis"). Vide = contrôle non applicable.
	// RetainedRate : TJM retenu, en euros. 0 = aucun tarif arbitré, contrôle non applicable.
	RetainedRate int
	MaxRunes     int // 0 = defaultMaxDraftRunes
	// PortfolioPages : identifiants des projets qui ont une page publique (/portfolio/<id>). nil =
	// liste indisponible, le contrôle des liens profonds est alors non applicable (jamais deviné).
	PortfolioPages map[string]bool
}

// MechanicalCheck : résultat d'un contrôle. Detail est un fragment factuel (le marqueur fautif, le
// montant contradictoire) — pas une phrase traduite : l'opérateur doit voir CE qui cloche, et le
// libellé vient du frontend.
type MechanicalCheck struct {
	ID     string      `json:"id"`
	Status CheckStatus `json:"status"`
	Detail string      `json:"detail,omitempty"`
}

// Formulations acceptables du travail à distance, sous forme repliée (voir foldAccents). Le modèle
// n'écrit pas toujours le même mot et l'opérateur ne peut pas arbitrer entre « télétravail » et
// « remote » — les deux doivent passer.
var remoteMarkers = []string{"remote", "teletravail", "a distance", "distanciel"}

// Marqueurs markdown qui cassent un copier-coller vers une plateforme. POURQUOI ce contrôle : en
// session réelle l'agent a rendu un message entier en citation (« > » sur chaque ligne) avec du gras,
// et l'opératrice a dû nettoyer à la main sans comprendre d'où venaient ces caractères. Un message
// destiné à être collé tel quel ne contient aucune syntaxe.
var inlineMarkdownMarkers = []string{"**", "__", "`"}

// rateWithCurrency : un montant suivi d'une devise (« 300€ », « 300 euros », « 1 300 € »).
// rateAfterKeyword : un montant introduit par un mot de tarification, même sans devise.
// Les deux sont nécessaires : le premier rate « TJM 300 », le second rate « budget de 300€ ».
var (
	rateWithCurrency = regexp.MustCompile(`(?i)(\d{1,2}[  ]\d{3}|\d{2,5})\s*(?:€|euros?\b|eur\b)`)
	rateAfterKeyword = regexp.MustCompile(`(?i)(?:tjm|tarif|taux journalier)[^.\n]{0,30}?(\d{1,2}[  ]\d{3}|\d{2,5})`)
)

// foldAccents : minuscules + repli des accents français. POURQUOI maison plutôt que golang.org/x/text
// (présent en indirect) : cinq lignes suffisent pour le sous-ensemble français et ça évite de
// promouvoir une dépendance directe pour un besoin de cette taille.
// COMMENT : 1. tout en minuscules, 2. remplacement rune à rune des accentuées par leur base.
func foldAccents(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'á', 'à', 'â', 'ä', 'ã', 'å':
			b.WriteRune('a')
		case 'é', 'è', 'ê', 'ë':
			b.WriteRune('e')
		case 'í', 'ì', 'î', 'ï':
			b.WriteRune('i')
		case 'ó', 'ò', 'ô', 'ö', 'õ':
			b.WriteRune('o')
		case 'ú', 'ù', 'û', 'ü':
			b.WriteRune('u')
		case 'ç':
			b.WriteRune('c')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// RunMechanicalChecks exécute tous les contrôles déterministes sur un draft.
// L'ordre du retour est stable : le frontend affiche les pastilles dans cet ordre.
func RunMechanicalChecks(in ComposeCheckInput) []MechanicalCheck {
	return []MechanicalCheck{
		checkRemote(in.Draft),
		checkMaicivyLink(in.Draft),
		checkProjectLink(in.Draft, in.PortfolioPages),
		checkNoMarkdown(in.Draft),
		checkSignature(in.Draft, in.Signature),
		checkRateCoherence(in.Draft, in.RetainedRate),
		checkLength(in.Draft, in.MaxRunes),
	}
}

// checkRemote : le message dit-il que la mission est à distance ? C'est la seule règle permanente dure
// du profil aujourd'hui — les autres (dispos, process) ne sont pas encore modélisées.
func checkRemote(draft string) MechanicalCheck {
	folded := foldAccents(draft)
	for _, marker := range remoteMarkers {
		if strings.Contains(folded, marker) {
			return MechanicalCheck{ID: CheckRemote, Status: CheckPass, Detail: marker}
		}
	}
	return MechanicalCheck{ID: CheckRemote, Status: CheckFail}
}

// maicivyHost : l'adresse du portfolio, en minuscules et sans schéma.
//
// En dur plutôt qu'en configuration : c'est le domaine du projet lui-même, pas un paramètre de
// déploiement. Le jour où il change, ce contrôle DOIT être revu avec le reste — le rendre
// configurable ferait passer un changement de domaine inaperçu, en laissant le contrôle vert sur
// une adresse morte.
const maicivyHost = "maicivy.etheryale.com"

// checkMaicivyLink : le message contient-il l'ADRESSE de maicivy, et pas seulement son nom ?
//
// COMMENT : on cherche l'hôte, pas une URL bien formée. Un lien collé depuis un navigateur, tapé à
// la main sans `https://`, ou glissé entre parenthèses doit compter — c'est le même geste pour le
// lecteur, qui verra une adresse cliquable dans tous les cas.
//
// POURQUOI on ne se contente PAS de chercher « maicivy » : c'est ce que fait déjà checkMaicivy, et
// « J'ai construit maicivy. » passerait alors pour un lien. Le point qui suit le nom en fin de
// phrase est précisément le piège — d'où la recherche de l'hôte COMPLET.
func checkMaicivyLink(draft string) MechanicalCheck {
	if strings.Contains(strings.ToLower(foldAccents(draft)), maicivyHost) {
		return MechanicalCheck{ID: CheckMaicivyLink, Status: CheckPass}
	}
	return MechanicalCheck{ID: CheckMaicivyLink, Status: CheckFail}
}

// lienProjet : un lien vers la page d'un projet, avec ou sans schéma, avec ou sans locale. L'id
// s'arrête au premier caractère qui ne peut pas en faire partie (ponctuation collée, parenthèse).
var lienProjet = regexp.MustCompile(`(?i)maicivy\.etheryale\.com/(?:[a-z]{2}/)?portfolio/([a-z0-9_-]+)`)

// checkProjectLink : chaque lien vers la page d'un projet mène-t-il à une page qui existe ?
//
// POURQUOI : constat Malt d'août-octobre 2026, 20 messages sur 25 liaient déjà l'accueil du site, et
// le seul prospect qui a vraiment répondu a décroché sur « pas de lien public » — il voulait voir le
// projet CITÉ, pas une vitrine générale. Le prompt demande donc le lien de la page du projet. Ce
// contrôle couvre le risque que ça crée : un identifiant déformé produit un lien mort, que l'opératrice
// ne peut pas repérer (elle ne lit pas le français et n'a aucune raison d'ouvrir le lien).
//
// COMMENT : pass si au moins un lien profond et tous valides ; fail au premier lien vers une page
// inexistante (Detail = l'identifiant fautif) ; na sans lien profond (l'accueil seul reste une preuve,
// couverte par checkMaicivyLink) ou si la liste des pages est indisponible.
func checkProjectLink(draft string, pages map[string]bool) MechanicalCheck {
	liens := lienProjet.FindAllStringSubmatch(draft, -1)
	if len(liens) == 0 || pages == nil {
		return MechanicalCheck{ID: CheckProjectLink, Status: CheckNA}
	}
	for _, l := range liens {
		id := strings.ToLower(l[1])
		if !pages[id] {
			return MechanicalCheck{ID: CheckProjectLink, Status: CheckFail, Detail: id}
		}
	}
	return MechanicalCheck{ID: CheckProjectLink, Status: CheckPass}
}

// checkNoMarkdown : aucune syntaxe markdown ne doit survivre dans un texte destiné au copier-coller.
// COMMENT : 1. marqueurs inline (gras, italique souligné, code), 2. marqueurs de début de ligne
// (citation, titre) qui ne comptent que s'ils ouvrent la ligne — un « # » en milieu de phrase est
// légitime, un « > » en tête de ligne ne l'est jamais dans un message de prospection.
func checkNoMarkdown(draft string) MechanicalCheck {
	for _, marker := range inlineMarkdownMarkers {
		if strings.Contains(draft, marker) {
			return MechanicalCheck{ID: CheckNoMarkdown, Status: CheckFail, Detail: marker}
		}
	}
	for _, line := range strings.Split(draft, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, ">") {
			return MechanicalCheck{ID: CheckNoMarkdown, Status: CheckFail, Detail: ">"}
		}
		if strings.HasPrefix(trimmed, "#") {
			return MechanicalCheck{ID: CheckNoMarkdown, Status: CheckFail, Detail: "#"}
		}
	}
	return MechanicalCheck{ID: CheckNoMarkdown, Status: CheckPass}
}

// checkSignature : le message se termine-t-il par le nom attendu ? Une signature manquante ou modifiée
// est invisible pour un opérateur qui ne lit pas la langue.
func checkSignature(draft, signature string) MechanicalCheck {
	if strings.TrimSpace(signature) == "" {
		return MechanicalCheck{ID: CheckSignature, Status: CheckNA}
	}
	lines := strings.Split(draft, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		last := strings.TrimSpace(lines[i])
		if last == "" {
			continue
		}
		if strings.EqualFold(last, strings.TrimSpace(signature)) {
			return MechanicalCheck{ID: CheckSignature, Status: CheckPass}
		}
		return MechanicalCheck{ID: CheckSignature, Status: CheckFail, Detail: last}
	}
	return MechanicalCheck{ID: CheckSignature, Status: CheckFail}
}

// checkRateCoherence : le VRAI risque sur le tarif n'est pas son absence mais la contradiction.
// POURQUOI pas « le TJM doit figurer dans le message » : en usage réel le tarif va dans un champ du
// formulaire de la plateforme, pas dans le texte — exiger sa présence recalerait un bon message. En
// revanche, si le message énonce un montant, il doit être celui arbitré : un modèle qui écrit 250 alors
// que le tarif retenu est 300 produit une contradiction que l'opérateur ne peut pas détecter en lisant.
// COMMENT : 1. extraire tous les montants énoncés, 2. si aucun ou si aucun tarif retenu → non
// applicable, 3. tout montant différent du tarif retenu → échec, en nommant le montant fautif.
func checkRateCoherence(draft string, retained int) MechanicalCheck {
	if retained <= 0 {
		return MechanicalCheck{ID: CheckRateCoherence, Status: CheckNA}
	}

	found := map[int]bool{}
	for _, re := range []*regexp.Regexp{rateWithCurrency, rateAfterKeyword} {
		for _, m := range re.FindAllStringSubmatch(draft, -1) {
			// Retire les séparateurs de milliers (espace fine ou insécable) avant conversion.
			raw := strings.NewReplacer(" ", "", " ", "").Replace(m[1])
			if v, err := strconv.Atoi(raw); err == nil {
				found[v] = true
			}
		}
	}

	if len(found) == 0 {
		return MechanicalCheck{ID: CheckRateCoherence, Status: CheckNA}
	}
	for amount := range found {
		if amount != retained {
			return MechanicalCheck{
				ID:     CheckRateCoherence,
				Status: CheckFail,
				Detail: fmt.Sprintf("%d ≠ %d", amount, retained),
			}
		}
	}
	return MechanicalCheck{ID: CheckRateCoherence, Status: CheckPass}
}

// checkLength : compté en RUNES et non en octets — un message français est plein d'accents, un
// compteur en octets annoncerait une longueur fausse à l'opérateur.
func checkLength(draft string, maxRunes int) MechanicalCheck {
	if maxRunes <= 0 {
		maxRunes = defaultMaxDraftRunes
	}
	n := len([]rune(draft))
	if n > maxRunes {
		return MechanicalCheck{
			ID:     CheckLength,
			Status: CheckFail,
			Detail: fmt.Sprintf("%d/%d", n, maxRunes),
		}
	}
	return MechanicalCheck{ID: CheckLength, Status: CheckPass, Detail: fmt.Sprintf("%d/%d", n, maxRunes)}
}
