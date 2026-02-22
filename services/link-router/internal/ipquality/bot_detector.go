package ipquality

import "strings"

// Known bot/crawler User-Agent substrings.
var botSignatures = []string{
	"bot", "crawl", "spider", "slurp", "search", "fetch",
	"mediapartners", "adsbot", "facebookexternalhit",
	"linkedinbot", "twitterbot", "whatsapp", "curl", "python-requests",
	"go-http-client", "java/", "wget",
}

// IsBot returns true if the User-Agent matches a known bot pattern.
func IsBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	for _, sig := range botSignatures {
		if strings.Contains(ua, sig) {
			return true
		}
	}
	return false
}
