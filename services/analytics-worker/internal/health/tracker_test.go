package health

import "testing"

func TestTracker_StartsHealthy(t *testing.T) {
	tr := NewTracker()
	if !tr.Healthy() {
		t.Fatal("expected new tracker to start healthy")
	}
}

func TestTracker_UnhealthyAfterMaxConsecutiveFailures(t *testing.T) {
	tr := NewTracker()
	for range maxConsecutiveFailures {
		tr.RecordFailure()
	}
	if tr.Healthy() {
		t.Fatal("expected tracker to be unhealthy after max consecutive failures")
	}
}

func TestTracker_StaysHealthyBelowThreshold(t *testing.T) {
	tr := NewTracker()
	for range maxConsecutiveFailures - 1 {
		tr.RecordFailure()
	}
	if !tr.Healthy() {
		t.Fatal("expected tracker to still be healthy just below the threshold")
	}
}

func TestTracker_SuccessResetsFailureCount(t *testing.T) {
	tr := NewTracker()
	for range maxConsecutiveFailures - 1 {
		tr.RecordFailure()
	}
	tr.RecordSuccess()
	for range maxConsecutiveFailures - 1 {
		tr.RecordFailure()
	}
	if !tr.Healthy() {
		t.Fatal("expected success to reset the consecutive failure count")
	}
}
