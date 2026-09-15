package router

import (
	"testing"

	"github.com/ws-minoro/link-router/internal/store"
)

func TestSelectWeighted_SingleDestination(t *testing.T) {
	dests := []store.Destination{{ID: "only", Weight: 5}}
	got := SelectWeighted(dests)
	if got.ID != "only" {
		t.Fatalf("expected only destination, got %v", got.ID)
	}
}

func TestSelectWeighted_ZeroTotalWeightReturnsFirst(t *testing.T) {
	dests := []store.Destination{
		{ID: "a", Weight: 0},
		{ID: "b", Weight: 0},
	}
	got := SelectWeighted(dests)
	if got.ID != "a" {
		t.Fatalf("expected first destination on zero weight, got %v", got.ID)
	}
}

func TestSelectWeighted_OnlyPicksAmongProvidedDestinations(t *testing.T) {
	dests := []store.Destination{
		{ID: "a", Weight: 1},
		{ID: "b", Weight: 2},
		{ID: "c", Weight: 3},
	}
	valid := map[string]bool{"a": true, "b": true, "c": true}

	for range 200 {
		got := SelectWeighted(dests)
		if !valid[got.ID] {
			t.Fatalf("unexpected destination selected: %v", got.ID)
		}
	}
}

func TestSelectWeighted_ZeroWeightDestinationNeverWinsAgainstPositiveWeights(t *testing.T) {
	dests := []store.Destination{
		{ID: "never", Weight: 0},
		{ID: "always", Weight: 10},
	}
	for range 100 {
		got := SelectWeighted(dests)
		if got.ID != "always" {
			t.Fatalf("expected zero-weight destination to never win, got %v", got.ID)
		}
	}
}

func TestSelectWeighted_DistributionRoughlyMatchesWeights(t *testing.T) {
	dests := []store.Destination{
		{ID: "a", Weight: 1},
		{ID: "b", Weight: 9},
	}

	const trials = 10000
	counts := map[string]int{}
	for range trials {
		got := SelectWeighted(dests)
		counts[got.ID]++
	}

	// "b" has 9x the weight of "a"; assert it dominates the distribution
	// with generous tolerance to avoid a flaky test.
	if counts["b"] <= counts["a"]*3 {
		t.Fatalf("expected b to heavily dominate distribution, got a=%d b=%d", counts["a"], counts["b"])
	}
}
