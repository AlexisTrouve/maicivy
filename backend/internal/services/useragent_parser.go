// backend/internal/services/useragent_parser.go
package services

import (
	"strings"
)

// UserAgentParser parse et analyse les User-Agent strings
type UserAgentParser struct{}

// NewUserAgentParser crée une nouvelle instance du parser
func NewUserAgentParser() *UserAgentParser {
	return &UserAgentParser{}
}

// Parse analyse un User-Agent et retourne les informations du device
func (p *UserAgentParser) Parse(userAgent string) (DeviceInfo, bool) {
	deviceInfo := DeviceInfo{
		Browser:    "Unknown",
		OS:         "Unknown",
		DeviceType: "desktop",
		IsBot:      false,
	}

	// Vérifier si c'est un bot
	isBot := p.isBot(userAgent)
	deviceInfo.IsBot = isBot

	// Parser le browser
	deviceInfo.Browser = p.parseBrowser(userAgent)

	// Parser l'OS
	deviceInfo.OS = p.parseOS(userAgent)

	// Détecter le type de device
	deviceInfo.DeviceType = p.parseDeviceType(userAgent)

	return deviceInfo, isBot
}

// parseBrowser détecte le navigateur
func (p *UserAgentParser) parseBrowser(ua string) string {
	uaLower := strings.ToLower(ua)

	// Ordre important : vérifier les plus spécifiques d'abord
	browsers := []struct {
		pattern string
		name    string
	}{
		{"edg/", "Edge"},
		{"chrome/", "Chrome"},
		{"firefox/", "Firefox"},
		{"safari/", "Safari"},
		{"opera/", "Opera"},
		{"postman", "Postman"},
		{"curl", "curl"},
		{"wget", "wget"},
		{"httpie", "HTTPie"},
		{"insomnia", "Insomnia"},
		{"linkedinapp", "LinkedIn App"},
		{"msie", "Internet Explorer"},
		{"trident", "Internet Explorer"},
	}

	for _, browser := range browsers {
		if strings.Contains(uaLower, browser.pattern) {
			return browser.name
		}
	}

	return "Unknown"
}

// parseOS détecte le système d'exploitation
func (p *UserAgentParser) parseOS(ua string) string {
	uaLower := strings.ToLower(ua)

	oses := []struct {
		pattern string
		name    string
	}{
		{"windows nt 10", "Windows 10"},
		{"windows nt 6.3", "Windows 8.1"},
		{"windows nt 6.2", "Windows 8"},
		{"windows nt 6.1", "Windows 7"},
		{"windows", "Windows"},
		{"mac os x", "macOS"},
		{"macintosh", "macOS"},
		{"iphone", "iOS"},
		{"ipad", "iPadOS"},
		{"android", "Android"},
		{"linux", "Linux"},
		{"ubuntu", "Ubuntu"},
		{"fedora", "Fedora"},
		{"debian", "Debian"},
	}

	for _, os := range oses {
		if strings.Contains(uaLower, os.pattern) {
			return os.name
		}
	}

	return "Unknown"
}

// parseDeviceType détecte le type de device
func (p *UserAgentParser) parseDeviceType(ua string) string {
	uaLower := strings.ToLower(ua)

	// Mobile
	mobilePatterns := []string{
		"mobile",
		"iphone",
		"ipod",
		"android",
		"blackberry",
		"windows phone",
		"opera mini",
	}

	for _, pattern := range mobilePatterns {
		if strings.Contains(uaLower, pattern) {
			return "mobile"
		}
	}

	// Tablet
	tabletPatterns := []string{
		"ipad",
		"tablet",
		"kindle",
	}

	for _, pattern := range tabletPatterns {
		if strings.Contains(uaLower, pattern) {
			return "tablet"
		}
	}

	// Tools (developer tools)
	toolPatterns := []string{
		"postman",
		"curl",
		"wget",
		"httpie",
		"insomnia",
	}

	for _, pattern := range toolPatterns {
		if strings.Contains(uaLower, pattern) {
			return "tool"
		}
	}

	// Default: desktop
	return "desktop"
}

// isBot détecte si le User-Agent est un bot
func (p *UserAgentParser) isBot(ua string) bool {
	uaLower := strings.ToLower(ua)

	botPatterns := []string{
		"bot",
		"crawler",
		"spider",
		"scraper",
		"googlebot",
		"bingbot",
		"linkedinbot",
		"facebookexternalhit",
		"slackbot",
		"twitterbot",
		"whatsapp",
		"telegram",
	}

	for _, pattern := range botPatterns {
		if strings.Contains(uaLower, pattern) {
			return true
		}
	}

	return false
}

// IsRecruiterTool détecte si le User-Agent est un outil de recrutement
func (p *UserAgentParser) IsRecruiterTool(ua string) bool {
	uaLower := strings.ToLower(ua)

	recruiterPatterns := []string{
		"linkedinapp",
		"linkedin",
		"greenhouse",
		"lever",
		"workday",
		"workable",
		"applicantstack",
		"jobvite",
		"smartrecruiters",
		"icims",
		"bullhorn",
		"taleo",
	}

	for _, pattern := range recruiterPatterns {
		if strings.Contains(uaLower, pattern) {
			return true
		}
	}

	return false
}

// IsDeveloperTool détecte si le User-Agent est un outil de développeur
func (p *UserAgentParser) IsDeveloperTool(ua string) bool {
	uaLower := strings.ToLower(ua)

	devPatterns := []string{
		"postman",
		"curl",
		"wget",
		"httpie",
		"insomnia",
		"paw",
		"restclient",
		"httpclient",
	}

	for _, pattern := range devPatterns {
		if strings.Contains(uaLower, pattern) {
			return true
		}
	}

	return false
}

// GetDetailedInfo retourne des informations détaillées sur le User-Agent
func (p *UserAgentParser) GetDetailedInfo(ua string) map[string]interface{} {
	deviceInfo, isBot := p.Parse(ua)

	return map[string]interface{}{
		"browser":          deviceInfo.Browser,
		"os":               deviceInfo.OS,
		"device_type":      deviceInfo.DeviceType,
		"is_bot":           isBot,
		"is_recruiter_tool": p.IsRecruiterTool(ua),
		"is_developer_tool": p.IsDeveloperTool(ua),
		"is_mobile":        deviceInfo.DeviceType == "mobile",
		"is_tablet":        deviceInfo.DeviceType == "tablet",
		"raw_user_agent":   ua,
	}
}
