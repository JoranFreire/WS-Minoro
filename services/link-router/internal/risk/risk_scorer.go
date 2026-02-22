package risk

import (
	"github.com/ws-minoro/link-router/internal/store"
)

// IsRisky returns true if the destination's risk score exceeds the given threshold.
func IsRisky(dest store.Destination, threshold float64) bool {
	return dest.RiskScore >= threshold
}
