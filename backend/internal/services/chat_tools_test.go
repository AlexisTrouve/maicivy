package services

import "testing"

// Verrouille la présence des tools maiProFiles dans buildTools (dont get_profile + get_stats ajoutés
// pour un accès complet). buildTools n'utilise pas le client Anthropic → testable sur un zéro-value.
func TestChatBuildTools_IncludesAllReadTools(t *testing.T) {
	s := &ChatService{}
	tools := s.buildTools()

	names := map[string]bool{}
	for _, tool := range tools {
		if tool.OfTool != nil {
			names[tool.OfTool.Name] = true
		}
	}

	want := []string{
		"get_project", "list_projects", "list_skills", "get_experience",
		"get_profile", "get_stats", "search_projects", "show_blog_list",
	}
	for _, w := range want {
		if !names[w] {
			t.Fatalf("tool %q manquant dans buildTools", w)
		}
	}
}
