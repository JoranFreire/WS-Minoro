package breaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedAllowsCalls(t *testing.T) {
	cb := New(3, time.Minute)

	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected fn to be called while closed")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := New(3, time.Minute)
	boom := errors.New("boom")

	for i := range 3 {
		if err := cb.Call(func() error { return boom }); !errors.Is(err, boom) {
			t.Fatalf("call %d: expected boom, got %v", i, err)
		}
	}

	// Breaker should now be open and reject without calling fn.
	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("expected ErrOpen, got %v", err)
	}
	if called {
		t.Fatal("fn should not be called while breaker is open")
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	cb := New(3, time.Minute)
	boom := errors.New("boom")

	_ = cb.Call(func() error { return boom })
	_ = cb.Call(func() error { return boom })
	_ = cb.Call(func() error { return nil }) // resets failures to 0
	_ = cb.Call(func() error { return boom })
	_ = cb.Call(func() error { return boom })

	// Only 2 consecutive failures since the reset — still below threshold of 3.
	err := cb.Call(func() error { return nil })
	if err != nil {
		t.Fatalf("expected breaker to still be closed, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := New(1, 10*time.Millisecond)
	boom := errors.New("boom")

	if err := cb.Call(func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
	if err := cb.Call(func() error { return nil }); !errors.Is(err, ErrOpen) {
		t.Fatalf("expected ErrOpen immediately after opening, got %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected half-open call to succeed, got %v", err)
	}
	if !called {
		t.Fatal("expected fn to be called once timeout elapsed")
	}
}

func TestCircuitBreaker_ConcurrentCallsAreSafe(t *testing.T) {
	cb := New(1000, time.Minute)
	done := make(chan struct{})

	for range 50 {
		go func() {
			_ = cb.Call(func() error { return nil })
			done <- struct{}{}
		}()
	}
	for range 50 {
		<-done
	}
}
