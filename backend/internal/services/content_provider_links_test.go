package services

import "testing"

// Verrouille le masquage des liens repo privés (Gitea) sur le CV public : seuls les github.com
// sont exposés, et toujours avec un schéma https. Cf. décision 2026-06-20 (liens morts vers Gitea privé).

func TestEnsureScheme(t *testing.T) {
	cases := map[string]string{
		"git.etheryale.com/x":  "https://git.etheryale.com/x",
		"github.com/a/b":       "https://github.com/a/b",
		"https://github.com/a": "https://github.com/a",
		"http://demo.local":    "http://demo.local",
		"":                     "",
	}
	for in, want := range cases {
		if got := ensureScheme(in); got != want {
			t.Errorf("ensureScheme(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsGithubURL(t *testing.T) {
	if !isGithubURL("https://github.com/x") {
		t.Error("github.com doit être détecté public")
	}
	if isGithubURL("git.etheryale.com/StillHammer/x") {
		t.Error("Gitea ne doit PAS être détecté github")
	}
}

func TestGithubURLOnlyPublicGithub(t *testing.T) {
	fr := []MPFProject{
		{ID: "a", Name: "A", Links: map[string]string{"repo": "git.etheryale.com/StillHammer/a"}},
		{ID: "b", Name: "B", Links: map[string]string{"repo": "github.com/AlexisTrouve/b"}},
		{ID: "c", Name: "C", Links: map[string]string{"github": "https://github.com/x/c", "demo": "demo.example/c"}},
	}
	out := mapProjects(fr, nil, nil)
	byTitle := map[string]MPFProject{}
	_ = byTitle
	got := map[string]string{}
	demo := map[string]string{}
	links := map[string]int{}
	for _, p := range out {
		got[p.Title] = p.GithubURL
		demo[p.Title] = p.DemoURL
		links[p.Title] = len(p.Links)
	}

	// A : repo Gitea privé → AUCUN GithubURL, AUCUN lien repo réémis
	if got["A"] != "" {
		t.Errorf("A (Gitea privé) ne doit pas avoir de GithubURL, got %q", got["A"])
	}
	if links["A"] != 0 {
		t.Errorf("A ne doit réémettre aucun lien (repo masqué), got %d", links["A"])
	}
	// B : github.com nu → GithubURL avec https
	if got["B"] != "https://github.com/AlexisTrouve/b" {
		t.Errorf("B GithubURL = %q, want https://github.com/AlexisTrouve/b", got["B"])
	}
	// C : github déjà https + demo nu → demo prend https, repo non réémis dans links
	if got["C"] != "https://github.com/x/c" {
		t.Errorf("C GithubURL = %q", got["C"])
	}
	if demo["C"] != "https://demo.example/c" {
		t.Errorf("C DemoURL = %q, want https://demo.example/c", demo["C"])
	}
	if links["C"] != 0 {
		t.Errorf("C ne doit réémettre aucun lien repo (github géré via GithubURL), got %d", links["C"])
	}
}
