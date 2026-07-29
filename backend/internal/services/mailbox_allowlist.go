package services

import "strings"

// ParseMailboxAllowlist parse MAILBOX_ALLOWED_DOMAINS ("malt.fr:malt,malt.com:malt") en map
// domaine(lowercase)→platform. Entrées malformées (pas de ':', domaine ou platform vide) ignorées
// silencieusement — étendre l'allowlist = ajouter une paire dans la variable d'env, pas de code à toucher.
func ParseMailboxAllowlist(raw string) map[string]string {
	out := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		domain := strings.ToLower(strings.TrimSpace(parts[0]))
		platform := strings.TrimSpace(parts[1])
		if domain == "" || platform == "" {
			continue
		}
		out[domain] = platform
	}
	return out
}

// MatchMailboxDomain dit si `domain` est couvert par l'allowlist — exact OU sous-domaine, avec une
// frontière "." STRICTE : "evilmalt.fr" ne matche PAS "malt.fr" (simple suffixe de caractères aurait
// laissé passer ce bypass), mais "mail.malt.fr" matche bien "malt.fr". Retourne le label platform associé.
func MatchMailboxDomain(domain string, allowlist map[string]string) (string, bool) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return "", false
	}
	if platform, ok := allowlist[domain]; ok {
		return platform, true
	}
	for allowed, platform := range allowlist {
		if strings.HasSuffix(domain, "."+allowed) {
			return platform, true
		}
	}
	return "", false
}

// EmailDomain extrait le domaine (lowercase) d'une adresse email brute ("user@host.tld" → "host.tld").
// Pas de '@' (ou '@' en dernier caractère) → chaîne vide, rien à matcher.
func EmailDomain(address string) string {
	i := strings.LastIndexByte(address, '@')
	if i < 0 || i == len(address)-1 {
		return ""
	}
	return strings.ToLower(address[i+1:])
}
