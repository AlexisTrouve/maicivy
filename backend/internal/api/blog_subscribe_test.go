package api

import (
	"testing"

	"maicivy/internal/models"
)

// distinctProjectNames = la source des topics suivables : project_name uniques, non vides, triés.
func TestDistinctProjectNames(t *testing.T) {
	posts := []models.BlogPost{
		{ProjectName: "tech"}, {ProjectName: "Drifterra"}, {ProjectName: "tech"},
		{ProjectName: ""}, {ProjectName: "veille"},
	}
	got := distinctProjectNames(posts)
	want := []string{"Drifterra", "tech", "veille"} // dédup + trié + saut des vides
	if len(got) != len(want) {
		t.Fatalf("distinctProjectNames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("distinctProjectNames = %v, want %v", got, want)
		}
	}
}

// cleanTopics nettoie les topics soumis : trim + drop des vides + dédup, ordre préservé.
func TestCleanTopics(t *testing.T) {
	got := cleanTopics([]string{" Drifterra ", "", "tech", "Drifterra", "  "})
	want := []string{"Drifterra", "tech"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("cleanTopics = %v, want %v", got, want)
	}
}
