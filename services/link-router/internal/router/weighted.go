package router

import (
	"math/rand"

	"github.com/ws-minoro/link-router/internal/store"
)

func SelectWeighted(destinations []store.Destination) store.Destination {
	totalWeight := 0
	for _, d := range destinations {
		totalWeight += d.Weight
	}

	if totalWeight == 0 {
		return destinations[0]
	}

	r := rand.Intn(totalWeight)
	cumulative := 0
	for _, d := range destinations {
		cumulative += d.Weight
		if r < cumulative {
			return d
		}
	}

	return destinations[len(destinations)-1]
}
