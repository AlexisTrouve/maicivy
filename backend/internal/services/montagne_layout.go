package services

import (
	"hash/fnv"
	"math"
	"sort"
)

// ============================================================================
// Disposition de la montagne — positions 3D précalculées, sans aucun hasard.
// ----------------------------------------------------------------------------
// QUOI : place chaque projet en (x, y, z). y est la hauteur (fondations en bas, œuvres en haut),
// (x, z) le plan horizontal. Le front ne calcule AUCUNE position : il projette et dessine.
//
// POURQUOI côté serveur et déterministe : la même montagne pour tous les visiteurs, à chaque
// visite, et un rendu qui ne « tremble » pas au chargement. Le calcul tourne une fois par
// rafraîchissement du cache (10 min), pas par visite.
//
// POURQUOI des projets SUR LES FLANCS d'un cône, en anneaux (refonte du 2026-10-30) : la première
// version remplissait le VOLUME d'une silhouette par un modèle de forces. Mesuré en vue
// d'ensemble : les 39 œuvres se projetaient en une pelote, les étiquettes se chevauchaient par
// dizaines et la forme ne se lisait plus — un nuage, pas une montagne. Posés sur la surface d'un
// cône, les projets DESSINENT la silhouette : vue de n'importe quel côté, c'est une pente qui monte
// des fondations au sommet. Chaque anneau est une courbe de niveau ; deux strates sont séparées
// par un vide vertical plus grand que deux anneaux d'une même strate.
//
// COMMENT :
//  1. Forme — un cône de pente `penteMontagne` : rayon = pente × profondeur sous le sommet.
//  2. Anneaux, du SOMMET vers le pied : la strate du haut commence au sommet (profondeur 0),
//     chaque anneau suivant descend de `ecartAnneaux`, et l'on ajoute des anneaux jusqu'à ce que
//     leur capacité cumulée loge toute la strate. La capacité d'un anneau = sa circonférence
//     divisée par `espacementArc` : deux voisins d'un même anneau ne sont jamais plus près que
//     cela, quel que soit le nombre de projets. Entre deux strates, `ecartStrates`.
//  3. Répartition — chaque anneau reçoit au moins un projet, le reste au prorata de la capacité
//     (plus grands restes). Les projets d'une strate, triés par (degré décroissant, id), vont de
//     l'anneau du haut à celui du bas : le SOMMET est l'œuvre la plus reliée.
//  4. Angles, anneau par anneau du PIED vers le sommet : un projet vise l'angle moyen (moyenne
//     circulaire) de ses voisins déjà placés — ce qui sert à le construire, plus bas — ou, à
//     défaut, un angle tiré d'un HACHAGE de son id (FNV-1a, pas un aléa). Les projets de l'anneau
//     sont triés par angle visé et posés sur des places équidistantes ; le décalage de l'anneau
//     est la moyenne circulaire des écarts (placerAnneau). Un lien monte donc la face de la
//     montagne au lieu de la traverser.
//  5. Arrondi à 3 décimales : le JSON est stable octet pour octet d'un calcul à l'autre.
// ============================================================================

const (
	// penteMontagne — rayon / profondeur sous le sommet. Mesuré le 2026-10-30 sur les 75 projets :
	// la hauteur ne dépend presque pas de la pente (12,1 unités de 0,75 à 0,95 ; les œuvres
	// demandent toujours 7 anneaux), la largeur si. Sur un écran 16:10 la montagne est limitée par
	// sa HAUTEUR : élargir les anneaux donne plus de place à chaque étiquette sans rapetisser le
	// dessin. À 0,62 (largeur/hauteur 1,24) le sommet s'entassait sur 650 px de large ; 0,85
	// (1,70) reste lisible en portrait, où la largeur limite.
	penteMontagne = 0.85
	// espacementArc — distance minimale (monde) entre deux voisins d'un anneau. Une étiquette fait
	// ~2 unités de large à l'échelle de la vue d'ensemble desktop : en dessous, deux voisins de
	// face se chevauchent à coup sûr.
	espacementArc = 2.3
	// ecartAnneaux — dénivelé entre deux anneaux d'une même strate : icône + étiquette à l'écran.
	ecartAnneaux = 1.0
	// ecartStrates — dénivelé entre deux strates, nettement plus grand : les couches se lisent.
	ecartStrates = 1.7
	angleTour    = 2 * math.Pi
)

// PositionNoeud — coordonnées monde d'un projet.
type PositionNoeud struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// MontagneStrate — une bande verticale, telle que le front la dessine et la survole.
type MontagneStrate struct {
	ID    Strate  `json:"id"`
	Level int     `json:"level"`
	YMin  float64 `json:"y_min"`
	YMax  float64 `json:"y_max"`
	// Radius — rayon du cône au PIED de la bande (son anneau le plus large). Le front s'en sert
	// pour cadrer la caméra sur une strate.
	Radius float64 `json:"radius"`
	Count  int     `json:"count"`
}

// MontagneForme — le cône sur lequel reposent les projets : le front en dessine la silhouette et
// les courbes de niveau. Rayon à la hauteur y = Slope × (ApexY − y).
type MontagneForme struct {
	ApexY float64 `json:"apex_y"`
	Slope float64 `json:"slope"`
}

// hacheUnitaire — FNV-1a de l'id ramené dans [0, 1). Remplace un aléa : même id, même valeur.
func hacheUnitaire(id string) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	return float64(h.Sum64()%1_000_003) / 1_000_003
}

// arrondi3 — étape 5.
func arrondi3(v float64) float64 { return math.Round(v*1000) / 1000 }

// capaciteAnneau — combien de projets tiennent sur l'anneau à cette profondeur sous le sommet
// (étape 2) : le plus grand n dont la CORDE entre deux voisins, 2r·sin(π/n), reste ≥
// espacementArc. La corde et non l'arc : sur un petit anneau l'arc surestime la place (mesuré :
// 3 projets sur un rayon de 1,24 tombaient à 2,15 d'écart). Toujours au moins 1 : le sommet
// (rayon nul) porte un projet.
func capaciteAnneau(profondeur float64) int {
	r := penteMontagne * profondeur
	n := 1
	for 2*r*math.Sin(math.Pi/float64(n+1)) >= espacementArc {
		n++
	}
	return n
}

// repartir — étape 3 : `n` projets sur des anneaux de capacités `caps` (somme ≥ n, len ≤ n).
// Un projet par anneau d'abord, puis le reste au prorata de (capacité − 1), méthode des plus
// grands restes ; à reste égal, l'anneau le plus haut (indice le plus petit) passe en premier.
func repartir(n int, caps []int) []int {
	out := make([]int, len(caps))
	libre := 0
	for i, c := range caps {
		out[i] = 1
		libre += c - 1
	}
	reste := n - len(caps)
	if reste <= 0 || libre == 0 {
		return out
	}
	type frac struct {
		i int
		f float64
	}
	var fracs []frac
	donne := 0
	for i, c := range caps {
		q := float64(reste) * float64(c-1) / float64(libre)
		out[i] += int(math.Floor(q))
		donne += int(math.Floor(q))
		fracs = append(fracs, frac{i, q - math.Floor(q)})
	}
	sort.SliceStable(fracs, func(a, b int) bool { return fracs[a].f > fracs[b].f })
	for k := 0; donne < reste; k++ {
		out[fracs[k].i]++
		donne++
	}
	return out
}

// moyenneCirculaire — angle moyen de vecteurs unitaires ; ok=false si la moyenne est indéfinie
// (liste vide, ou vecteurs qui s'annulent).
func moyenneCirculaire(angles []float64) (float64, bool) {
	var sx, sy float64
	for _, a := range angles {
		sx += math.Cos(a)
		sy += math.Sin(a)
	}
	if math.Hypot(sx, sy) < 1e-9 {
		return 0, false
	}
	return math.Atan2(sy, sx), true
}

// placerAnneau — étape 4 : `vises` = angles visés des projets d'un anneau, déjà TRIÉS par angle
// croissant (ordre cyclique). Rend l'angle de chaque projet : places équidistantes, décalage
// commun = moyenne circulaire de (visé_i − 2πi/n). Ce décalage minimise l'écart global entre
// visée et place, et le résultat ne dépend pas du projet qui ouvre l'ordre cyclique (changer de
// premier décale tous les indices et le décalage d'autant).
func placerAnneau(vises []float64) []float64 {
	n := len(vises)
	pas := angleTour / float64(n)
	ecarts := make([]float64, n)
	for i, v := range vises {
		ecarts[i] = v - pas*float64(i)
	}
	decalage, _ := moyenneCirculaire(ecarts)
	out := make([]float64, n)
	for i := range vises {
		out[i] = decalage + pas*float64(i)
	}
	return out
}

// anneau — une courbe de niveau : sa profondeur sous le sommet et ses projets (indices).
type anneau struct {
	profondeur float64
	membres    []int
}

// calculerPositions — voir l'en-tête. `strates` donne la strate de chaque id ; les liens dont une
// extrémité est inconnue sont ignorés. Le résultat ne dépend QUE du contenu des entrées, jamais
// de leur ordre (les ids sont triés avant tout calcul).
func calculerPositions(strates map[string]Strate, liens []MontagneLien) (map[string]PositionNoeud, []MontagneStrate, MontagneForme) {
	ids := make([]string, 0, len(strates))
	for id := range strates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	index := make(map[string]int, len(ids))
	for i, id := range ids {
		index[id] = i
	}

	// Voisins (les deux sens) et degré, en indices triés : l'ordre d'entrée des liens ne compte pas.
	voisins := make([][]int, len(ids))
	for _, l := range liens {
		a, okA := index[l.From]
		b, okB := index[l.To]
		if !okA || !okB || a == b {
			continue
		}
		voisins[a] = append(voisins[a], b)
		voisins[b] = append(voisins[b], a)
	}
	for i := range voisins {
		sort.Ints(voisins[i])
	}

	parStrate := map[Strate][]int{}
	for i, id := range ids {
		parStrate[strates[id]] = append(parStrate[strates[id]], i)
	}

	// Étapes 2 et 3 — anneaux du sommet vers le pied, strate du haut d'abord.
	anneauxDe := map[Strate][]anneau{}
	profondeur := 0.0
	premiere := true
	for k := len(ordreStrates) - 1; k >= 0; k-- {
		s := ordreStrates[k]
		membres := parStrate[s]
		if len(membres) == 0 {
			continue
		}
		if !premiere {
			profondeur += ecartStrates
		}
		premiere = false
		sort.SliceStable(membres, func(i, j int) bool {
			if len(voisins[membres[i]]) != len(voisins[membres[j]]) {
				return len(voisins[membres[i]]) > len(voisins[membres[j]])
			}
			return ids[membres[i]] < ids[membres[j]]
		})
		var caps []int
		var profs []float64
		for total := 0; total < len(membres); profondeur += ecartAnneaux {
			c := capaciteAnneau(profondeur)
			caps = append(caps, c)
			profs = append(profs, profondeur)
			total += c
		}
		profondeur -= ecartAnneaux // la boucle a avancé d'un cran de trop
		parts := repartir(len(membres), caps)
		debut := 0
		for r, nb := range parts {
			anneauxDe[s] = append(anneauxDe[s], anneau{profondeur: profs[r], membres: membres[debut : debut+nb]})
			debut += nb
		}
	}
	apexY := profondeur // le pied est à y = 0

	// Étape 4 — angles, du pied vers le sommet.
	n := len(ids)
	angle := make([]float64, n)
	place := make([]bool, n)
	xs, ys, zs := make([]float64, n), make([]float64, n), make([]float64, n)
	for _, s := range ordreStrates {
		anneaux := anneauxDe[s]
		for r := len(anneaux) - 1; r >= 0; r-- {
			a := anneaux[r]
			type vise struct {
				i int
				v float64
			}
			vises := make([]vise, 0, len(a.membres))
			for _, i := range a.membres {
				var ref []float64
				for _, j := range voisins[i] {
					if place[j] {
						ref = append(ref, angle[j])
					}
				}
				v, ok := moyenneCirculaire(ref)
				if !ok {
					v = hacheUnitaire(ids[i]) * angleTour
				}
				// Ramené dans [0, 2π) : le tri cyclique compare des angles de même tour.
				vises = append(vises, vise{i, math.Mod(v+angleTour, angleTour)})
			}
			sort.SliceStable(vises, func(p, q int) bool {
				if vises[p].v != vises[q].v {
					return vises[p].v < vises[q].v
				}
				return ids[vises[p].i] < ids[vises[q].i]
			})
			brut := make([]float64, len(vises))
			for k, v := range vises {
				brut[k] = v.v
			}
			poses := placerAnneau(brut)
			rayon := penteMontagne * a.profondeur
			y := apexY - a.profondeur
			for k, v := range vises {
				angle[v.i] = poses[k]
				place[v.i] = true
				xs[v.i] = rayon * math.Cos(poses[k])
				zs[v.i] = rayon * math.Sin(poses[k])
				ys[v.i] = y
			}
		}
	}

	// Bandes : du pied (anneau le plus profond) au haut de chaque strate. Les strates vides
	// apparaissent avec une bande nulle (Count = 0) pour que le front reçoive toujours les quatre.
	bandes := make([]MontagneStrate, 0, len(ordreStrates))
	for niveau, s := range ordreStrates {
		b := MontagneStrate{ID: s, Level: niveau, Count: len(parStrate[s])}
		if anneaux := anneauxDe[s]; len(anneaux) > 0 {
			bas := anneaux[len(anneaux)-1].profondeur
			b.YMin = arrondi3(apexY - bas)
			b.YMax = arrondi3(apexY - anneaux[0].profondeur)
			b.Radius = arrondi3(penteMontagne * bas)
		}
		bandes = append(bandes, b)
	}

	out := make(map[string]PositionNoeud, n)
	for i, id := range ids {
		out[id] = PositionNoeud{X: arrondi3(xs[i]), Y: arrondi3(ys[i]), Z: arrondi3(zs[i])}
	}
	return out, bandes, MontagneForme{ApexY: arrondi3(apexY), Slope: penteMontagne}
}
