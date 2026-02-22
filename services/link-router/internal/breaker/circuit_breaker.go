package breaker

import (
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned when the circuit breaker is open and not allowing calls.
var ErrOpen = errors.New("circuit breaker open")

// CircuitBreaker is a simple three-state (closed → open → half-open) breaker.
type CircuitBreaker struct {
	mu        sync.Mutex
	failures  int
	threshold int
	openAt    time.Time
	timeout   time.Duration
}

// New creates a CircuitBreaker that opens after `threshold` consecutive failures
// and attempts recovery after `timeout`.
func New(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{threshold: threshold, timeout: timeout}
}

// Call executes fn under the circuit breaker.
// Returns ErrOpen immediately if the breaker is open.
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	if !cb.openAt.IsZero() {
		if time.Since(cb.openAt) < cb.timeout {
			cb.mu.Unlock()
			return ErrOpen
		}
		// Half-open: let one request through and reset counters.
		cb.failures = 0
		cb.openAt = time.Time{}
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.failures++
		if cb.failures >= cb.threshold {
			cb.openAt = time.Now()
		}
		return err
	}
	cb.failures = 0
	return nil
}
