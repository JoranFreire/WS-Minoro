package health

import "sync"

// maxConsecutiveFailures is how many fully-failed click events in a row mark
// the worker unhealthy. A single flaky write to one sink shouldn't trigger a
// restart, but every sink failing repeatedly in a row means the worker is
// stuck (e.g. all downstream stores are unreachable) and k8s should recycle
// the pod.
const maxConsecutiveFailures = 10

// Tracker records whether recently processed click events succeeded, so the
// HTTP health endpoint has something real to report instead of always
// answering OK regardless of whether clicks are actually being persisted.
type Tracker struct {
	mu                  sync.Mutex
	consecutiveFailures int
}

func NewTracker() *Tracker {
	return &Tracker{}
}

func (t *Tracker) RecordSuccess() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.consecutiveFailures = 0
}

func (t *Tracker) RecordFailure() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.consecutiveFailures++
}

func (t *Tracker) Healthy() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.consecutiveFailures < maxConsecutiveFailures
}
