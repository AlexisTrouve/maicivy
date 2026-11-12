package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrPortfolioAbsent — le projet demandé n'est pas dans la curation du portfolio.
//
// POURQUOI une sentinelle plutôt qu'un test de chaîne chez l'appelant : le client générique rend
// « not found: <path> » et rien ne garantit cette formulation dans le temps. On traduit une fois,
// ici, pour que le handler HTTP puisse distinguer « absent de la curation » (404 légitime, le
// portfolio est curé) de « maiProFiles est tombé » (502).
var ErrPortfolioAbsent = errors.New("projet absent de la curation du portfolio")

// Accès au portfolio curé de maiProFiles.
//
// CONTEXTE : `data/book.yaml` côté maiProFiles est la couche de CURATION du portfolio —
// 25 projets choisis, avec leur récit, leur image, leurs badges et leurs chiffres. C'est la même
// source qui produit le PDF du portfolio book. maicivy en rend les pages web ; le livre en rend
// le PDF. Ni l'un ni l'autre ne recopie la donnée.

// langsLivre : éditions RÉELLEMENT curées dans book.yaml.
//
// ⚠️ Ce n'est PAS `validLangs`. Le contenu général de maiProFiles existe en fr/en/ka ; le livre,
// lui, est curé en en/fr/zh. Passer par la table générale replierait le chinois sur l'anglais et
// ferait disparaître une édition qui existe pourtant, traduite.
var langsLivre = map[string]bool{"en": true, "fr": true, "zh": true}

// langueDuLivre replie sur l'anglais toute langue non curée — jamais sur le français, règle
// établie pour tout le contenu maiProFiles : un visiteur de/it ne doit pas tomber sur du FR.
func langueDuLivre(lang string) string {
	if langsLivre[lang] {
		return lang
	}
	return fallbackLang
}

// MPFImage — une image de la curation. `src` est le chemin disque dont le rendu PDF se sert ;
// `url` est la route servie par l'API, la seule utilisable depuis le web.
type MPFImage struct {
	Src      string `json:"src"`
	Alt      string `json:"alt"`
	Portrait bool   `json:"portrait"`
	URL      string `json:"url"`
}

// MPFBadge — une pastille de stack, avec sa classe de couleur décidée à la curation.
type MPFBadge struct {
	Label string `json:"label"`
	Class string `json:"class"`
}

// MPFStat — un chiffre mis en avant ("33 000+" / " lignes Rust").
type MPFStat struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// MPFGalleryImage — un visuel de la galerie web d'un projet.
//
// La page A4 du livre n'en tient qu'un (`Image`) ; le web les rend tous. Chaque entrée est déjà
// résolue dans la langue demandée côté maiProFiles : une capture d'interface est commune à toutes
// les éditions, un diagramme légendé peut avoir sa variante.
type MPFGalleryImage struct {
	Src      string `json:"src"`
	Alt      string `json:"alt"`
	Caption  string `json:"caption"`
	URL      string `json:"url"`
	Portrait bool   `json:"portrait"`
}

// MPFVideo — la démo vidéo d'un projet, déclarée dans book.yaml côté maiProFiles.
//
// QUOI : le fichier (`url`), l'image affichée avant lecture (`poster_url`) et la légende, déjà
// résolue dans la langue demandée. POURQUOI hors de la galerie : un lecteur n'est pas une image,
// et le front n'aurait sinon que l'extension pour deviner le type. Les fichiers vivent dans le store
// d'images de maiProFiles (`/images/<id>`), qui répond 206 aux requêtes Range — sans ça Safari
// refuse de lire et la barre de progression ne permet pas de se déplacer.
type MPFVideo struct {
	Caption   string `json:"caption"`
	URL       string `json:"url"`
	PosterURL string `json:"poster_url"`
}

// MPFBookProject — une entrée du livre, telle qu'assemblée par maiProFiles.
type MPFBookProject struct {
	Num         int      `json:"num"`
	Anchor      string   `json:"anchor"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	IndexName   string   `json:"index_name"`
	IndexDomain string   `json:"index_domain"`
	IndexStack  string   `json:"index_stack"`
	Image       MPFImage `json:"image"`
	// Images : la galerie. Présente parce que le client demande `web=true` ; absente du modèle
	// par défaut, qui doit rester byte-compatible avec le rendu du PDF.
	Images []MPFGalleryImage `json:"images"`
	// Video : la démo vidéo, nil quand le projet n'en déclare pas (le front n'affiche alors rien).
	// Même raison que la galerie pour n'exister que sur la vue web : un PDF ne lit pas de vidéo.
	Video      *MPFVideo  `json:"video"`
	Paragraphs []string   `json:"paragraphs"`
	Badges     []MPFBadge `json:"badges"`
	Stats      []MPFStat  `json:"stats"`
}

// MPFBook — une édition complète du portfolio.
type MPFBook struct {
	Lang     string           `json:"lang"`
	Title    string           `json:"title"`
	Projects []MPFBookProject `json:"projects"`
}

// MPFPortfolioPage — tout ce qu'il faut pour une page projet : le récit curé ET la fiche
// technique, plus les liens, le statut et la galerie.
type MPFPortfolioPage struct {
	MPFBookProject
	DescriptionPortfolio string `json:"description_portfolio"`
	// Rendu HTML de la fiche technique. POURQUOI côté serveur : le front n'a pas de moteur
	// markdown et en ajouter un pour ça seul serait une dépendance de plus, alors que goldmark
	// est déjà là pour le blog. Même contrat que `content_html` côté articles.
	DescriptionPortfolioHTML string              `json:"description_portfolio_html"`
	Links                    map[string]string   `json:"links"`
	Status                   string              `json:"status"`
	Tags                     []string            `json:"tags"`
	Gallery                  []MPFPortfolioImage `json:"gallery"`
}

// MPFPortfolioImage — une image du store, rattachée à un projet.
//
// Distincte de l'image du livre : celle-ci vit dans le store d'images de maiProFiles et se sert
// par `/images/<id>`. La page A4 du PDF n'en tient qu'une ; le web n'a pas cette contrainte, donc
// toutes les images rattachées à un projet sont rendues.
type MPFPortfolioImage struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Alt      string `json:"alt"`
	LinkedTo struct {
		// Section : hero, screenshot, architecture, diagram, demo — sert de légende.
		Section string `json:"section"`
	} `json:"linked_to"`
}

// GetPortfolioBook récupère l'édition `lang` du portfolio.
//
// COMMENT : `web=true` est indispensable — sans lui la réponse ne porte ni `id` ni `image.url`,
// donc le site ne saurait ni router vers une fiche ni afficher une capture. Ce paramètre est
// absent par défaut côté API pour une raison précise : le rendu du PDF décode avec
// `DisallowUnknownFields` et un champ en plus le fait échouer.
func (c *MaiProFilesClient) GetPortfolioBook(ctx context.Context, lang string) (*MPFBook, error) {
	var livre MPFBook
	// La langue est résolue ici et passée telle quelle : le langSuffix générique appliquerait
	// la table du contenu courant (fr/en/ka) et perdrait l'édition chinoise.
	path := fmt.Sprintf("/book?web=true&lang=%s", langueDuLivre(lang))
	if err := c.get(ctx, path, "", &livre); err != nil {
		return nil, fmt.Errorf("portfolio: lecture du livre (%s) : %w", lang, err)
	}
	return &livre, nil
}

// GetPortfolioPage récupère la page complète d'un projet curé.
// Une erreur remonte si le projet n'est pas dans la curation : le portfolio est CURÉ, on
// n'affiche pas une page à moitié vide pour un projet qui n'y a pas sa place.
func (c *MaiProFilesClient) GetPortfolioPage(ctx context.Context, id, lang string) (*MPFPortfolioPage, error) {
	var page MPFPortfolioPage
	path := fmt.Sprintf("/portfolio/%s?lang=%s", id, langueDuLivre(lang))
	if err := c.get(ctx, path, "", &page); err != nil {
		if strings.HasPrefix(err.Error(), "not found:") {
			return nil, fmt.Errorf("%w : %s", ErrPortfolioAbsent, id)
		}
		return nil, fmt.Errorf("portfolio: lecture de la fiche %q (%s) : %w", id, lang, err)
	}
	page.DescriptionPortfolioHTML = markdownToHTML(page.DescriptionPortfolio)
	return &page, nil
}
