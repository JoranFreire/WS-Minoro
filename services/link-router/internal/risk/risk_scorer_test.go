package risk

import (
	"testing"

	"github.com/ws-minoro/link-router/internal/store"
)

func TestIsRisky(t *testing.T) {
	cases := []struct {
		name      string
		score     float64
		threshold float64
		want      bool
	}{
		{"below threshold", 0.3, 0.7, false},
		{"equal to threshold", 0.7, 0.7, true},
		{"above threshold", 0.9, 0.7, true},
		{"zero score never risky at positive threshold", 0.0, 0.1, false},
		{"zero threshold flags any score", 0.01, 0.0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dest := store.Destination{RiskScore: tc.score}
			got := IsRisky(dest, tc.threshold)
			if got != tc.want {
				t.Fatalf("IsRisky(score=%v, threshold=%v) = %v, want %v", tc.score, tc.threshold, got, tc.want)
			}
		})
	}
}
