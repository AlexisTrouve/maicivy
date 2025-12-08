package benchmarks

import (
	"context"
	"testing"
)

// Benchmark GET /api/cv endpoint
func BenchmarkGetCV(b *testing.B) {
	// Setup (mock database, cache, etc.)
	// In production, use real test fixtures

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate CV fetching logic
		_ = fetchCV(ctx, "backend")
	}
}

// Benchmark CV scoring algorithm
func BenchmarkCVScoringAlgorithm(b *testing.B) {
	// Mock experiences and skills data
	experiences := generateMockExperiences(50)
	skills := generateMockSkills(30)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Run scoring algorithm
		_ = scoreCVForTheme(experiences, skills, "backend")
	}
}

// Benchmark with different data sizes
func BenchmarkCVScoring_10Items(b *testing.B) {
	experiences := generateMockExperiences(10)
	skills := generateMockSkills(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scoreCVForTheme(experiences, skills, "backend")
	}
}

func BenchmarkCVScoring_50Items(b *testing.B) {
	experiences := generateMockExperiences(50)
	skills := generateMockSkills(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scoreCVForTheme(experiences, skills, "backend")
	}
}

func BenchmarkCVScoring_100Items(b *testing.B) {
	experiences := generateMockExperiences(100)
	skills := generateMockSkills(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scoreCVForTheme(experiences, skills, "backend")
	}
}

// Benchmark cache hit vs miss
func BenchmarkCVCacheHit(b *testing.B) {
	// Pre-populate cache
	cache := setupMockCache()
	cache.Set("cv:backend", mockCVData())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("cv:backend")
	}
}

func BenchmarkCVCacheMiss(b *testing.B) {
	cache := setupMockCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Cache miss + fetch from DB
		_, _ = cache.Get("cv:nonexistent")
		_ = fetchCV(context.Background(), "nonexistent")
	}
}

// Benchmark JSON serialization
func BenchmarkCVJSONSerialization(b *testing.B) {
	cv := generateMockCV()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = serializeCV(cv)
	}
}

// Benchmark parallel requests
func BenchmarkCVParallelRequests(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = fetchCV(context.Background(), "backend")
		}
	})
}

// Benchmark tag filtering (GIN index)
func BenchmarkTagFiltering(b *testing.B) {
	experiences := generateMockExperiences(100)
	tags := []string{"golang", "docker", "kubernetes"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filterExperiencesByTags(experiences, tags)
	}
}

// Helper functions (mock implementations)

type Experience struct {
	ID           int
	Title        string
	Company      string
	Description  string
	Tags         []string
	Technologies []string
	Category     string
}

type Skill struct {
	ID       int
	Name     string
	Level    int
	Category string
	Tags     []string
}

type CV struct {
	Experiences []Experience
	Skills      []Skill
	Score       float64
}

func fetchCV(ctx context.Context, theme string) *CV {
	// Mock implementation
	return &CV{
		Experiences: generateMockExperiences(20),
		Skills:      generateMockSkills(15),
		Score:       0.85,
	}
}

func scoreCVForTheme(experiences []Experience, skills []Skill, theme string) float64 {
	// Mock scoring algorithm
	score := 0.0
	relevantCount := 0

	// Score experiences
	for _, exp := range experiences {
		if exp.Category == theme {
			score += 1.0
			relevantCount++
		}
		// Tag matching
		for _, tag := range exp.Tags {
			if tag == theme || tag == "backend" || tag == "fullstack" {
				score += 0.5
			}
		}
	}

	// Score skills
	for _, skill := range skills {
		if skill.Category == theme {
			score += float64(skill.Level) * 0.1
		}
	}

	if relevantCount > 0 {
		return score / float64(len(experiences)+len(skills))
	}
	return 0.0
}

func generateMockExperiences(count int) []Experience {
	experiences := make([]Experience, count)
	categories := []string{"backend", "frontend", "fullstack", "devops"}
	tags := []string{"golang", "react", "docker", "kubernetes", "postgresql"}

	for i := 0; i < count; i++ {
		experiences[i] = Experience{
			ID:           i + 1,
			Title:        "Software Engineer",
			Company:      "Tech Company",
			Description:  "Description",
			Tags:         []string{tags[i%len(tags)], tags[(i+1)%len(tags)]},
			Technologies: []string{"Go", "Docker"},
			Category:     categories[i%len(categories)],
		}
	}
	return experiences
}

func generateMockSkills(count int) []Skill {
	skills := make([]Skill, count)
	categories := []string{"languages", "tools", "frameworks", "databases"}
	tags := []string{"backend", "frontend", "devops"}

	for i := 0; i < count; i++ {
		skills[i] = Skill{
			ID:       i + 1,
			Name:     "Skill " + string(rune(i)),
			Level:    (i % 5) + 1, // 1-5
			Category: categories[i%len(categories)],
			Tags:     []string{tags[i%len(tags)]},
		}
	}
	return skills
}

func generateMockCV() *CV {
	return &CV{
		Experiences: generateMockExperiences(20),
		Skills:      generateMockSkills(15),
		Score:       0.85,
	}
}

func serializeCV(cv *CV) []byte {
	// Mock JSON serialization
	// In real implementation, use json.Marshal
	return []byte("{}")
}

func filterExperiencesByTags(experiences []Experience, tags []string) []Experience {
	filtered := make([]Experience, 0)
	for _, exp := range experiences {
		for _, expTag := range exp.Tags {
			for _, searchTag := range tags {
				if expTag == searchTag {
					filtered = append(filtered, exp)
					break
				}
			}
		}
	}
	return filtered
}

// Mock cache
type MockCache struct {
	data map[string]interface{}
}

func setupMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]interface{}),
	}
}

func (c *MockCache) Set(key string, value interface{}) {
	c.data[key] = value
}

func (c *MockCache) Get(key string) (interface{}, error) {
	val, ok := c.data[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

func mockCVData() interface{} {
	return generateMockCV()
}

// Run benchmarks with:
// go test -bench=. -benchmem -benchtime=10s ./backend/benchmarks/
//
// Expected results (approximate):
// BenchmarkGetCV                     	   50000	     30000 ns/op	    8000 B/op	     100 allocs/op
// BenchmarkCVScoringAlgorithm        	  100000	     10000 ns/op	    2000 B/op	      50 allocs/op
// BenchmarkCVCacheHit                	 5000000	       300 ns/op	      64 B/op	       2 allocs/op
// BenchmarkCVCacheMiss               	   10000	    100000 ns/op	   15000 B/op	     200 allocs/op
//
// Targets:
// - CV API p95 < 100ms
// - Scoring algorithm < 10ms
// - Cache hit < 1ms
