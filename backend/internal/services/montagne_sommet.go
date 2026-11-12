package services

import (
	"sort"
	"strings"
)

// ============================================================================
// Sommet de la montagne — quatre œuvres en grandes cartes, chacune avec SES preuves.
// ----------------------------------------------------------------------------
// QUOI : la liste EXPLICITE des cartes qui coiffent la montagne, dans leur ordre d'affichage
// (de gauche à droite sur grand écran, deux par rangée sur téléphone). Pour chaque carte : l'id
// du projet et la RÈGLE qui dit où sont ses preuves. Une carte ne montre pas « une capture » :
// elle montre ce qui prouve que le projet existe et sert — les jeux bâtis sur un moteur, une
// démo filmée, le monde généré, le site lui-même.
//
// POURQUOI une liste écrite à la main et pas un calcul : le sommet est une décision éditoriale
// d'Alexis (ce qu'on voit en premier), pas une mesure. Changer le sommet = changer cette liste
// (et son test). Les PREUVES, elles, ne sont jamais écrites à la main : chaque règle les lit dans
// les données (liens « sert à construire », visuels de maiProFiles) — sauf la règle `site`, dont
// les preuves sont des pages de ce site-ci, captures rangées côté front.
//
// COMMENT :
//   - `construireSommet` applique la règle de chaque carte présente (un projet retiré de
//     maiProFiles disparaît du sommet au lieu d'y laisser une carte vide).
//   - Chaque règle range ses preuves de la meilleure à la moins bonne : c'est l'ordre de la fleur
//     de la carte. La COUVERTURE de la carte n'est plus une preuve (v4) : c'est une illustration
//     d'ambiance, choisie côté front (frontend/lib/montagne/art.ts).
//   - `members` : projets ABSORBÉS par la carte (un étage de Theomen qui a des preuves). Ils ne
//     sont plus des tuiles — sinon on les verrait deux fois — et leurs filets partent de la carte.
//     Un étage sans preuve n'est pas absorbé : il reste une tuile, seule place où on le voit.
// ============================================================================

// RegleSommet — d'où une carte tire ses preuves. Valeur servie au front (`rule`).
type RegleSommet string

const (
	// RegleJeuxConstruits — les JEUX que le projet sert à construire (liens sortants réels,
	// montagne_liens.go), chacun par son meilleur vrai visuel. Pour un moteur.
	RegleJeuxConstruits RegleSommet = "built_games"
	// RegleVideo — la démo vidéo du projet (déclarée dans book.yaml, sinon le store).
	RegleVideo RegleSommet = "video"
	// RegleEtages — une chaîne de projets, un étage chacun (Theomen génère le monde global,
	// Emergika en dérive le terrain local) : les vrais visuels de chaque étage, dans l'ordre.
	RegleEtages RegleSommet = "stages"
	// RegleSite — des pages de CE site (blog, statistiques git, chat, CV) : le projet est le site.
	RegleSite RegleSommet = "site"
)

// Étages d'une chaîne (RegleEtages) — valeurs servies au front, qui en tire la légende i18n.
const (
	EtageGlobal = "global"
	EtageLocal  = "local"
)

type etageSommet struct {
	Projet string
	Etage  string
}

// pageSite — une page de ce site prouvant le projet maicivy. `Chemin` est relatif à la langue
// (le front préfixe /[locale]) ; la capture correspondante est un fichier du front
// (public/montagne/maicivy/<page>-{mobile,desktop}.webp), prise sur la prod.
type pageSite struct {
	Page   string
	Chemin string
}

type carteSommet struct {
	ID     string
	Regle  RegleSommet
	Etages []etageSommet // RegleEtages seulement
	Pages  []pageSite    // RegleSite seulement
}

// sommetMontagne — décision d'Alexis du 2026-10-31 (v3 : « chaque carte montre ses preuves »).
var sommetMontagne = []carteSommet{
	{ID: "grove_engine", Regle: RegleJeuxConstruits},
	{ID: "melodicode", Regle: RegleVideo},
	{ID: "theomen", Regle: RegleEtages, Etages: []etageSommet{
		{Projet: "theomen", Etage: EtageGlobal},
		{Projet: "emergika", Etage: EtageLocal},
	}},
	{ID: "maicivy", Regle: RegleSite, Pages: []pageSite{
		{Page: "blog", Chemin: "/blog"},
		{Page: "gitstats", Chemin: "/gitstats"},
		{Page: "chat", Chemin: "/chat"},
		{Page: "cv", Chemin: "/cv"},
	}},
}

// MontagnePreuve — une preuve affichée par une carte du sommet. Les champs du visuel (kind, url,
// poster_url, vector, alt) sont à plat ; `kind` vaut "page" pour une page du site (url vide).
type MontagnePreuve struct {
	MontagneVisuel
	// ProjectID / ProjectName / Curated — le projet qui PORTE la preuve (un jeu bâti sur le
	// moteur, un étage de la chaîne) : la galerie le nomme et mène à sa page s'il en a une.
	ProjectID   string `json:"project_id,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
	Curated     bool   `json:"curated,omitempty"`
	Stage       string `json:"stage,omitempty"` // RegleEtages : global | local
	Page        string `json:"page,omitempty"`  // RegleSite : identifiant de la page
	Path        string `json:"path,omitempty"`  // RegleSite : chemin sans la langue
}

// MontagneCarte — une carte du sommet telle que servie.
type MontagneCarte struct {
	ID      string           `json:"id"`
	Rule    RegleSommet      `json:"rule"`
	Proofs  []MontagnePreuve `json:"proofs"`  // jamais nil ; de la meilleure à la moins bonne
	Members []string         `json:"members"` // jamais nil ; projets absorbés (cf. en-tête)
}

// construireSommet — les cartes présentes, dans l'ordre de sommetMontagne, avec leurs preuves.
// `categories` : catégorie maiProFiles (anglaise) de chaque projet, pour reconnaître un jeu.
func construireSommet(noeuds []MontagneNoeud, liens []MontagneLien, categories map[string]string) []MontagneCarte {
	parID := make(map[string]MontagneNoeud, len(noeuds))
	for _, n := range noeuds {
		parID[n.ID] = n
	}
	out := make([]MontagneCarte, 0, len(sommetMontagne))
	for _, c := range sommetMontagne {
		noeud, ok := parID[c.ID]
		if !ok {
			continue
		}
		carte := MontagneCarte{ID: c.ID, Rule: c.Regle, Proofs: []MontagnePreuve{}, Members: []string{}}
		switch c.Regle {
		case RegleJeuxConstruits:
			carte.Proofs = preuvesJeuxConstruits(c.ID, parID, liens, categories)
		case RegleVideo:
			for _, v := range noeud.Visuals {
				if v.Kind == "video" {
					carte.Proofs = append(carte.Proofs, MontagnePreuve{MontagneVisuel: v, ProjectID: c.ID})
					break
				}
			}
		case RegleEtages:
			for _, e := range c.Etages {
				etage, ok := parID[e.Projet]
				if !ok {
					continue
				}
				avant := len(carte.Proofs)
				for _, v := range vraisVisuels(etage) {
					carte.Proofs = append(carte.Proofs, preuveDuProjet(v, etage, e.Etage))
				}
				// Un étage n'est absorbé que s'il a apporté une preuve (cf. en-tête).
				if e.Projet != c.ID && len(carte.Proofs) > avant {
					carte.Members = append(carte.Members, e.Projet)
				}
			}
		case RegleSite:
			for _, p := range c.Pages {
				carte.Proofs = append(carte.Proofs, MontagnePreuve{
					MontagneVisuel: MontagneVisuel{Kind: "page"}, ProjectID: c.ID, Page: p.Page, Path: p.Chemin,
				})
			}
		}
		out = append(out, carte)
	}
	return out
}

// preuvesJeuxConstruits — RegleJeuxConstruits. Les cibles des liens SORTANTS du projet qui sont
// des jeux (estJeu) et ont un vrai visuel (vraisVisuels), UNE preuve par jeu (son meilleur
// visuel : le premier, l'image principale du livre passant avant la galerie).
// Ordre = « meilleure capture d'abord » : le jeu le plus abouti (rangMaturite), puis le lien le
// plus fort (stack avant description), puis l'id — déterministe.
func preuvesJeuxConstruits(id string, parID map[string]MontagneNoeud, liens []MontagneLien, categories map[string]string) []MontagnePreuve {
	type candidat struct {
		noeud MontagneNoeud
		kind  string
	}
	var jeux []candidat
	for _, l := range liens {
		if l.From != id {
			continue
		}
		cible, ok := parID[l.To]
		if !ok || !estJeu(categories[l.To]) || len(vraisVisuels(cible)) == 0 {
			continue
		}
		jeux = append(jeux, candidat{cible, l.Kind})
	}
	sort.SliceStable(jeux, func(i, j int) bool {
		a, b := jeux[i], jeux[j]
		if ra, rb := rangMaturite(a.noeud.Status), rangMaturite(b.noeud.Status); ra != rb {
			return ra < rb
		}
		if (a.kind == lienStack) != (b.kind == lienStack) {
			return a.kind == lienStack
		}
		return a.noeud.ID < b.noeud.ID
	})
	out := make([]MontagnePreuve, 0, len(jeux))
	for _, j := range jeux {
		out = append(out, preuveDuProjet(vraisVisuels(j.noeud)[0], j.noeud, ""))
	}
	return out
}

func preuveDuProjet(v MontagneVisuel, n MontagneNoeud, etage string) MontagnePreuve {
	return MontagnePreuve{MontagneVisuel: v, ProjectID: n.ID, ProjectName: n.Name, Curated: n.Curated, Stage: etage}
}

// estJeu — la catégorie maiProFiles commence par un segment de jeu (« Game / C++ engine »,
// « Game development / C++ / Prototype »). Le PREMIER segment seulement : c'est la nature du
// projet ; les suivants disent sa technique (« C++ engine / … » n'est pas un jeu).
func estJeu(categorie string) bool {
	tete, _, _ := strings.Cut(categorie, "/")
	return strings.Contains(strings.ToLower(tete), "game")
}

// marqueursSchema — mots qui, dans le nom de fichier ou le texte alternatif d'une image,
// désignent un SCHÉMA et non une capture du produit. Mesuré le 2026-10-31 sur les visuels réels :
// « drifterra_architecture.jpg », « daos_architecture.jpg », « melodicode_architecture.jpg »,
// « GroveLifeCycle », « GroveDevWorkFlow », « MaicivySystemOverview ».
var marqueursSchema = []string{"architecture", "diagram", "schema", "schéma", "overview", "workflow", "lifecycle"}

// vraisVisuels — les images d'un projet qui MONTRENT le produit : ni vidéo, ni dessin vectoriel,
// ni schéma (marqueursSchema). Ordre des pétales conservé (image principale du livre d'abord).
func vraisVisuels(n MontagneNoeud) []MontagneVisuel {
	var out []MontagneVisuel
	for _, v := range n.Visuals {
		if v.Kind != "image" || v.Vector || estSchema(v) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func estSchema(v MontagneVisuel) bool {
	texte := strings.ToLower(cleVisuel(v.URL) + " " + v.Alt)
	for _, m := range marqueursSchema {
		if strings.Contains(texte, m) {
			return true
		}
	}
	return false
}

// rangMaturite — plus petit = plus abouti. Un statut inconnu passe après tous les connus.
func rangMaturite(statut string) int {
	switch statut {
	case "production":
		return 0
	case "active":
		return 1
	case "wip":
		return 2
	case "R&D":
		return 3
	case "retired":
		return 4
	}
	return 5
}
