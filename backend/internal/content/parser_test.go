package content

import (
	"testing"

	"maicivy/internal/models"
)

func TestParseExperience(t *testing.T) {
	md := `---
title: "Dev Full-Stack"
company: "Acme Corp"
category: backend
start_date: 2021-01
end_date: 2024-08
technologies: [Go, PostgreSQL, Docker]
tags: [backend, devops]
featured: true
---

## Description

Building awesome APIs.

## Description Technique

Go + Fiber + GORM.

## Description Fonctionnelle

Développement backend complet.
`

	exp, err := ParseExperience(md)
	if err != nil {
		t.Fatalf("ParseExperience failed: %v", err)
	}

	if exp.Title != "Dev Full-Stack" {
		t.Errorf("Title = %q, want %q", exp.Title, "Dev Full-Stack")
	}
	if exp.Company != "Acme Corp" {
		t.Errorf("Company = %q, want %q", exp.Company, "Acme Corp")
	}
	if exp.Category != "backend" {
		t.Errorf("Category = %q, want %q", exp.Category, "backend")
	}
	if exp.StartDate.Year() != 2021 || exp.StartDate.Month() != 1 {
		t.Errorf("StartDate = %v, want 2021-01", exp.StartDate)
	}
	if exp.EndDate == nil {
		t.Fatal("EndDate should not be nil")
	}
	if exp.EndDate.Year() != 2024 || exp.EndDate.Month() != 8 {
		t.Errorf("EndDate = %v, want 2024-08", *exp.EndDate)
	}
	if len(exp.Technologies) != 3 {
		t.Errorf("Technologies count = %d, want 3", len(exp.Technologies))
	}
	if !exp.Featured {
		t.Error("Featured should be true")
	}
	if exp.Description != "Building awesome APIs." {
		t.Errorf("Description = %q", exp.Description)
	}
	if exp.TechnicalDescription != "Go + Fiber + GORM." {
		t.Errorf("TechnicalDescription = %q", exp.TechnicalDescription)
	}
	if exp.FunctionalDescription != "Développement backend complet." {
		t.Errorf("FunctionalDescription = %q", exp.FunctionalDescription)
	}
}

func TestParseExperienceCurrentJob(t *testing.T) {
	md := `---
title: "Freelance"
company: "Self"
category: backend
start_date: 2025-01
end_date: null
technologies: [Python]
tags: [backend]
featured: false
---

## Description

Freelancing.
`

	exp, err := ParseExperience(md)
	if err != nil {
		t.Fatalf("ParseExperience failed: %v", err)
	}
	if exp.EndDate != nil {
		t.Errorf("EndDate should be nil for current job, got %v", *exp.EndDate)
	}
}

func TestParseSkills(t *testing.T) {
	md := `# Skills

## Expert
- Excel | Tools | office, data | 5 | Advanced formulas | excel

## Advanced
- Go | Languages | backend | 2 | Backend dev | golang | featured
- TypeScript | Languages | frontend, backend | 3 | Full-stack | typescript | featured

## Intermediate
- Docker | DevOps | devops, containers | 2 | Containers | docker
`

	skills, err := ParseSkills(md)
	if err != nil {
		t.Fatalf("ParseSkills failed: %v", err)
	}

	if len(skills) != 4 {
		t.Fatalf("Expected 4 skills, got %d", len(skills))
	}

	// Excel - expert
	if skills[0].Name != "Excel" {
		t.Errorf("skills[0].Name = %q, want Excel", skills[0].Name)
	}
	if skills[0].Level != models.SkillLevelExpert {
		t.Errorf("skills[0].Level = %q, want expert", skills[0].Level)
	}
	if skills[0].Category != "Tools" {
		t.Errorf("skills[0].Category = %q, want Tools", skills[0].Category)
	}
	if skills[0].YearsExperience != 5 {
		t.Errorf("skills[0].YearsExperience = %d, want 5", skills[0].YearsExperience)
	}

	// Go - advanced, featured
	if skills[1].Name != "Go" {
		t.Errorf("skills[1].Name = %q, want Go", skills[1].Name)
	}
	if skills[1].Level != models.SkillLevelAdvanced {
		t.Errorf("skills[1].Level = %q, want advanced", skills[1].Level)
	}
	if !skills[1].Featured {
		t.Error("Go should be featured")
	}
	if skills[1].Icon != "golang" {
		t.Errorf("skills[1].Icon = %q, want golang", skills[1].Icon)
	}

	// Docker - intermediate, not featured
	if skills[3].Level != models.SkillLevelIntermediate {
		t.Errorf("skills[3].Level = %q, want intermediate", skills[3].Level)
	}
	if skills[3].Featured {
		t.Error("Docker should not be featured")
	}
}

func TestParseProject(t *testing.T) {
	md := `---
title: "MyProject"
category: fullstack
technologies: [Go, React]
featured: true
in_progress: true
github_language: Go
catchphrase: "A short tagline"
---

## Description

A cool project.
`

	proj, err := ParseProject(md)
	if err != nil {
		t.Fatalf("ParseProject failed: %v", err)
	}

	if proj.Title != "MyProject" {
		t.Errorf("Title = %q", proj.Title)
	}
	if proj.Category != "fullstack" {
		t.Errorf("Category = %q", proj.Category)
	}
	if !proj.Featured {
		t.Error("Featured should be true")
	}
	if !proj.InProgress {
		t.Error("InProgress should be true")
	}
	if proj.GithubLanguage != "Go" {
		t.Errorf("GithubLanguage = %q", proj.GithubLanguage)
	}
	if proj.Description != "A cool project." {
		t.Errorf("Description = %q", proj.Description)
	}
	if proj.Catchphrase != "A short tagline" {
		t.Errorf("Catchphrase = %q, want %q", proj.Catchphrase, "A short tagline")
	}
}

func TestParseExperienceWithCatchphraseAndLinks(t *testing.T) {
	md := `---
title: "Senior Dev"
company: "BigCo"
category: backend
start_date: 2020-01
end_date: 2023-12
technologies: [Go]
tags: [backend]
featured: false
catchphrase: "Led backend team"
images:
  - /img/a.jpg
  - /img/b.jpg
links:
  - name: "GitHub"
    url: "https://github.com/test"
    icon: "github"
  - name: "Blog"
    url: "https://blog.test"
---

## Description

Some work.
`

	exp, err := ParseExperience(md)
	if err != nil {
		t.Fatalf("ParseExperience failed: %v", err)
	}

	if exp.Catchphrase != "Led backend team" {
		t.Errorf("Catchphrase = %q, want %q", exp.Catchphrase, "Led backend team")
	}
	if len(exp.Images) != 2 {
		t.Errorf("Images count = %d, want 2", len(exp.Images))
	}
	if len(exp.Links) != 2 {
		t.Fatalf("Links count = %d, want 2", len(exp.Links))
	}
	if exp.Links[0].Name != "GitHub" || exp.Links[0].URL != "https://github.com/test" {
		t.Errorf("Links[0] = %+v", exp.Links[0])
	}
	if exp.Links[1].Icon != "" {
		t.Errorf("Links[1].Icon = %q, want empty", exp.Links[1].Icon)
	}
}
