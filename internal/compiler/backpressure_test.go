package compiler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackpressureController_Basic(t *testing.T) {
	bc := NewBackpressureController(4)
	if bc.CurrentLimit() != 4 {
		t.Errorf("initial limit = %d, want 4", bc.CurrentLimit())
	}
	if bc.MaxParallel() != 4 {
		t.Errorf("max = %d, want 4", bc.MaxParallel())
	}
}

func TestBackpressureController_AcquireRelease(t *testing.T) {
	bc := NewBackpressureController(3)

	// Acquire 3 slots
	r1 := bc.Acquire()
	r2 := bc.Acquire()
	r3 := bc.Acquire()

	if bc.InFlight() != 3 {
		t.Errorf("in-flight = %d, want 3", bc.InFlight())
	}

	// Release one
	r1()
	if bc.InFlight() != 2 {
		t.Errorf("in-flight after release = %d, want 2", bc.InFlight())
	}

	// Can acquire again
	r4 := bc.Acquire()
	r2()
	r3()
	r4()
}

func TestBackpressureController_ConcurrentAcquire(t *testing.T) {
	bc := NewBackpressureController(5)
	var maxSeen atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release := bc.Acquire()
			defer release()

			current := bc.InFlight()
			for {
				old := maxSeen.Load()
				if int32(current) <= old {
					break
				}
				if maxSeen.CompareAndSwap(old, int32(current)) {
					break
				}
			}

			time.Sleep(5 * time.Millisecond)
			bc.OnSuccess()
		}()
	}

	wg.Wait()

	if maxSeen.Load() > 5 {
		t.Errorf("max concurrent = %d, exceeded limit of 5", maxSeen.Load())
	}
}

func TestBackpressureController_OnRateLimit(t *testing.T) {
	bc := NewBackpressureController(20)

	// First rate limit: 20 → 10
	delay := bc.OnRateLimit()
	if bc.CurrentLimit() != 10 {
		t.Errorf("after first 429: limit = %d, want 10", bc.CurrentLimit())
	}
	if delay <= 0 {
		t.Error("backoff delay should be positive")
	}

	// Second rate limit: 10 → 5
	bc.OnRateLimit()
	if bc.CurrentLimit() != 5 {
		t.Errorf("after second 429: limit = %d, want 5", bc.CurrentLimit())
	}

	// Third: 5 → 2
	bc.OnRateLimit()
	if bc.CurrentLimit() != 2 {
		t.Errorf("after third 429: limit = %d, want 2", bc.CurrentLimit())
	}

	// Fourth: 2 → 1
	bc.OnRateLimit()
	if bc.CurrentLimit() != 1 {
		t.Errorf("after fourth 429: limit = %d, want 1", bc.CurrentLimit())
	}

	// Fifth: stays at 1 (minimum)
	bc.OnRateLimit()
	if bc.CurrentLimit() != 1 {
		t.Errorf("after fifth 429: limit = %d, want 1 (minimum)", bc.CurrentLimit())
	}
}

func TestBackpressureController_Recovery(t *testing.T) {
	bc := NewBackpressureController(20)

	// Drop to 1
	bc.OnRateLimit() // 10
	bc.OnRateLimit() // 5
	bc.OnRateLimit() // 2
	bc.OnRateLimit() // 1

	if bc.CurrentLimit() != 1 {
		t.Fatalf("limit = %d, want 1", bc.CurrentLimit())
	}

	// Recover via multiplicative increase (double every 5 successes)
	// 1 → 2 → 4 → 8 → 16 → 20 (capped)
	for i := 0; i < 5; i++ {
		bc.OnSuccess()
	}
	if bc.CurrentLimit() != 2 {
		t.Errorf("after 5 successes: limit = %d, want 2", bc.CurrentLimit())
	}

	for i := 0; i < 5; i++ {
		bc.OnSuccess()
	}
	if bc.CurrentLimit() != 4 {
		t.Errorf("after 10 successes: limit = %d, want 4", bc.CurrentLimit())
	}

	for i := 0; i < 5; i++ {
		bc.OnSuccess()
	}
	if bc.CurrentLimit() != 8 {
		t.Errorf("after 15 successes: limit = %d, want 8", bc.CurrentLimit())
	}

	for i := 0; i < 5; i++ {
		bc.OnSuccess()
	}
	if bc.CurrentLimit() != 16 {
		t.Errorf("after 20 successes: limit = %d, want 16", bc.CurrentLimit())
	}

	for i := 0; i < 5; i++ {
		bc.OnSuccess()
	}
	if bc.CurrentLimit() != 20 {
		t.Errorf("after 25 successes: limit = %d, want 20 (capped)", bc.CurrentLimit())
	}

	// Already at max — should stay
	for i := 0; i < 10; i++ {
		bc.OnSuccess()
	}
	if bc.CurrentLimit() != 20 {
		t.Errorf("should stay at max 20, got %d", bc.CurrentLimit())
	}
}

func TestBackpressureController_RateLimitResetsRecovery(t *testing.T) {
	bc := NewBackpressureController(20)

	// Drop to 5
	bc.OnRateLimit() // 10
	bc.OnRateLimit() // 5

	// Start recovering
	for i := 0; i < 4; i++ {
		bc.OnSuccess()
	}
	// Not yet at 5 successes, so still at 5

	// Another rate limit — resets consecutive OK counter
	bc.OnRateLimit() // 2
	if bc.CurrentLimit() != 2 {
		t.Errorf("limit = %d, want 2", bc.CurrentLimit())
	}

	// 4 more successes shouldn't trigger recovery (counter was reset)
	for i := 0; i < 4; i++ {
		bc.OnSuccess()
	}
	if bc.CurrentLimit() != 2 {
		t.Errorf("limit should still be 2 (only 4 successes since reset), got %d", bc.CurrentLimit())
	}
}

func TestBackpressureController_DefaultMaxParallel(t *testing.T) {
	bc := NewBackpressureController(0)
	if bc.MaxParallel() != 20 {
		t.Errorf("default max = %d, want 20", bc.MaxParallel())
	}
}
