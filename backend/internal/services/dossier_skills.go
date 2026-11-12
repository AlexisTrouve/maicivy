package services

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// dossier_skills.go — la section « Compétences mobilisées » du dossier vivant (écran ET PDF).
//
// QUOI : à partir des projets cités en PREUVE dans le dossier remis, rend les compétences du profil
// d'Alexis (maiProFiles GET /skills) que ces projets mobilisent, chacune avec les projets d'où elle
// vient, plus les technologies que plusieurs preuves partagent (piles des fiches).
//
// POURQUOI le code choisit, et aucun modèle : une compétence imprimée sous le nom d'Alexis doit
// exister dans son profil et se voir dans un projet réel. Ici rien ne s'invente : la liste vient
// de /skills, le lien compétence → projet vient de la fiche (ses `tags` portent le nom canonique
// des compétences, sa `stack` les technologies), et l'ordre est une règle écrite plus bas. Le seul
// rapport au besoin est textuel et vérifiable : le nom de la compétence figure dans le besoin
// reformulé ou dans l'architecture proposée (`Need`).
//
// COMMENT :
//   - les tags des fiches portent les noms CANONIQUES (français) des compétences, quelle que soit la
//     langue demandée ; /skills, lui, traduit ses noms. On apparie donc sur la liste canonique
//     (lang "fr") et on affiche le nom de la liste localisée de même rang — maiProFiles sert les
//     deux depuis le même fichier, dans le même ordre ; si les deux listes ne se correspondent pas
//     rang pour rang (niveau, années), on garde le nom canonique plutôt que d'en afficher un faux ;
//   - UNE implémentation, deux lecteurs : l'écran l'obtient par POST /dossier-vivant/skills, le PDF
//     l'appelle directement, sur les mêmes sections du dossier (skillsInputOf). Les deux ne peuvent
//     donc pas diverger.

const (
	// dossierSkillsMax — compétences rendues au plus (profil : 26 ; au-delà de 10, la liste ne
	// sélectionne plus rien).
	dossierSkillsMax = 10
	// DossierSkillsCore — compétences toujours imprimées ; les suivantes sont la RÉSERVE que la mise
	// en page du PDF dévoile pour remplir la dernière page (dossier_pdf_template.html).
	DossierSkillsCore = 6
	// dossierTechMax / dossierTechMinProofs — technologies communes : au moins deux preuves, dix au plus.
	dossierTechMax       = 10
	dossierTechMinProofs = 2
)

// DossierSkill — une compétence du profil, mobilisée par au moins une preuve.
type DossierSkill struct {
	Name     string   `json:"name"`     // nom affiché (liste localisée de maiProFiles)
	Level    string   `json:"level"`    // expert | advanced | intermediate | beginner (clé, libellé côté interface)
	Need     bool     `json:"need"`     // son nom figure dans le besoin reformulé ou l'architecture
	Projects []string `json:"projects"` // preuves qui la mobilisent, dans l'ordre du dossier
}

// DossierTechnology — une technologie de la pile d'au moins deux preuves.
type DossierTechnology struct {
	Name     string   `json:"name"`
	Projects []string `json:"projects"`
}

// DossierSkills — la section entière. Vide quand le dossier ne cite aucune preuve.
type DossierSkills struct {
	Skills       []DossierSkill      `json:"skills"`
	Technologies []DossierTechnology `json:"technologies"`
}

// Empty — rien à montrer : la section n'est pas rendue (jamais un titre sans contenu).
func (d DossierSkills) Empty() bool { return len(d.Skills) == 0 && len(d.Technologies) == 0 }

// skillsInputOf — ce que la sélection lit dans le dossier : les preuves, dans l'ordre et sans
// doublon, et le texte du besoin (sections `need` et `architecture`, schéma compris).
func skillsInputOf(sections []DossierPDFSection) (proofs []string, needText string) {
	seen := map[string]bool{}
	var sb strings.Builder
	for _, sec := range sections {
		for _, b := range sec.Blocks {
			if b.Type == "proof" && !seen[b.ProjectID] {
				seen[b.ProjectID] = true
				proofs = append(proofs, b.ProjectID)
			}
			if sec.ID != "need" && sec.ID != "architecture" {
				continue
			}
			sb.WriteString(b.Text)
			sb.WriteByte('\n')
			if d := b.Diagram; d != nil {
				sb.WriteString(d.Label + "\n" + d.Journal + "\n")
				for _, box := range d.Boxes {
					sb.WriteString(box.Title + "\n" + box.Subtitle + "\n")
				}
			}
		}
	}
	return proofs, sb.String()
}

// versionSuffix — « c++17 », « python 3.11 », « sdl2 » : la même technologie que sans version.
var versionSuffix = regexp.MustCompile(`^(.*?[a-z+#])[ \-]?v?\d+(\.\d+)*$`)

// parenthetical — « MCP (Model Context Protocol) » s'apparie sur « mcp ».
var parenthetical = regexp.MustCompile(`\s*\([^)]*\)`)

// techKey — clé de comparaison d'une technologie ou d'un tag : minuscules, sans version.
func techKey(s string) string {
	k := strings.ToLower(strings.TrimSpace(s))
	if m := versionSuffix.FindStringSubmatch(k); m != nil {
		k = m[1]
	}
	return k
}

// skillTokens — les formes sous lesquelles une compétence se reconnaît : son nom entier et chacune
// de ses parties (« C/C++ » → c/c++, c, c++ ; « Flutter / Dart » → flutter, dart).
func skillTokens(name string) []string {
	base := strings.ToLower(strings.TrimSpace(parenthetical.ReplaceAllString(name, "")))
	out := []string{base}
	if strings.Contains(base, "/") {
		for _, p := range strings.Split(base, "/") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

// containsWord — `token` figure dans `text` (déjà en minuscules) comme mot entier. Un caractère CJK
// n'a pas de frontière de mot : la sous-chaîne suffit.
func containsWord(text, token string) bool {
	if token == "" {
		return false
	}
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	first, _ := utf8.DecodeRuneInString(token)
	cjk := unicode.Is(unicode.Han, first)
	for from := 0; ; {
		i := strings.Index(text[from:], token)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(token)
		before, _ := utf8.DecodeLastRuneInString(text[:start]) // RuneError en tête : pas une lettre
		after, _ := utf8.DecodeRuneInString(text[end:])
		if cjk || (!isWord(before) && !isWord(after)) {
			return true
		}
		from = start + 1
	}
}

var levelRank = map[string]int{"expert": 0, "advanced": 1, "intermediate": 2, "beginner": 3}

// selectDossierSkills — la sélection, pure : `canon` = /skills en français (noms des tags),
// `local` = /skills dans la langue du dossier, `projects` = fiches (tags et piles).
//
// Règles :
//   - une compétence est retenue si au moins une preuve la mobilise : un tag de la fiche porte son
//     nom canonique, ou un tag / une technologie de la fiche porte l'une de ses formes (skillTokens) ;
//   - ordre : nommée dans le besoin d'abord, puis le plus de preuves, puis le niveau, puis l'ordre du
//     profil (stable) ; dossierSkillsMax au plus ;
//   - technologies : celles de la pile d'au moins dossierTechMinProofs preuves, hors formes d'une
//     compétence retenue (« python » est déjà la compétence Python), les plus partagées d'abord.
func selectDossierSkills(canon, local []MPFSkill, projects []MPFProject, proofIDs []string, needText string) DossierSkills {
	fiches := make(map[string]MPFProject, len(projects))
	for _, p := range projects {
		fiches[p.ID] = p
	}
	// Clés de chaque preuve (tags + pile), et la pile seule pour les technologies.
	keys := make(map[string]map[string]bool, len(proofIDs))
	var proofs []string
	for _, id := range proofIDs {
		f, ok := fiches[id]
		if !ok {
			continue // preuve sans fiche : rien de vérifiable à en tirer
		}
		proofs = append(proofs, id)
		k := map[string]bool{}
		for _, s := range append(append([]string{}, f.Tags...), f.Stack...) {
			k[techKey(s)] = true
		}
		keys[id] = k
	}

	sameShape := len(local) == len(canon)
	for i := 0; sameShape && i < len(canon); i++ {
		sameShape = canon[i].Level == local[i].Level && canon[i].Years == local[i].Years
	}
	need := strings.ToLower(needText)

	type ranked struct {
		DossierSkill
		order int
	}
	var picked []ranked
	used := map[string]bool{} // formes des compétences retenues : exclues des technologies
	for i, s := range canon {
		tokens := skillTokens(s.Name)
		canonTag := strings.ToLower(strings.TrimSpace(s.Name))
		var from []string
		for _, id := range proofs {
			hit := keys[id][canonTag]
			for _, t := range tokens {
				hit = hit || keys[id][t]
			}
			if hit {
				from = append(from, id)
			}
		}
		if len(from) == 0 {
			continue
		}
		name := s.Name
		if sameShape && strings.TrimSpace(local[i].Name) != "" {
			name = local[i].Name
		}
		inNeed := false
		for _, t := range append(tokens, skillTokens(name)...) {
			inNeed = inNeed || containsWord(need, t)
		}
		picked = append(picked, ranked{DossierSkill{Name: name, Level: s.Level, Need: inNeed, Projects: from}, i})
	}
	sort.SliceStable(picked, func(a, b int) bool {
		pa, pb := picked[a], picked[b]
		if pa.Need != pb.Need {
			return pa.Need
		}
		if len(pa.Projects) != len(pb.Projects) {
			return len(pa.Projects) > len(pb.Projects)
		}
		ra, okA := levelRank[pa.Level]
		rb, okB := levelRank[pb.Level]
		if !okA {
			ra = len(levelRank)
		}
		if !okB {
			rb = len(levelRank)
		}
		if ra != rb {
			return ra < rb
		}
		return pa.order < pb.order
	})
	if len(picked) > dossierSkillsMax {
		picked = picked[:dossierSkillsMax]
	}
	// Listes jamais nulles : le JSON de la route rend [] et non null.
	out := DossierSkills{Skills: make([]DossierSkill, len(picked)), Technologies: []DossierTechnology{}}
	for i, p := range picked {
		out.Skills[i] = p.DossierSkill
		for _, t := range skillTokens(canon[p.order].Name) {
			used[t] = true
		}
		for _, t := range skillTokens(p.Name) {
			used[t] = true
		}
	}

	// Technologies : première graphie rencontrée, preuves dans l'ordre du dossier.
	type tech struct {
		name  string
		from  []string
		first int
	}
	techs := map[string]*tech{}
	n := 0
	for _, id := range proofs {
		seen := map[string]bool{}
		for _, s := range fiches[id].Stack {
			k := techKey(s)
			if k == "" || seen[k] || used[k] {
				continue
			}
			seen[k] = true
			t, ok := techs[k]
			if !ok {
				t = &tech{name: strings.TrimSpace(s), first: n}
				techs[k] = t
				n++
			}
			t.from = append(t.from, id)
		}
	}
	var shared []*tech
	for _, t := range techs {
		if len(t.from) >= dossierTechMinProofs {
			shared = append(shared, t)
		}
	}
	sort.Slice(shared, func(a, b int) bool {
		if len(shared[a].from) != len(shared[b].from) {
			return len(shared[a].from) > len(shared[b].from)
		}
		return shared[a].first < shared[b].first
	})
	if len(shared) > dossierTechMax {
		shared = shared[:dossierTechMax]
	}
	for _, t := range shared {
		out.Technologies = append(out.Technologies, DossierTechnology{Name: t.name, Projects: t.from})
	}
	return out
}

// Skills — la section du dossier `sections`, en langue `lang`. Les sections passent par la même
// validation que le PDF : la route de l'écran est publique.
func (s *DossierPDFService) Skills(ctx context.Context, lang string, sections []DossierPDFSection) (DossierSkills, error) {
	if err := validateDossierSections(lang, sections); err != nil {
		return DossierSkills{}, err
	}
	projects, err := s.facts.ListProjects(ctx, lang)
	if err != nil {
		return DossierSkills{}, fmt.Errorf("dossier skills: fiches : %w", err)
	}
	return s.skillsFor(ctx, lang, sections, projects)
}

// skillsFor — lit le profil (canonique + localisé) et sélectionne. Partagé avec le rendu du PDF,
// qui a déjà les fiches en main.
func (s *DossierPDFService) skillsFor(ctx context.Context, lang string, sections []DossierPDFSection, projects []MPFProject) (DossierSkills, error) {
	proofs, need := skillsInputOf(sections)
	if len(proofs) == 0 {
		return DossierSkills{}, nil
	}
	canon, err := s.facts.Skills(ctx, "fr")
	if err != nil {
		return DossierSkills{}, fmt.Errorf("dossier skills: profil : %w", err)
	}
	local := canon
	if lang != "fr" {
		if local, err = s.facts.Skills(ctx, lang); err != nil {
			return DossierSkills{}, fmt.Errorf("dossier skills: profil (%s) : %w", lang, err)
		}
	}
	return selectDossierSkills(canon, local, projects, proofs, need), nil
}
