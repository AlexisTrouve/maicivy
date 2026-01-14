package validation

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

func init() {
	// Register custom validators on package initialization
	RegisterCustomValidators(Validate)
}

// RegisterCustomValidators registers all custom validation rules
func RegisterCustomValidators(v *validator.Validate) {
	v.RegisterValidation("alpha_space", validateAlphaSpace)
	v.RegisterValidation("safe_url", validateSafeURL)
	v.RegisterValidation("company_name", validateCompanyName)
	v.RegisterValidation("job_title", validateJobTitle)
	v.RegisterValidation("no_html", validateNoHTML)
}

// validateAlphaSpace validates that a string contains only letters and spaces
// Supports international characters (accented letters, etc.)
func validateAlphaSpace(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// Allow letters (including accented), spaces, hyphens, apostrophes
	matched, _ := regexp.MatchString(`^[a-zA-ZÀ-ÿ\s\-']+$`, value)
	return matched
}

// validateSafeURL validates that a URL uses safe schemes (http/https only)
// Prevents javascript:, data:, file: schemes
func validateSafeURL(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	parsedURL, err := url.Parse(value)
	if err != nil {
		return false
	}

	// Only allow http and https schemes
	scheme := strings.ToLower(parsedURL.Scheme)
	return scheme == "http" || scheme == "https"
}

// validateCompanyName validates company names
// Allows letters, numbers, spaces, and common company name characters
func validateCompanyName(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// Allow letters, numbers, spaces, hyphens, apostrophes, periods, ampersands
	// Examples: "Google Inc.", "L'Oréal", "AT&T", "3M Company"
	matched, _ := regexp.MatchString(`^[a-zA-ZÀ-ÿ0-9\s\-'\.&,()]+$`, value)
	return matched
}

// validateJobTitle validates job titles
// Similar to company names but more permissive
func validateJobTitle(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// Allow letters, numbers, spaces, and common job title characters
	// Examples: "Senior Software Engineer", "C++ Developer", "VP of Engineering"
	matched, _ := regexp.MatchString(`^[a-zA-ZÀ-ÿ0-9\s\-'/\\.+#]+$`, value)
	return matched
}

// validateNoHTML ensures no HTML tags are present
// Prevents XSS attacks
func validateNoHTML(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Empty is valid
	}

	// Check for < or > characters (HTML tags)
	matched, _ := regexp.MatchString(`[<>]`, value)
	return !matched // Return true if NO HTML tags found
}

// ValidateEmail validates email format (built-in validator "email" is also available)
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}

	// RFC 5322 simplified regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidateURL validates a URL string
func ValidateURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Must have scheme and host
	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

// SanitizeString removes potentially dangerous characters
// Use this as a second line of defense after validation
func SanitizeString(input string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(input)

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Normalize whitespace (multiple spaces -> single space)
	spaceRegex := regexp.MustCompile(`\s+`)
	sanitized = spaceRegex.ReplaceAllString(sanitized, " ")

	return sanitized
}

// SanitizeHTML removes all HTML tags and dangerous characters
// This is a basic sanitizer; use bluemonday for more complex HTML
func SanitizeHTML(input string) string {
	// Remove HTML tags
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	sanitized := htmlRegex.ReplaceAllString(input, "")

	// Remove script content
	scriptRegex := regexp.MustCompile(`<script[\s\S]*?</script>`)
	sanitized = scriptRegex.ReplaceAllString(sanitized, "")

	// Decode HTML entities (prevent double encoding attacks)
	sanitized = strings.ReplaceAll(sanitized, "&lt;", "<")
	sanitized = strings.ReplaceAll(sanitized, "&gt;", ">")
	sanitized = strings.ReplaceAll(sanitized, "&quot;", "\"")
	sanitized = strings.ReplaceAll(sanitized, "&#x27;", "'")
	sanitized = strings.ReplaceAll(sanitized, "&amp;", "&")

	// Now remove any < > that were decoded
	sanitized = strings.ReplaceAll(sanitized, "<", "")
	sanitized = strings.ReplaceAll(sanitized, ">", "")

	return SanitizeString(sanitized)
}

// IsValidTheme checks if a CV theme is valid
func IsValidTheme(theme string) bool {
	validThemes := []string{
		"backend",
		"frontend",
		"fullstack",
		"cpp",
		"devops",
		"artistic",
		"data-science",
		"mobile",
	}

	theme = strings.ToLower(strings.TrimSpace(theme))
	for _, valid := range validThemes {
		if theme == valid {
			return true
		}
	}
	return false
}

// IsValidLetterType checks if a letter type is valid
func IsValidLetterType(letterType string) bool {
	validTypes := []string{
		"motivation",
		"anti-motivation",
	}

	letterType = strings.ToLower(strings.TrimSpace(letterType))
	for _, valid := range validTypes {
		if letterType == valid {
			return true
		}
	}
	return false
}

// IsValidEventType checks if an analytics event type is valid
func IsValidEventType(eventType string) bool {
	validTypes := []string{
		"page_view",
		"click",
		"cv_theme_change",
		"letter_generated",
		"pdf_downloaded",
		"analytics_viewed",
	}

	eventType = strings.ToLower(strings.TrimSpace(eventType))
	for _, valid := range validTypes {
		if eventType == valid {
			return true
		}
	}
	return false
}

// ValidateRequestSize checks if request body size is within limits
func ValidateRequestSize(size int64, maxSizeMB int) bool {
	maxBytes := int64(maxSizeMB * 1024 * 1024)
	return size <= maxBytes
}

// ContainsSQLInjection checks for common SQL injection patterns
// This is a basic check; parameterized queries are the real defense
func ContainsSQLInjection(input string) bool {
	input = strings.ToLower(input)

	// Common SQL injection patterns
	sqlPatterns := []string{
		"' or '1'='1",
		"' or 1=1",
		"'; drop table",
		"'; delete from",
		"union select",
		"' union select",
		"exec(",
		"execute(",
		"--|",
		"/*",
		"xp_",
	}

	for _, pattern := range sqlPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}

	return false
}

// ContainsXSS checks for common XSS attack patterns
func ContainsXSS(input string) bool {
	input = strings.ToLower(input)

	// Common XSS patterns
	xssPatterns := []string{
		"<script",
		"javascript:",
		"onerror=",
		"onload=",
		"onclick=",
		"<iframe",
		"<embed",
		"<object",
		"eval(",
		"expression(",
	}

	for _, pattern := range xssPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}

	return false
}
