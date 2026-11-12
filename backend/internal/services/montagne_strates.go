package services

import (
	"strings"
	"unicode"
)

// ============================================================================
// Strates de la montagne — où un projet se range sur l'axe vertical.
// ----------------------------------------------------------------------------
// QUOI : quatre strates, du bas vers le haut — fondations (infra), outillage, moteurs, œuvres
// (jeux, applications). Chaque projet reçoit UNE strate, dérivée de sa `category` puis, à défaut,
// de ses `tags`, par la table explicite `tableStrates`.
//
// POURQUOI une table et pas un champ maiProFiles : la donnée n'a pas de champ « couche » ; la
// `category` est un texte libre (« Game / C++ engine / Procedural generation ») rédigé fiche par
// fiche. Une table lisible et testée rend la décision vérifiable : on sait POURQUOI un projet est
// en bas ou en haut, et on corrige la table plutôt que la donnée.
//
// COMMENT :
//  1. La catégorie est découpée sur « / » en segments ; le PREMIER segment qui contient un mot-clé
//     décide. Le premier segment est celui que l'auteur de la fiche a mis en tête — c'est la
//     nature du projet (« Game / C++ engine » est un jeu, « C++ engine / Navigation » est un
//     moteur). Un segment purement langage (« Python », « C++ », « Rust ») ne contient aucun
//     mot-clé et laisse la main au suivant.
//  2. Dans un segment, la table est lue DANS L'ORDRE : moteurs, puis fondations, puis outillage,
//     puis œuvres. L'ordre tranche les segments mixtes : « Game Engine » est un moteur et non un
//     jeu, « DevOps Automation » une fondation et non un outil.
//  3. Aucun segment ne parle → mêmes règles sur les tags, dans leur ordre.
//  4. Rien ne parle → `strateParDefaut` (outillage), le milieu de la montagne : un projet qu'on ne
//     sait pas classer n'est ni présenté comme une fondation ni comme une œuvre.
//
// Un mot-clé matche au DÉBUT d'un mot (« engine » matche « engines », « game » matche « games »
// mais « rag » ne matche pas « storage ») après normalisation : minuscules, tout ce qui n'est pas
// lettre ou chiffre devient une espace (« Full-Stack » → « full stack »).
// ============================================================================

// Strate — identifiant stable d'une couche, utilisé tel quel par le front (clé i18n).
type Strate string

const (
	StrateFondations Strate = "foundations"
	StrateOutillage  Strate = "tooling"
	StrateMoteurs    Strate = "engines"
	StrateOeuvres    Strate = "works"
)

// ordreStrates — du bas (0) vers le haut. Sert au rendu (bandes verticales) et aux tests d'ordre.
var ordreStrates = []Strate{StrateFondations, StrateOutillage, StrateMoteurs, StrateOeuvres}

// niveauStrate — position de la strate dans ordreStrates.
func niveauStrate(s Strate) int {
	for i, x := range ordreStrates {
		if x == s {
			return i
		}
	}
	return -1
}

// strateParDefaut — voir l'étape 4 de l'en-tête.
const strateParDefaut = StrateOutillage

// regleStrate — un mot-clé (déjà normalisé) et la strate qu'il désigne.
type regleStrate struct {
	motCle string
	strate Strate
}

// tableStrates — LA table de correspondance. L'ordre compte (cf. en-tête, étape 2).
// Chaque ligne cite le cas réel qui l'a motivée, pour qu'on puisse la retirer en connaissance
// de cause.
var tableStrates = []regleStrate{
	// --- Moteurs : ce sur quoi les œuvres tournent ---
	{"engine", StrateMoteurs},  // « C++ engine / Navigation » (daedalium), « Game Engine »
	{"runtime", StrateMoteurs}, // « Modular Runtime » (grove_engine)

	// --- Fondations : infra, réseau, sécurité, plateformes internes ---
	{"infrastructure", StrateFondations},      // « Data infrastructure », « Security infrastructure »
	{"devops", StrateFondations},              // « DevOps Automation »
	{"backend development", StrateFondations}, // « Backend Development / API Infrastructure » — PAS « Rust backend », qui qualifie une appli
	{"networking", StrateFondations},          // « Networking / Anti-censorship » (ladderkeg)
	{"internal platform", StrateFondations},   // « Infrastructure / Internal platform »
	{"client library", StrateFondations},      // « Client library / Dart »
	{"security", StrateFondations},

	// --- Outillage : ce qui sert à fabriquer (outils dev, agents, automatisation, données) ---
	{"developer tool", StrateOutillage}, // « Developer Tools », « Developer tooling »
	{"tooling", StrateOutillage},        // « AI / LLM Tooling »
	{"mcp", StrateOutillage},            // « MCP Server »
	{"automation", StrateOutillage},     // « AI Automation », « Browser Automation »
	{"scraping", StrateOutillage},       // « Web Scraping »
	{"ai integration", StrateOutillage},
	{"ai tools", StrateOutillage},
	{"ai development", StrateOutillage},
	{"ai agent", StrateOutillage},
	{"rag", StrateOutillage},
	{"nlp", StrateOutillage},
	{"llm", StrateOutillage},
	{"api development", StrateOutillage},
	{"content generation", StrateOutillage},
	{"data engineering", StrateOutillage},
	{"data processing", StrateOutillage},
	{"audio processing", StrateOutillage},
	{"applied research", StrateOutillage},
	{"cli", StrateOutillage},

	// --- Œuvres : ce qu'un utilisateur final ouvre — jeux, applications, produits ---
	{"game", StrateOeuvres},
	{"application", StrateOeuvres}, // « Web application », « Desktop application », « AI Application »
	{"web app", StrateOeuvres},
	{"saas", StrateOeuvres},
	{"mobile", StrateOeuvres},
	{"productivity", StrateOeuvres},
	{"worldbuilding", StrateOeuvres},
	{"constructed language", StrateOeuvres},
	{"wechat", StrateOeuvres},
	{"finance", StrateOeuvres},
	{"trading", StrateOeuvres},
}

// normaliserTexte — minuscules, tout non-alphanumérique devient une espace, espaces compactées,
// encadré d'espaces pour que la recherche « début de mot » soit un simple Contains(" "+motCle).
func normaliserTexte(s string) string {
	var b strings.Builder
	b.WriteByte(' ')
	espace := true
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			espace = false
			continue
		}
		if !espace {
			b.WriteByte(' ')
			espace = true
		}
	}
	if !espace {
		b.WriteByte(' ')
	}
	return b.String()
}

// strateDuTexte — première règle de la table qui matche ce texte, ou "" si aucune.
func strateDuTexte(texte string) Strate {
	norm := normaliserTexte(texte)
	for _, r := range tableStrates {
		if strings.Contains(norm, " "+r.motCle) {
			return r.strate
		}
	}
	return ""
}

// strateDuProjet — voir l'en-tête du fichier (étapes 1 à 4).
func strateDuProjet(category string, tags []string) Strate {
	for _, segment := range strings.Split(category, "/") {
		if s := strateDuTexte(segment); s != "" {
			return s
		}
	}
	for _, tag := range tags {
		if s := strateDuTexte(tag); s != "" {
			return s
		}
	}
	return strateParDefaut
}
