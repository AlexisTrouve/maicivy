package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// La sélection des « Compétences mobilisées » (dossier_skills.go) : tout vient des données — le
// profil pour la liste, les fiches pour le lien compétence → projet — et rien d'autre. Formes des
// données : celles de maiProFiles (tags des fiches = noms canoniques français des compétences,
// /skills traduit ses noms, piles en minuscules avec versions).

var canonSkills = []MPFSkill{
	{Name: "AI / LLM Integration", Level: "expert", Years: 4},
	{Name: "Architecture Logicielle", Level: "expert", Years: 10},
	{Name: "C/C++", Level: "expert", Years: 10},
	{Name: "Automatisation", Level: "advanced", Years: 5},
	{Name: "Python", Level: "advanced", Years: 3},
	{Name: "MCP (Model Context Protocol)", Level: "expert", Years: 2},
	{Name: "Rust", Level: "intermediate", Years: 1},
}

// Même fichier, autre langue : même rang, mêmes niveaux et années, noms traduits.
var enSkills = []MPFSkill{
	{Name: "AI / LLM Integration", Level: "expert", Years: 4},
	{Name: "Software Architecture", Level: "expert", Years: 10},
	{Name: "C/C++", Level: "expert", Years: 10},
	{Name: "Automation", Level: "advanced", Years: 5},
	{Name: "Python", Level: "advanced", Years: 3},
	{Name: "MCP (Model Context Protocol)", Level: "expert", Years: 2},
	{Name: "Rust", Level: "intermediate", Years: 1},
}

var skillProjects = []MPFProject{
	{ID: "envoia", Stack: []string{"python", "openai", "sqlite", "pytest"}, Tags: []string{"llm", "AI / LLM Integration", "Architecture Logicielle", "Automatisation"}},
	{ID: "veracite", Stack: []string{"python", "spacy", "sqlite", "pytest"}, Tags: []string{"nlp", "AI / LLM Integration", "Architecture Logicielle"}},
	{ID: "grove_engine", Stack: []string{"c++17", "cmake", "sdl2"}, Tags: []string{"Architecture Logicielle"}},
	{ID: "melodicode", Stack: []string{"flutter", "dart", "mcp"}, Tags: []string{"AI / LLM Integration"}},
}

func names(s []DossierSkill) []string {
	out := make([]string, len(s))
	for i, k := range s {
		out[i] = k.Name
	}
	return out
}

func TestDossierSkills_ToutVientDesDonnees(t *testing.T) {
	got := selectDossierSkills(canonSkills, canonSkills, skillProjects, []string{"envoia", "veracite", "grove_engine"}, "")
	// Rust n'est mobilisé par aucune preuve, MCP seulement par melodicode (non cité) : absents.
	// Ordre : le plus de preuves d'abord, puis le niveau, puis l'ordre du profil.
	assert.Equal(t, []string{"Architecture Logicielle", "AI / LLM Integration", "Python", "C/C++", "Automatisation"}, names(got.Skills))
	// Chaque compétence dit d'où elle vient, dans l'ordre du dossier.
	assert.Equal(t, []string{"envoia", "veracite", "grove_engine"}, got.Skills[0].Projects)
	// « c++17 » de la pile est la compétence C/C++ (version retirée, partie du nom).
	assert.Equal(t, []string{"grove_engine"}, got.Skills[3].Projects)
	// Technologies communes à deux preuves au moins, hors compétences retenues (python).
	require.Len(t, got.Technologies, 2)
	assert.Equal(t, DossierTechnology{Name: "sqlite", Projects: []string{"envoia", "veracite"}}, got.Technologies[0])
	assert.Equal(t, "pytest", got.Technologies[1].Name)
}

func TestDossierSkills_NommeeDansLeBesoinPasseDevant(t *testing.T) {
	need := "Des contrôles écrits en Python, hors prompt.\nLe C++ n'est pas requis... sauf la brique C/C++ existante."
	got := selectDossierSkills(canonSkills, canonSkills, skillProjects, []string{"envoia", "veracite", "grove_engine"}, need)
	assert.Equal(t, "Python", got.Skills[0].Name)
	assert.True(t, got.Skills[0].Need)
	assert.Equal(t, "C/C++", got.Skills[1].Name)
	assert.True(t, got.Skills[1].Need)
	assert.False(t, got.Skills[2].Need)
	// Mot entier seulement : « Rustique » ne nomme pas Rust, « api » ne nomme pas « rapide ».
	assert.False(t, containsWord("un outil rustique", "rust"))
	assert.True(t, containsWord("écrit en rust.", "rust"))
	assert.True(t, containsWord("du c++ moderne", "c++"))
	assert.True(t, containsWord("用python写", "python") == false, "collé à des idéogrammes : pas un mot latin séparé")
	assert.True(t, containsWord("需要软件架构设计", "软件架构"), "CJK : sous-chaîne")
}

func TestDossierSkills_NomLocaliseAppariéSurLeCanonique(t *testing.T) {
	got := selectDossierSkills(canonSkills, enSkills, skillProjects, []string{"envoia"}, "")
	assert.Equal(t, []string{"AI / LLM Integration", "Software Architecture", "Automation", "Python"}, names(got.Skills))

	// Listes qui ne se correspondent pas rang pour rang : le nom canonique, jamais un faux nom.
	shifted := append([]MPFSkill{{Name: "Go", Level: "advanced", Years: 1}}, enSkills...)
	got = selectDossierSkills(canonSkills, shifted, skillProjects, []string{"envoia"}, "")
	assert.Equal(t, []string{"AI / LLM Integration", "Architecture Logicielle", "Automatisation", "Python"}, names(got.Skills))
}

func TestDossierSkills_PreuveSansFicheEtDossierSansPreuve(t *testing.T) {
	got := selectDossierSkills(canonSkills, canonSkills, skillProjects, []string{"inconnu"}, "Python")
	assert.True(t, got.Empty(), "aucune fiche : rien de vérifiable, aucune compétence")
	assert.True(t, selectDossierSkills(canonSkills, canonSkills, skillProjects, nil, "").Empty())
}

func TestDossierSkills_Plafonds(t *testing.T) {
	var canon []MPFSkill
	var tags []string
	for i := 0; i < 15; i++ {
		n := string(rune('A'+i)) + "-skill"
		canon = append(canon, MPFSkill{Name: n, Level: "expert"})
		tags = append(tags, n)
	}
	stack := []string{"ta", "tb", "tc", "td", "te", "tf", "tg", "th", "ti", "tj", "tk", "tl"}
	projects := []MPFProject{{ID: "a", Tags: tags, Stack: stack}, {ID: "b", Tags: tags, Stack: stack}}
	got := selectDossierSkills(canon, canon, projects, []string{"a", "b"}, "")
	assert.Len(t, got.Skills, dossierSkillsMax)
	assert.Equal(t, "A-skill", got.Skills[0].Name, "à égalité, l'ordre du profil")
	assert.Len(t, got.Technologies, dossierTechMax)
}

func TestDossierSkills_EntreeLueDansLeDossier(t *testing.T) {
	proofs, need := skillsInputOf(dossierFixture().Sections)
	assert.Equal(t, []string{"envoia", "aria"}, proofs)
	assert.Contains(t, need, "sous validation", "le besoin reformulé")
	assert.Contains(t, need, "Extraction LLM", "les boîtes du schéma")
	assert.NotContains(t, need, "Un court échange", "la prochaine étape ne décrit pas le besoin")
}

func TestDossierSkills_TechKey(t *testing.T) {
	for in, want := range map[string]string{"c++17": "c++", "Python 3.11": "python", "sdl2": "sdl", "node.js": "node.js", "ed25519-dalek": "ed25519-dalek", "MCP": "mcp"} {
		assert.Equal(t, want, techKey(in), in)
	}
	assert.Equal(t, []string{"c/c++", "c", "c++"}, skillTokens("C/C++"))
	assert.Equal(t, []string{"mcp"}, skillTokens("MCP (Model Context Protocol)"))
}
