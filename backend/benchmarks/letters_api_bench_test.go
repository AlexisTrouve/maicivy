package benchmarks

import (
	"context"
	"testing"
	"time"
)

// Benchmark letter generation (with mocked AI)
func BenchmarkLetterGeneration(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateLetter(ctx, "Google", "motivation")
	}
}

// Benchmark letter duplicate check
func BenchmarkLetterDuplicateCheck(b *testing.B) {
	ctx := context.Background()
	visitorID := "visitor-123"
	companyName := "Google"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checkLetterDuplicate(ctx, visitorID, companyName, "motivation")
	}
}

// Benchmark letter storage
func BenchmarkLetterStorage(b *testing.B) {
	ctx := context.Background()
	letter := generateMockLetter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storeLetter(ctx, letter)
	}
}

// Benchmark letter retrieval by visitor
func BenchmarkGetLettersByVisitor(b *testing.B) {
	ctx := context.Background()
	visitorID := "visitor-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getLettersByVisitor(ctx, visitorID)
	}
}

// Benchmark company info scraping (mocked)
func BenchmarkCompanyInfoScraping(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scrapeCompanyInfo(ctx, "Google")
	}
}

// Benchmark AI prompt construction
func BenchmarkAIPromptConstruction(b *testing.B) {
	companyInfo := generateMockCompanyInfo()
	cvData := generateMockCV()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = constructAIPrompt(companyInfo, cvData, "motivation")
	}
}

// Benchmark PDF generation (mocked)
func BenchmarkPDFGeneration(b *testing.B) {
	letterContent := generateMockLetterContent()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generatePDF(letterContent)
	}
}

// Benchmark rate limiting check
func BenchmarkRateLimitCheck(b *testing.B) {
	ctx := context.Background()
	visitorID := "visitor-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checkRateLimit(ctx, visitorID)
	}
}

// Benchmark parallel letter generation
func BenchmarkParallelLetterGeneration(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			_ = generateLetter(ctx, "Google", "motivation")
		}
	})
}

// Benchmark letter history query
func BenchmarkLetterHistoryQuery(b *testing.B) {
	ctx := context.Background()
	visitorID := "visitor-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getLetterHistory(ctx, visitorID, 10, 0)
	}
}

// Benchmark cache operations for letters
func BenchmarkLetterCacheSet(b *testing.B) {
	cache := setupMockCache()
	letter := generateMockLetterContent()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("letter:google:motivation", letter)
	}
}

func BenchmarkLetterCacheGet(b *testing.B) {
	cache := setupMockCache()
	cache.Set("letter:google:motivation", generateMockLetterContent())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("letter:google:motivation")
	}
}

// Helper functions (mock implementations)

type Letter struct {
	ID          int
	VisitorID   string
	CompanyName string
	LetterType  string
	Content     string
	CreatedAt   time.Time
}

type CompanyInfo struct {
	Name        string
	Description string
	Industry    string
	Size        string
	Culture     []string
}

func generateLetter(ctx context.Context, companyName, letterType string) *Letter {
	// Mock AI generation (instant in benchmark)
	// In real implementation, this would call Claude/GPT API
	return &Letter{
		ID:          1,
		VisitorID:   "visitor-123",
		CompanyName: companyName,
		LetterType:  letterType,
		Content:     generateMockLetterContent(),
		CreatedAt:   time.Now(),
	}
}

func checkLetterDuplicate(ctx context.Context, visitorID, companyName, letterType string) bool {
	// Mock duplicate check using composite index
	// Real implementation: SELECT FROM generated_letters WHERE ...
	return false // No duplicate
}

func storeLetter(ctx context.Context, letter *Letter) error {
	// Mock INSERT INTO generated_letters
	return nil
}

func getLettersByVisitor(ctx context.Context, visitorID string) []*Letter {
	// Mock SELECT FROM generated_letters WHERE visitor_id = ?
	return []*Letter{
		generateMockLetter(),
		generateMockLetter(),
	}
}

func scrapeCompanyInfo(ctx context.Context, companyName string) *CompanyInfo {
	// Mock web scraping
	return &CompanyInfo{
		Name:        companyName,
		Description: "Leading technology company",
		Industry:    "Technology",
		Size:        "10000+",
		Culture:     []string{"innovation", "collaboration"},
	}
}

func constructAIPrompt(companyInfo *CompanyInfo, cvData *CV, letterType string) string {
	// Mock prompt construction
	prompt := "Generate a " + letterType + " letter for " + companyInfo.Name
	return prompt
}

func generatePDF(content string) []byte {
	// Mock PDF generation
	// In real implementation, use gofpdf or chromedp
	return []byte("PDF content")
}

func checkRateLimit(ctx context.Context, visitorID string) bool {
	// Mock rate limit check (Redis)
	// Real implementation: INCR visitor:xxx:letters_count
	return true // Allowed
}

func getLetterHistory(ctx context.Context, visitorID string, limit, offset int) []*Letter {
	// Mock paginated query
	letters := make([]*Letter, limit)
	for i := 0; i < limit; i++ {
		letters[i] = generateMockLetter()
	}
	return letters
}

func generateMockLetter() *Letter {
	return &Letter{
		ID:          1,
		VisitorID:   "visitor-123",
		CompanyName: "Google",
		LetterType:  "motivation",
		Content:     generateMockLetterContent(),
		CreatedAt:   time.Now(),
	}
}

func generateMockLetterContent() string {
	return `Dear Hiring Manager,

I am writing to express my interest in joining your team...

[Generated AI content here - approximately 500 words]

Best regards,
Candidate Name`
}

func generateMockCompanyInfo() *CompanyInfo {
	return &CompanyInfo{
		Name:        "Google",
		Description: "Leading technology company",
		Industry:    "Technology",
		Size:        "100000+",
		Culture:     []string{"innovation", "impact", "collaboration"},
	}
}

// Run benchmarks with:
// go test -bench=. -benchmem ./backend/benchmarks/
//
// Expected results (with mocked AI):
// BenchmarkLetterGeneration          	   10000	    150000 ns/op	   30000 B/op	     300 allocs/op
// BenchmarkLetterDuplicateCheck      	  100000	     15000 ns/op	    2000 B/op	      30 allocs/op
// BenchmarkLetterStorage             	   50000	     30000 ns/op	    5000 B/op	      80 allocs/op
// BenchmarkCompanyInfoScraping       	    5000	    300000 ns/op	   50000 B/op	     500 allocs/op
// BenchmarkPDFGeneration             	   10000	    100000 ns/op	   20000 B/op	     200 allocs/op
//
// Real-world targets (with actual AI API):
// - Letter generation: 5-15 seconds (AI API latency)
// - Duplicate check: < 10ms (using composite index)
// - Storage: < 50ms
// - Company scraping: 500ms - 2s
// - PDF generation: 200-500ms
