package services

import (
	"regexp"
	"sort"
	"strings"
)

// ============================================================================
// Liens de la montagne — « ceci sert à construire cela ».
// ----------------------------------------------------------------------------
// QUOI : un lien orienté B → A dit « le projet B sert à construire le projet A ». Il n'est JAMAIS
// inventé : il est déduit des fiches maiProFiles par deux règles, toutes deux déterministes.
//
// RÈGLE 1 — la stack (kind "stack"). B sert à construire A quand un élément de la stack de A EST
// B, c'est-à-dire qu'il est égal, après normalisation (minuscules, lettres et chiffres seuls), à
// une des clés de B : son id (`grove_engine` → `groveengine`), son nom court (« GroveEngine »), ou
// la tête de ce nom avant une parenthèse (« AISSIA (CELUNA) » → `aissia`). Égalité stricte d'un
// élément, pas sous-chaîne : sinon un projet nommé « Go… » serait dans toutes les stacks Go.
//
// RÈGLE 2 — la description (kind "description"). La stack ne dit pas tout : `soliano` déclare
// « First consumer: Drifterra » sans figurer dans la stack de Drifterra. La règle lit la
// description courte ANGLAISE de A, cherche une mention d'un autre projet B, et ne crée un lien
// que si la mention est introduite par une tournure qui DONNE LE SENS :
//   - B sert A : « built on B », « based on B », « on top of B », « powered by B », « runs on B »,
//     « on B », « B engine » (« (Demiurgos engine) »), « of a B world » (« a cell of a Theomen
//     world ») ;
//   - A sert B : « for the game B », « consumed by B », « first consumer: B », « used by B ».
// POURQUOI des tournures et pas une simple mention : mesuré sur les 75 fiches, une mention nue
// est ambiguë sur le sens — « voices for the game Drifterra » (aveg sert Drifterra) et « on
// GroveEngine » (GroveEngine sert Drifterra) mentionnent tous deux un projet, en sens contraires.
// Une mention sans tournure reconnue ne produit RIEN.
// POURQUOI l'anglais seul : les tournures sont des motifs anglais ; lire la fiche française
// produirait un autre graphe. La structure de la montagne est donc toujours calculée sur
// l'édition anglaise, quelle que soit la langue affichée.
//
// Un lien présent par les deux règles garde le kind "stack" (la preuve la plus forte).
// ============================================================================

// projetStructure — ce dont la déduction des liens et des strates a besoin d'une fiche. Découplé
// de MPFProject pour que les tests construisent des fiches sans tout le reste.
type projetStructure struct {
	ID          string
	Name        string
	Category    string
	Tags        []string
	Stack       []string
	Description string // description courte, ANGLAISE (cf. en-tête, règle 2)
}

// MontagneLien — B (From) sert à construire A (To).
type MontagneLien struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"` // "stack" | "description"
}

const (
	lienStack       = "stack"
	lienDescription = "description"
	// longueurCleMin — une clé plus courte (« go », « ai ») ne désigne pas un projet mais une
	// techno ou un mot courant : elle ne sert à aucune des deux règles.
	longueurCleMin = 4
)

// cleCompacte — minuscules, lettres et chiffres seuls : « GroveEngine », « grove_engine » et
// « groveengine » ont la même clé.
func cleCompacte(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// nomsDuProjet — les façons de nommer un projet : nom court, tête avant parenthèse, id.
// Renvoyés en clair (pour la règle 2, qui cherche dans un texte) ; dédupliqués, triés.
func nomsDuProjet(p projetStructure) []string {
	court := nomIdentite(p.Name)
	noms := []string{court, p.ID}
	if i := strings.Index(court, "("); i > 0 {
		noms = append(noms, strings.TrimSpace(court[:i]))
	}
	vus := map[string]bool{}
	var out []string
	for _, n := range noms {
		n = strings.TrimSpace(n)
		if len(cleCompacte(n)) < longueurCleMin || vus[strings.ToLower(n)] {
			continue
		}
		vus[strings.ToLower(n)] = true
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// motifsBServantA / motifsAServantB — les tournures de la règle 2. `%s` reçoit le nom échappé.
// « on %s » couvre « built on », « based on », « runs on » : la borne de mot gauche posée par
// compilerMotifs empêche seulement « Avalon X » de passer pour « on X ».
var (
	motifsBServantA = []string{
		`on %s`,
		`on top of %s`,
		`powered by %s`,
		`%s engine`,
		`of an? %s world`,
	}
	motifsAServantB = []string{
		`for the game %s`,
		`consumed by %s`,
		`first consumer:? %s`,
		`used by %s`,
	}
)

// compilerMotifs — une regex par (nom, motif), insensible à la casse, bornée par des fins de mot
// des deux côtés. Bornes faites à la main ((?:^|[^\pL\pN_])) : RE2 n'a pas de lookbehind, et \b
// n'y connaît que l'ASCII. Compilées une fois par déduction : 75 × 3 noms × 9 motifs reste
// négligeable devant un rafraîchissement de cache toutes les 10 min.
func compilerMotifs(nom string, motifs []string) []*regexp.Regexp {
	const gauche, droite = `(?:^|[^\pL\pN_])`, `(?:$|[^\pL\pN_])`
	echappe := regexp.QuoteMeta(nom)
	out := make([]*regexp.Regexp, 0, len(motifs))
	for _, m := range motifs {
		out = append(out, regexp.MustCompile(`(?i)`+gauche+strings.Replace(m, "%s", echappe, 1)+droite))
	}
	return out
}

// deduireLiens — voir l'en-tête. Sortie triée (From, To) : déterministe quel que soit l'ordre
// des fiches en entrée.
func deduireLiens(projets []projetStructure) []MontagneLien {
	// Index des clés compactes → id, pour la règle 1.
	cles := map[string]string{}
	for _, p := range projets {
		for _, n := range nomsDuProjet(p) {
			cles[cleCompacte(n)] = p.ID
		}
	}

	liens := map[[2]string]string{} // (from, to) → kind
	ajouter := func(from, to, kind string) {
		if from == to {
			return
		}
		k := [2]string{from, to}
		if existant, ok := liens[k]; ok && existant == lienStack {
			return // la stack prime
		}
		liens[k] = kind
	}

	// Règle 1 — la stack.
	for _, a := range projets {
		for _, element := range a.Stack {
			if b, ok := cles[cleCompacte(element)]; ok {
				ajouter(b, a.ID, lienStack)
			}
		}
	}

	// Règle 2 — les tournures de la description anglaise.
	for _, b := range projets {
		var servant, servi []*regexp.Regexp
		for _, n := range nomsDuProjet(b) {
			servant = append(servant, compilerMotifs(n, motifsBServantA)...)
			servi = append(servi, compilerMotifs(n, motifsAServantB)...)
		}
		for _, a := range projets {
			if a.ID == b.ID || a.Description == "" {
				continue
			}
			for _, re := range servant {
				if re.MatchString(a.Description) {
					ajouter(b.ID, a.ID, lienDescription)
					break
				}
			}
			for _, re := range servi {
				if re.MatchString(a.Description) {
					ajouter(a.ID, b.ID, lienDescription)
					break
				}
			}
		}
	}

	out := make([]MontagneLien, 0, len(liens))
	for k, kind := range liens {
		out = append(out, MontagneLien{From: k[0], To: k[1], Kind: kind})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}
