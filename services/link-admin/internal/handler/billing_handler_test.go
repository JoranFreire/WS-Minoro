package handler

import (
	"encoding/base64"
	"testing"
)

func basicAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func TestParseBasicAuth_ValidHeader(t *testing.T) {
	user, pass := parseBasicAuth(basicAuthHeader("pagarme", "s3cr3t"))
	if user != "pagarme" || pass != "s3cr3t" {
		t.Fatalf("expected user=pagarme pass=s3cr3t, got user=%q pass=%q", user, pass)
	}
}

func TestParseBasicAuth_PasswordContainingColon(t *testing.T) {
	user, pass := parseBasicAuth(basicAuthHeader("pagarme", "s3cr3t:with:colons"))
	if user != "pagarme" || pass != "s3cr3t:with:colons" {
		t.Fatalf("expected the password to keep embedded colons, got user=%q pass=%q", user, pass)
	}
}

func TestParseBasicAuth_MissingHeader(t *testing.T) {
	user, pass := parseBasicAuth("")
	if user != "" || pass != "" {
		t.Fatalf("expected empty user/pass for missing header, got user=%q pass=%q", user, pass)
	}
}

func TestParseBasicAuth_WrongScheme(t *testing.T) {
	user, pass := parseBasicAuth("Bearer sometoken")
	if user != "" || pass != "" {
		t.Fatalf("expected empty user/pass for a non-Basic scheme, got user=%q pass=%q", user, pass)
	}
}

func TestParseBasicAuth_MalformedBase64(t *testing.T) {
	user, pass := parseBasicAuth("Basic not-valid-base64!!!")
	if user != "" || pass != "" {
		t.Fatalf("expected empty user/pass for malformed base64, got user=%q pass=%q", user, pass)
	}
}
