package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"maicivy/internal/models"
)

// Vérification d'un draft pour l'écran /admin/compose : traduction opérateur + checklist.
//
// POURQUOI ce service est SÉPARÉ de l'agent de chat : l'opérateur ne lit pas le français, donc les
// deux seules choses qui lui permettent de faire confiance au message sont une traduction fidèle et
// une checklist qu'il n'a pas écrite lui-même. Si l'une ou l'autre était produite dans le contexte de
// la conversation, le modèle traduirait et jugerait son INTENTION plutôt que le texte réellement
// produit : une faute de rédaction resterait invisible précisément là où elle compte. Chaque appel
// d'ici part donc d'un contexte vierge, avec le strict minimum d'information.
//
// POURQUOI le front appelle ce service séparément plutôt que l'agent en interne : le draft s'affiche
// instantanément côté UI dès l'event tool_call (l'opérateur voit le message bouger en direct), et la
// vérification arrive juste après sans ralentir le tour de l'agent.

// textGenerator : sous-ensemble d'AIService dont ce service a besoin. POURQUOI une interface — pour
// tester les prompts (et surtout ce qui NE doit PAS s'y trouver) sans appeler l'API.
type textGenerator interface {
	GenerateTextWithModel(ctx context.Context, prompt, model string) (string, *models.AIMetrics, error)
}

// composeAuxModel : modèle des tâches auxiliaires. Traduire et confronter un texte court à une annonce
// sont des tâches simples et fréquentes (rejouées à CHAQUE version du draft) — Haiku suffit et garde le
// coût de l'écran raisonnable. La rédaction, elle, reste sur le modèle de l'agent.
const composeAuxModel = "claude-haiku-5-5"

type ComposeService struct {
	gen textGenerator
	// pages : projets qui ont une page publique — nourrit le contrôle des liens profonds
	// (checkProjectLink). nil = contrôle non applicable.
	pages PagesPortfolio
}

// SetPortfolioPages branche la liste des pages publiques sur le contrôle des liens de projet.
func (s *ComposeService) SetPortfolioPages(p PagesPortfolio) { s.pages = p }

func NewComposeService(gen textGenerator) *ComposeService {
	return &ComposeService{gen: gen}
}

// ComposeVerifyInput : état à vérifier. Announce et History sont fournis séparément à dessein — c'est
// le service qui décide lequel entre dans quel prompt, et History n'entre dans AUCUN.
type ComposeVerifyInput struct {
	Draft        string
	Signature    string
	RetainedRate int
	OperatorLang string // langue de l'opérateur (ISO 639-1) : cible de la traduction et des libellés
	Announce     string // annonce source. Vide = pas de jugement sémantique possible.
	// WantTitle : demander un nom d'atelier. Le front ne le met à vrai qu'une fois, tant que l'atelier
	// n'a pas de nom.
	//
	// POURQUOI ce n'est PAS automatique à chaque vérification : d'abord ça coûterait un appel modèle
	// par tour d'ajustement, ensuite et surtout le nom CHANGERAIT sous les yeux de l'opératrice à
	// chaque retouche du message — or c'est précisément ce nom qui lui sert de repère pour retrouver
	// l'atelier dans sa liste. Un repère qui bouge n'en est plus un.
	WantTitle bool
	// History : présent dans la structure UNIQUEMENT pour rendre explicite qu'il ne doit fuiter dans
	// aucun prompt (verrouillé par test). Ne jamais l'injecter.
	History string
}

// SemanticCheck : item de checklist dérivé de l'annonce, évalué par un juge en contexte vierge. Séparé
// des contrôles mécaniques dans la réponse pour que l'UI puisse les afficher différemment — un jugement
// n'a pas la même valeur qu'une preuve, et l'opérateur doit voir la différence.
type SemanticCheck struct {
	ID     string      `json:"id"`
	Label  string      `json:"label"` // rédigé dans la langue de l'opérateur par le juge
	Status CheckStatus `json:"status"`
}

type ComposeVerifyResult struct {
	Translation string            `json:"translation"`
	Mechanical  []MechanicalCheck `json:"mechanical"`
	Semantic    []SemanticCheck   `json:"semantic"`
	// Title : nom de l'atelier, présent uniquement si WantTitle était vrai ET que le nommage a abouti.
	// Vide sinon — l'appelant garde alors le nom qu'il avait.
	Title string `json:"title,omitempty"`
}

// portfolioPages : la liste des pages publiques, ou nil si non branchée / indisponible.
func (s *ComposeService) portfolioPages(ctx context.Context) map[string]bool {
	if s.pages == nil {
		return nil
	}
	return s.pages(ctx)
}

// composeTitleMaxRunes : un titre plus long ne tient pas sur une ligne de la liste et n'aide plus à
// se repérer. Borné ICI parce que la valeur est PERSISTÉE (colonne varchar(200)) : couper à
// l'affichage laisserait passer une écriture trop longue en base. Compté en RUNES et non en octets —
// un titre chinois de 60 caractères ferait 180 octets.
const composeTitleMaxRunes = 60

// Verify produit la traduction et les deux niveaux de checklist.
// COMMENT : 1. contrôles mécaniques en local (aucun appel réseau, donc jamais perdus), 2. traduction
// en contexte vierge, 3. jugement sémantique seulement s'il y a une annonce à confronter. Les échecs
// des étapes 2 et 3 sont journalisés mais NON fatals : perdre la checklist mécanique parce qu'un appel
// réseau a échoué serait absurde — c'est la garantie la plus solide des trois.
func (s *ComposeService) Verify(ctx context.Context, in ComposeVerifyInput) (ComposeVerifyResult, error) {
	res := ComposeVerifyResult{
		Mechanical: RunMechanicalChecks(ComposeCheckInput{
			Draft:          in.Draft,
			Signature:      in.Signature,
			RetainedRate:   in.RetainedRate,
			PortfolioPages: s.portfolioPages(ctx),
		}),
	}

	lang := in.OperatorLang
	if lang == "" {
		lang = "en"
	}

	if translation, err := s.translate(ctx, in.Draft, lang); err != nil {
		log.Warn().Err(err).Msg("compose: traduction indisponible, checklist mécanique conservée")
	} else {
		res.Translation = translation
	}

	if strings.TrimSpace(in.Announce) != "" {
		if semantic, err := s.judge(ctx, in.Draft, in.Announce, lang); err != nil {
			log.Warn().Err(err).Msg("compose: jugement sémantique indisponible")
		} else {
			res.Semantic = semantic
		}
	}

	// Nommage en DERNIER et non fatal, comme les deux étapes précédentes : c'est un confort de
	// navigation, pas une garantie sur le message. Un atelier sans nom reste parfaitement utilisable.
	if in.WantTitle {
		if title, err := s.title(ctx, in.Draft, in.Announce, lang); err != nil {
			log.Warn().Err(err).Msg("compose: nommage indisponible, atelier sans titre")
		} else {
			res.Title = title
		}
	}

	return res, nil
}

// translate : traduction LITTÉRALE du draft. Le prompt ne contient QUE le draft — pas l'annonce, pas
// la conversation. C'est ce qui en fait un contrôle : l'opérateur lit ce qui est écrit, pas ce que
// l'agent voulait écrire.
func (s *ComposeService) translate(ctx context.Context, draft, lang string) (string, error) {
	prompt := fmt.Sprintf(`TRADUCTION LITTÉRALE.

Traduis le texte ci-dessous en %s. Rends le SENS EXACT de ce qui est écrit, y compris les maladresses,
les répétitions et les erreurs éventuelles — ne corrige rien, n'améliore rien, n'ajoute rien. Cette
traduction sert à contrôler le texte, pas à le vendre.

Réponds UNIQUEMENT avec la traduction, sans commentaire ni préambule.

--- TEXTE ---
%s
--- FIN ---`, lang, draft)

	out, _, err := s.gen.GenerateTextWithModel(ctx, prompt, composeAuxModel)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// judge : confronte le draft à l'annonce et renvoie une checklist d'exigences.
// POURQUOI contexte vierge ici aussi : le juge ne doit connaître ni la conversation ni le raisonnement
// de l'agent. Il ne voit que ce qu'un lecteur verrait — l'annonce et le message.
// COMMENT : sortie JSON stricte, libellés dans la langue de l'opérateur (c'est lui qui les lit), et
// tolérance au bavardage du modèle en extrayant le tableau JSON de la réponse.
func (s *ComposeService) judge(ctx context.Context, draft, announce, lang string) ([]SemanticCheck, error) {
	prompt := fmt.Sprintf(`Tu contrôles un message de candidature face à l'annonce à laquelle il répond.
Tu n'as PAS écrit ce message et tu ne connais rien de son auteur au-delà de ce qui suit.

Liste les exigences CONCRÈTES et VÉRIFIABLES de l'annonce, puis dis pour chacune si le message y répond.
Maximum 6 items. Ignore ce qui relève du goût ou du style : uniquement ce qu'un lecteur peut trancher.

Réponds UNIQUEMENT par un tableau JSON, sans texte autour :
[{"id":"slug_court","label":"formulation en %s, max 60 caractères","status":"pass"|"fail"}]

--- ANNONCE ---
%s
--- MESSAGE ---
%s
--- FIN ---`, lang, announce, draft)

	out, _, err := s.gen.GenerateTextWithModel(ctx, prompt, composeAuxModel)
	if err != nil {
		return nil, err
	}

	// Le modèle encadre parfois le JSON de texte ; on isole le tableau plutôt que d'échouer.
	start, end := strings.Index(out, "["), strings.LastIndex(out, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("réponse du juge sans tableau JSON")
	}

	var checks []SemanticCheck
	if err := json.Unmarshal([]byte(out[start:end+1]), &checks); err != nil {
		return nil, fmt.Errorf("juge: JSON invalide: %w", err)
	}

	// Un statut inattendu est ramené à « non applicable » : afficher un item sans état lisible serait
	// pire que de l'afficher comme non tranché.
	for i := range checks {
		if checks[i].Status != CheckPass && checks[i].Status != CheckFail {
			checks[i].Status = CheckNA
		}
	}
	return checks, nil
}

// title : nomme l'atelier pour que l'opératrice le retrouve dans sa liste.
//
// POURQUOI un appel modèle plutôt qu'un découpage du texte de l'annonce : l'annonce est SOUVENT une
// capture d'écran, auquel cas il n'y a aucun texte à découper — le draft, lui, existe toujours. Et
// une heuristique du genre « première ligne de l'annonce » produit des titres du type « Nouvelle
// opportunité » sur toutes les lignes, ce qui ne distingue rien.
//
// POURQUOI dans la langue de l'OPÉRATRICE : ce titre est le seul texte de la liste qu'elle doit
// pouvoir lire pour choisir. Le message, lui, reste en français — ce sont deux publics différents.
//
// POURQUOI contexte vierge ici aussi : cohérence avec la traduction et le juge. Un nom n'a pas
// besoin de la conversation, et ne pas la passer évite qu'un titre reprenne une formule de l'agent
// plutôt que le sujet réel.
func (s *ComposeService) title(ctx context.Context, draft, announce, lang string) (string, error) {
	source := announce
	if strings.TrimSpace(source) == "" {
		// Pas d'annonce exploitable (capture d'écran) : on nomme depuis le message produit, qui parle
		// forcément de l'entreprise et du poste.
		source = draft
	}

	prompt := fmt.Sprintf(`TITRE D'ARCHIVE.

Donne un nom court à ce dossier de candidature, pour qu'on le retrouve dans une liste.

Format : "Entreprise — poste ou sujet". Si l'entreprise est inconnue, décris le sujet.
Langue : %s. Les noms propres (entreprise, technologies) restent tels quels.
Maximum %d caractères. Pas de guillemets, pas de point final.

Réponds UNIQUEMENT par le titre.

--- DOSSIER ---
%s
--- FIN ---`, lang, composeTitleMaxRunes, source)

	out, _, err := s.gen.GenerateTextWithModel(ctx, prompt, composeAuxModel)
	if err != nil {
		return "", err
	}

	// Le modèle encadre parfois sa réponse de guillemets malgré la consigne.
	titre := strings.TrimSpace(out)
	titre = strings.Trim(titre, "\"'«»")
	titre = strings.TrimSpace(titre)
	if titre == "" {
		return "", fmt.Errorf("nommage: réponse vide")
	}

	// Coupe sur les RUNES : trancher des octets couperait un caractère accentué ou chinois en deux et
	// produirait une chaîne invalide en base.
	if r := []rune(titre); len(r) > composeTitleMaxRunes {
		titre = strings.TrimSpace(string(r[:composeTitleMaxRunes]))
	}
	return titre, nil
}
