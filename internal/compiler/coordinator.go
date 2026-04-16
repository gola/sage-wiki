package compiler

import (
	"context"
	"errors"
	"sync/atomic"
)

// ErrCompileTimeout is returned when CompileOrWait exceeds the context deadline.
var ErrCompileTimeout = errors.New("compile: timed out waiting for lock")

// CompileCoordinator serializes compilation across watch mode, CLI, and
// on-demand MCP requests. Created once at startup, injected into both
// the watcher and the MCP server.
//
// Uses a buffered channel as a semaphore instead of sync.Mutex so that
// CompileOrWait can select on ctx.Done() without spawning cleanup goroutines.
type CompileCoordinator struct {
	sem    chan struct{} // capacity 1 — acts as a mutex
	active atomic.Bool
}

// NewCompileCoordinator creates a new coordinator.
func NewCompileCoordinator() *CompileCoordinator {
	cc := &CompileCoordinator{
		sem: make(chan struct{}, 1),
	}
	// Pre-fill the token so the first Acquire succeeds.
	cc.sem <- struct{}{}
	return cc
}

// TryCompile attempts to acquire the compile lock and run fn.
// Returns (true, nil) on success, (false, nil) if another compile is running.
// Returns (true, err) if fn returned an error.
func (cc *CompileCoordinator) TryCompile(fn func() error) (bool, error) {
	select {
	case <-cc.sem:
		// Acquired
	default:
		return false, nil
	}
	cc.active.Store(true)
	defer func() {
		cc.active.Store(false)
		cc.sem <- struct{}{} // release
	}()
	return true, fn()
}

// CompileOrWait acquires the compile lock, blocking until available or
// context is cancelled. Used by on-demand compilation which can afford
// a short wait. Returns ErrCompileTimeout if context expires.
//
// No goroutines are leaked on timeout — select cleanly chooses between
// the semaphore and the context cancellation.
func (cc *CompileCoordinator) CompileOrWait(ctx context.Context, fn func() error) error {
	select {
	case <-cc.sem:
		// Got the lock
		cc.active.Store(true)
		defer func() {
			cc.active.Store(false)
			cc.sem <- struct{}{} // release
		}()
		return fn()
	case <-ctx.Done():
		return ErrCompileTimeout
	}
}

// IsActive returns whether a compile is currently running.
func (cc *CompileCoordinator) IsActive() bool {
	return cc.active.Load()
}
