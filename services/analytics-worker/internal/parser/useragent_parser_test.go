package parser

import "testing"

func TestParseUserAgent(t *testing.T) {
	cases := []struct {
		name           string
		ua             string
		wantDeviceType string
	}{
		{
			name:           "desktop chrome",
			ua:             "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantDeviceType: "desktop",
		},
		{
			name:           "mobile iphone safari",
			ua:             "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
			wantDeviceType: "mobile",
		},
		{
			name:           "googlebot",
			ua:             "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			wantDeviceType: "bot",
		},
		{
			name:           "empty user agent",
			ua:             "",
			wantDeviceType: "desktop",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseUserAgent(tc.ua)
			if got.DeviceType != tc.wantDeviceType {
				t.Fatalf("ParseUserAgent(%q).DeviceType = %q, want %q", tc.ua, got.DeviceType, tc.wantDeviceType)
			}
		})
	}
}

func TestHashIP_IsDeterministic(t *testing.T) {
	a := HashIP("203.0.113.42")
	b := HashIP("203.0.113.42")
	if a != b {
		t.Fatalf("expected same IP to hash deterministically, got %q and %q", a, b)
	}
}

func TestHashIP_DifferentIPsHashDifferently(t *testing.T) {
	a := HashIP("203.0.113.42")
	b := HashIP("198.51.100.7")
	if a == b {
		t.Fatal("expected different IPs to produce different hashes")
	}
}

func TestHashIP_NeverReturnsThePlaintextIP(t *testing.T) {
	ip := "203.0.113.42"
	got := HashIP(ip)
	if got == ip {
		t.Fatal("HashIP must not return the plaintext IP")
	}
	if len(got) != 16 {
		t.Fatalf("expected a 16-char truncated hash, got length %d", len(got))
	}
}
