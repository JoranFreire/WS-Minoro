package geo

import "github.com/ws-minoro/link-router/internal/store"

// FilterByCountry removes destinations whose allowed_countries list does not
// include the given country code. Destinations with an empty list allow all
// countries. An empty countryCode (unknown origin) is always allowed.
func FilterByCountry(dests []store.Destination, countryCode string) []store.Destination {
	if countryCode == "" {
		return dests
	}
	result := make([]store.Destination, 0, len(dests))
	for _, d := range dests {
		if len(d.AllowedCountries) == 0 {
			result = append(result, d)
			continue
		}
		for _, c := range d.AllowedCountries {
			if c == countryCode {
				result = append(result, d)
				break
			}
		}
	}
	return result
}
