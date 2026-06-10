// Package runnotify provides in-process completion signals for async chat/agent runs.
package runnotify

import (
	"strings"
	"sync"
	"time"
)

// Result is the terminal outcome of a run.
type Result struct {
	Status string
	Error  string
}

type waiter struct {
	ch chan Result
}

var waiters sync.Map

// Register prepares a completion waiter for runID. Idempotent if already registered.
func Register(runID string) {
	runID = normalizeID(runID)
	if runID == "" {
		return
	}
	if _, loaded := waiters.LoadOrStore(runID, &waiter{ch: make(chan Result, 1)}); loaded {
		return
	}
}

// Complete signals a registered run as finished.
func Complete(runID, status, errMsg string) {
	runID = normalizeID(runID)
	if runID == "" {
		return
	}
	v, ok := waiters.Load(runID)
	if !ok {
		return
	}
	w := v.(*waiter)
	select {
	case w.ch <- Result{Status: status, Error: errMsg}:
	default:
	}
	waiters.Delete(runID)
}

// Wait blocks until Complete is called or timeout elapses.
func Wait(runID string, timeout time.Duration) (Result, bool) {
	runID = normalizeID(runID)
	if runID == "" {
		return Result{Status: "failed", Error: "empty run id"}, false
	}
	v, ok := waiters.Load(runID)
	if !ok {
		v = &waiter{ch: make(chan Result, 1)}
		waiters.Store(runID, v)
	}
	w := v.(*waiter)
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	select {
	case res := <-w.ch:
		waiters.Delete(runID)
		return res, true
	case <-time.After(timeout):
		waiters.Delete(runID)
		return Result{Status: "timeout", Error: "run completion timeout"}, false
	}
}

func normalizeID(id string) string {
	return strings.TrimSpace(id)
}
