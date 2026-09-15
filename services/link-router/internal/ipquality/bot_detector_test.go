package ipquality

import "testing"

func TestIsBot(t *testing.T) {
	cases := []struct {
		name      string
		userAgent string
		want      bool
	}{
		{"googlebot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", true},
		{"whatsapp preview crawler", "WhatsApp/2.23.1 A", true},
		{"curl", "curl/8.4.0", true},
		{"python requests", "python-requests/2.31.0", true},
		{"go http client", "Go-http-client/1.1", true},
		{"case insensitive match", "MOZILLA/5.0 BOT", true},
		{"regular chrome desktop", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", false},
		{"regular iphone safari", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15", false},
		{"empty user agent", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsBot(tc.userAgent)
			if got != tc.want {
				t.Fatalf("IsBot(%q) = %v, want %v", tc.userAgent, got, tc.want)
			}
		})
	}
}
