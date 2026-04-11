package content

import (
	"path/filepath"
	"runtime"
	"testing"
)

func contentDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "content")
}

func TestLoaderIntegration(t *testing.T) {
	dir := contentDir()
	loader := NewLoader(dir)

	if err := loader.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	experiences := loader.GetExperiences()
	if len(experiences) < 5 {
		t.Errorf("Expected at least 5 experiences, got %d", len(experiences))
	}

	// Verify sorted by start_date DESC
	for i := 1; i < len(experiences); i++ {
		if experiences[i].StartDate.After(experiences[i-1].StartDate) {
			t.Errorf("Experiences not sorted: %v after %v", experiences[i].StartDate, experiences[i-1].StartDate)
		}
	}

	skills := loader.GetSkills()
	if len(skills) < 20 {
		t.Errorf("Expected at least 20 skills, got %d", len(skills))
	}

	projects := loader.GetProjects()
	if len(projects) < 10 {
		t.Errorf("Expected at least 10 projects, got %d", len(projects))
	}

	// Verify first experience is the most recent (CoconSystem Cogesco 2026)
	if experiences[0].Title != "Développeur Système SEO — CoconSystem" {
		t.Errorf("First experience should be CoconSystem (2026-02), got %s @ %s", experiences[0].Title, experiences[0].Company)
	}
}
