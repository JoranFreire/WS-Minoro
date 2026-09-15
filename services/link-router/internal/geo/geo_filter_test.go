package geo

import (
	"testing"

	"github.com/ws-minoro/link-router/internal/store"
)

func TestFilterByCountry_EmptyCountryCodeAllowsAll(t *testing.T) {
	dests := []store.Destination{
		{ID: "a", AllowedCountries: []string{"BR"}},
		{ID: "b", AllowedCountries: nil},
	}
	got := FilterByCountry(dests, "")
	if len(got) != 2 {
		t.Fatalf("expected all destinations returned, got %d", len(got))
	}
}

func TestFilterByCountry_UnrestrictedDestinationAlwaysAllowed(t *testing.T) {
	dests := []store.Destination{
		{ID: "a", AllowedCountries: nil},
	}
	got := FilterByCountry(dests, "US")
	if len(got) != 1 {
		t.Fatalf("expected unrestricted destination to pass, got %d", len(got))
	}
}

func TestFilterByCountry_MatchingCountryIncluded(t *testing.T) {
	dests := []store.Destination{
		{ID: "a", AllowedCountries: []string{"BR", "US"}},
	}
	got := FilterByCountry(dests, "BR")
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("expected destination a to match, got %+v", got)
	}
}

func TestFilterByCountry_NonMatchingCountryExcluded(t *testing.T) {
	dests := []store.Destination{
		{ID: "a", AllowedCountries: []string{"BR", "US"}},
	}
	got := FilterByCountry(dests, "FR")
	if len(got) != 0 {
		t.Fatalf("expected no destinations, got %d", len(got))
	}
}

func TestFilterByCountry_MixedRestrictedAndUnrestricted(t *testing.T) {
	dests := []store.Destination{
		{ID: "restricted-match", AllowedCountries: []string{"FR"}},
		{ID: "restricted-nomatch", AllowedCountries: []string{"DE"}},
		{ID: "unrestricted", AllowedCountries: nil},
	}
	got := FilterByCountry(dests, "FR")
	if len(got) != 2 {
		t.Fatalf("expected 2 destinations, got %d: %+v", len(got), got)
	}
	ids := map[string]bool{got[0].ID: true, got[1].ID: true}
	if !ids["restricted-match"] || !ids["unrestricted"] {
		t.Fatalf("unexpected result set: %+v", got)
	}
}
