package compiler

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCompileCoordinator_TryCompile(t *testing.T) {
	cc := NewCompileCoordinator()

	ran := false
	ok, err := cc.TryCompile(func() error {
		ran = true
		return nil
	})
	if !ok || err != nil {
		t.Fatalf("TryCompile: ok=%v err=%v", ok, err)
	}
	if !ran {
		t.Error("function should have run")
	}
}

func TestCompileCoordinator_TryCompile_Concurrent(t *testing.T) {
	cc := NewCompileCoordinator()

	started := make(chan struct{})
	done := make(chan struct{})

	// First compile — holds the lock
	go func() {
		cc.TryCompile(func() error {
			close(started)
			<-done // block until released
			return nil
		})
	}()

	<-started // wait for first compile to start

	// Second compile should fail immediately
	ok, _ := cc.TryCompile(func() error {
		t.Error("second compile should not run")
		return nil
	})
	if ok {
		t.Error("TryCompile should return false when lock is held")
	}

	if !cc.IsActive() {
		t.Error("IsActive should be true while compile is running")
	}

	close(done) // release first compile
	time.Sleep(10 * time.Millisecond)

	if cc.IsActive() {
		t.Error("IsActive should be false after compile completes")
	}
}

func TestCompileCoordinator_CompileOrWait(t *testing.T) {
	cc := NewCompileCoordinator()

	ctx := context.Background()
	ran := false
	err := cc.CompileOrWait(ctx, func() error {
		ran = true
		return nil
	})
	if err != nil {
		t.Fatalf("CompileOrWait: %v", err)
	}
	if !ran {
		t.Error("function should have run")
	}
}

func TestCompileCoordinator_CompileOrWait_Timeout(t *testing.T) {
	cc := NewCompileCoordinator()

	// Hold the lock
	var wg sync.WaitGroup
	wg.Add(1)
	release := make(chan struct{})
	go func() {
		cc.TryCompile(func() error {
			wg.Done()
			<-release
			return nil
		})
	}()
	wg.Wait() // wait for lock to be held

	// Try with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := cc.CompileOrWait(ctx, func() error {
		t.Error("should not run — timed out")
		return nil
	})
	if err != ErrCompileTimeout {
		t.Errorf("expected ErrCompileTimeout, got %v", err)
	}

	close(release)
}

func TestCompileCoordinator_CompileOrWait_WaitsForLock(t *testing.T) {
	cc := NewCompileCoordinator()

	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	// Hold lock briefly
	go func() {
		cc.TryCompile(func() error {
			wg.Done()
			<-release
			return nil
		})
	}()
	wg.Wait()

	// Release after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(release)
	}()

	// Should wait and succeed
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ran := false
	err := cc.CompileOrWait(ctx, func() error {
		ran = true
		return nil
	})
	if err != nil {
		t.Fatalf("CompileOrWait: %v", err)
	}
	if !ran {
		t.Error("function should have run after lock was released")
	}
}
