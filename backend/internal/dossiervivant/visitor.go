package dossiervivant

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// QUOI : le visiteur tel qu'il s'est présenté — une ligne « Nom, rôle chez organisation » — et son
// besoin, mis en forme pour `run.started` (nom, affiliation, organisation, initiales de sa tuile).
//
// POURQUOI côté serveur : run.started est le premier événement du flux, émis avant tout appel de
// modèle. Le découpage doit donc être déterministe et immédiat — pas une tâche pour un agent.
//
// COMMENT : nom = avant la première virgule (latine ou chinoise), affiliation = après. Organisation
// = ce qui suit le DERNIER connecteur connu (« chez », « at », « bei », « presso », « @ ») ; sans
// connecteur, l'affiliation entière (« Tessera » seul reste « Tessera »).

// Bornes de saisie : le besoin doit dire quelque chose, et tenir dans un prompt raisonnable.
const (
	needMinRunes = 20
	needMaxRunes = 2000
	whoMaxRunes  = 120
)

var (
	ErrNeedLength   = errors.New("dossier vivant: besoin trop court ou trop long")
	ErrVisitorShape = errors.New("dossier vivant: « nom, rôle » attendu")
)

var orgConnectors = []string{" chez ", " at ", " bei ", " presso ", " @ ", "@"}

// ParseVisitor — valide la saisie (entrée externe) et construit la fiche du visiteur.
func ParseVisitor(who, need string) (VisitorBrief, error) {
	need = strings.TrimSpace(need)
	if n := utf8.RuneCountInString(need); n < needMinRunes || n > needMaxRunes {
		return VisitorBrief{}, ErrNeedLength
	}
	who = strings.TrimSpace(who)
	if utf8.RuneCountInString(who) > whoMaxRunes {
		return VisitorBrief{}, ErrVisitorShape
	}
	cut := strings.IndexAny(who, ",，")
	if cut < 0 {
		return VisitorBrief{}, ErrVisitorShape
	}
	name := strings.TrimSpace(who[:cut])
	_, sepLen := utf8.DecodeRuneInString(who[cut:])
	affiliation := strings.TrimSpace(who[cut+sepLen:])
	if name == "" || affiliation == "" {
		return VisitorBrief{}, ErrVisitorShape
	}
	return VisitorBrief{
		Name:         name,
		Affiliation:  affiliation,
		Organisation: organisationOf(affiliation),
		Initials:     initialsOf(name),
		Need:         need,
	}, nil
}

func organisationOf(affiliation string) string {
	// Recherche sur la chaîne d'origine (connecteurs en minuscules, comme on les écrit) : un
	// ToLower préalable peut changer la longueur en octets et décaler les indices.
	best := -1
	bestLen := 0
	for _, c := range orgConnectors {
		if i := strings.LastIndex(affiliation, c); i > best {
			best, bestLen = i, len(c)
		}
	}
	if best < 0 {
		return affiliation
	}
	if org := strings.TrimSpace(affiliation[best+bestLen:]); org != "" {
		return org
	}
	return affiliation
}

// initialsOf — deux lettres pour la tuile : initiale du premier et du dernier mot ; un seul mot →
// ses deux premières lettres (un nom chinois de deux caractères reste entier).
func initialsOf(name string) string {
	words := strings.Fields(name)
	first := []rune(words[0])
	if len(words) == 1 {
		if len(first) > 2 {
			first = first[:2]
		}
		return strings.ToUpper(string(first))
	}
	last := []rune(words[len(words)-1])
	return string([]rune{unicode.ToUpper(first[0]), unicode.ToUpper(last[0])})
}
