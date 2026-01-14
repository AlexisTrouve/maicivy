// backend/internal/services/profile_detector_test.go
package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock ClearbitClient
type MockClearbitClient struct {
	mock.Mock
}

func (m *MockClearbitClient) EnrichByIP(ctx context.Context, hashedIP, realIP string) (map[string]interface{}, error) {
	args := m.Called(ctx, hashedIP, realIP)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// Mock UserAgentParser
type MockUserAgentParser struct {
	mock.Mock
}

func (m *MockUserAgentParser) Parse(userAgent string) (DeviceInfo, bool) {
	args := m.Called(userAgent)
	return args.Get(0).(DeviceInfo), args.Bool(1)
}

// Test: isRecruiterBot
func TestIsRecruiterBot(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  bool
	}{
		{
			name:      "LinkedIn Bot",
			userAgent: "LinkedInBot/2.0",
			expected:  true,
		},
		{
			name:      "HubSpot",
			userAgent: "HubSpot/1.0",
			expected:  true,
		},
		{
			name:      "Workable",
			userAgent: "Workable/1.0",
			expected:  true,
		},
		{
			name:      "Normal Chrome",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
			expected:  false,
		},
		{
			name:      "Firefox",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; rv:120.0) Gecko/20100101 Firefox/120.0",
			expected:  false,
		},
		{
			name:      "Lever Bot",
			userAgent: "LeverBot/1.0",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRecruiterBot(tt.userAgent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test: calculateFinalConfidence
func TestCalculateFinalConfidence(t *testing.T) {
	tests := []struct {
		name            string
		userAgentScore  int
		refererScore    int
		ipScore         int
		expectedMin     int
		expectedMax     int
	}{
		{
			name:           "All high scores",
			userAgentScore: 30,
			refererScore:   20,
			ipScore:        50,
			expectedMin:    30,
			expectedMax:    40,
		},
		{
			name:           "IP only",
			userAgentScore: 0,
			refererScore:   0,
			ipScore:        50,
			expectedMin:    24,
			expectedMax:    26,
		},
		{
			name:           "UserAgent only",
			userAgentScore: 30,
			refererScore:   0,
			ipScore:        0,
			expectedMin:    8,
			expectedMax:    10,
		},
		{
			name:           "Referer only",
			userAgentScore: 0,
			refererScore:   20,
			ipScore:        0,
			expectedMin:    3,
			expectedMax:    5,
		},
		{
			name:           "All max",
			userAgentScore: 100,
			refererScore:   100,
			ipScore:        100,
			expectedMin:    99,
			expectedMax:    100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateFinalConfidence(tt.userAgentScore, tt.refererScore, tt.ipScore)
			assert.GreaterOrEqual(t, result, tt.expectedMin)
			assert.LessOrEqual(t, result, tt.expectedMax)
			assert.LessOrEqual(t, result, 100, "Confidence should never exceed 100")
		})
	}
}

// Test: hashIP
func TestHashIP(t *testing.T) {
	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	hash1 := hashIP(ip1)
	hash2 := hashIP(ip2)

	// Vérifier que les hashs sont différents pour des IPs différentes
	assert.NotEqual(t, hash1, hash2)

	// Vérifier que le hash est déterministe (même IP = même hash)
	hash1_2 := hashIP(ip1)
	assert.Equal(t, hash1, hash1_2)

	// Vérifier la longueur du hash SHA-256 (64 caractères hex)
	assert.Len(t, hash1, 64)
}

// Test: detectFromUserAgent
func TestDetectFromUserAgent(t *testing.T) {
	mockClearbit := new(MockClearbitClient)
	mockUAParser := new(MockUserAgentParser)

	service := &ProfileDetectorService{
		clearbitClient: mockClearbit,
		uaParser:       mockUAParser,
	}

	tests := []struct {
		name              string
		userAgent         string
		expectedScore     int
		expectedType      ProfileType
	}{
		{
			name:          "LinkedIn App",
			userAgent:     "LinkedInApp/1.0",
			expectedScore: 30,
			expectedType:  ProfileTypeRecruiter,
		},
		{
			name:          "Greenhouse",
			userAgent:     "Greenhouse/1.0",
			expectedScore: 30,
			expectedType:  ProfileTypeRecruiter,
		},
		{
			name:          "Postman",
			userAgent:     "Postman/1.0",
			expectedScore: 20,
			expectedType:  ProfileTypeDeveloper,
		},
		{
			name:          "curl",
			userAgent:     "curl/7.68.0",
			expectedScore: 20,
			expectedType:  ProfileTypeDeveloper,
		},
		{
			name:          "Normal browser",
			userAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
			expectedScore: 0,
			expectedType:  ProfileTypeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, profileType := service.detectFromUserAgent(tt.userAgent)
			assert.Equal(t, tt.expectedScore, score)
			assert.Equal(t, tt.expectedType, profileType)
		})
	}
}

// Test: detectFromReferer
func TestDetectFromReferer(t *testing.T) {
	mockClearbit := new(MockClearbitClient)
	mockUAParser := new(MockUserAgentParser)

	service := &ProfileDetectorService{
		clearbitClient: mockClearbit,
		uaParser:       mockUAParser,
	}

	tests := []struct {
		name          string
		referer       string
		expectedScore int
		expectedType  ProfileType
	}{
		{
			name:          "LinkedIn Jobs",
			referer:       "https://www.linkedin.com/jobs/view/12345",
			expectedScore: 20,
			expectedType:  ProfileTypeRecruiter,
		},
		{
			name:          "LinkedIn Recruiter",
			referer:       "https://www.linkedin.com/recruiter/projects/12345",
			expectedScore: 20,
			expectedType:  ProfileTypeRecruiter,
		},
		{
			name:          "LinkedIn Generic",
			referer:       "https://www.linkedin.com/in/john-doe",
			expectedScore: 10,
			expectedType:  ProfileTypeRecruiter,
		},
		{
			name:          "Indeed",
			referer:       "https://www.indeed.com/jobs?q=developer",
			expectedScore: 15,
			expectedType:  ProfileTypeRecruiter,
		},
		{
			name:          "GitHub",
			referer:       "https://github.com/user/repo",
			expectedScore: 10,
			expectedType:  ProfileTypeDeveloper,
		},
		{
			name:          "No referer",
			referer:       "",
			expectedScore: 0,
			expectedType:  ProfileTypeOther,
		},
		{
			name:          "Google",
			referer:       "https://www.google.com/search?q=developer",
			expectedScore: 0,
			expectedType:  ProfileTypeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, profileType := service.detectFromReferer(tt.referer)
			assert.Equal(t, tt.expectedScore, score)
			assert.Equal(t, tt.expectedType, profileType)
		})
	}
}

// Test: analyzeEnrichmentData
func TestAnalyzeEnrichmentData(t *testing.T) {
	mockClearbit := new(MockClearbitClient)
	mockUAParser := new(MockUserAgentParser)

	service := &ProfileDetectorService{
		clearbitClient: mockClearbit,
		uaParser:       mockUAParser,
	}

	tests := []struct {
		name          string
		data          map[string]interface{}
		expectedScore int
		expectedType  ProfileType
	}{
		{
			name: "CTO",
			data: map[string]interface{}{
				"job_title": "CTO",
			},
			expectedScore: 50,
			expectedType:  ProfileTypeCTO,
		},
		{
			name: "CEO",
			data: map[string]interface{}{
				"job_title": "Chief Executive Officer",
			},
			expectedScore: 50,
			expectedType:  ProfileTypeCEO,
		},
		{
			name: "Tech Lead",
			data: map[string]interface{}{
				"job_title": "Tech Lead",
			},
			expectedScore: 40,
			expectedType:  ProfileTypeTechLead,
		},
		{
			name: "Recruiter company",
			data: map[string]interface{}{
				"company_type": "Recruiting",
			},
			expectedScore: 40,
			expectedType:  ProfileTypeRecruiter,
		},
		{
			name: "Recruiter title",
			data: map[string]interface{}{
				"job_title": "Senior Recruiter",
			},
			expectedScore: 40,
			expectedType:  ProfileTypeRecruiter,
		},
		{
			name: "HR",
			data: map[string]interface{}{
				"job_title": "HR Manager",
			},
			expectedScore: 40,
			expectedType:  ProfileTypeRecruiter,
		},
		{
			name:          "No data",
			data:          map[string]interface{}{},
			expectedScore: 0,
			expectedType:  ProfileTypeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, profileType := service.analyzeEnrichmentData(tt.data)
			assert.Equal(t, tt.expectedScore, score)
			assert.Equal(t, tt.expectedType, profileType)
		})
	}
}

// Test: ShouldBypassAccessGate
func TestShouldBypassAccessGate(t *testing.T) {
	mockClearbit := new(MockClearbitClient)
	mockUAParser := new(MockUserAgentParser)

	service := &ProfileDetectorService{
		clearbitClient: mockClearbit,
		uaParser:       mockUAParser,
	}

	tests := []struct {
		name     string
		profile  *DetectedProfile
		expected bool
	}{
		{
			name: "Recruiter with high confidence",
			profile: &DetectedProfile{
				ProfileType: ProfileTypeRecruiter,
				Confidence:  80,
			},
			expected: true,
		},
		{
			name: "CTO with high confidence",
			profile: &DetectedProfile{
				ProfileType: ProfileTypeCTO,
				Confidence:  90,
			},
			expected: true,
		},
		{
			name: "Tech Lead with medium confidence",
			profile: &DetectedProfile{
				ProfileType: ProfileTypeTechLead,
				Confidence:  60,
			},
			expected: true,
		},
		{
			name: "CEO with high confidence",
			profile: &DetectedProfile{
				ProfileType: ProfileTypeCEO,
				Confidence:  95,
			},
			expected: true,
		},
		{
			name: "Developer (no bypass)",
			profile: &DetectedProfile{
				ProfileType: ProfileTypeDeveloper,
				Confidence:  80,
			},
			expected: false,
		},
		{
			name: "Other (no bypass)",
			profile: &DetectedProfile{
				ProfileType: ProfileTypeOther,
				Confidence:  80,
			},
			expected: false,
		},
		{
			name: "Recruiter with low confidence",
			profile: &DetectedProfile{
				ProfileType: ProfileTypeRecruiter,
				Confidence:  50,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ShouldBypassAccessGate(tt.profile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark: hashIP
func BenchmarkHashIP(b *testing.B) {
	ip := "192.168.1.1"
	for i := 0; i < b.N; i++ {
		hashIP(ip)
	}
}

// Benchmark: calculateFinalConfidence
func BenchmarkCalculateFinalConfidence(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculateFinalConfidence(30, 20, 50)
	}
}
