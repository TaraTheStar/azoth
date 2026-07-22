// SPDX-License-Identifier: AGPL-3.0-or-later

package llm

import (
	"sync"
	"time"
)

// ConnState is the LLM-provider connection state surfaced to the TUI
// status bar. It tracks the *transport* relationship to the configured
// provider endpoint — HTTP-status errors (401, 429, 5xx) leave the state
// alone since the TLS+TCP path was healthy enough to get a response.
type ConnState int

const (
	StateConnected ConnState = iota
	StateReconnecting
	StateDisconnected
)

func (s ConnState) String() string {
	switch s {
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateDisconnected:
		return "disconnected"
	default:
		return "unknown"
	}
}

// ConnStateReporter is the read side of a ChatClient's connection-state
// tracker. The TUI status bar takes a Provider.Client and type-asserts
// to this interface; test fakes that don't implement it simply render
// nothing (treated as healthy).
type ConnStateReporter interface {
	LLMConnState() ConnState
}

// ConnTracker holds a ChatClient's last-known transport state plus the
// lifecycle bookkeeping for the recovery probe goroutine. All fields are
// mutex-guarded so Chat() (writer) and the TUI refresh ticker (reader) can
// race safely.
//
// It is exported so out-of-package vendor adapters (an app's Bedrock/Vertex/
// Anthropic clients) can embed it and drive the same TUI connection indicator
// as OpenAIClient, which embeds it too. The zero value is ready to use
// (StateConnected). The fields stay unexported: embedders call the methods.
type ConnTracker struct {
	mu      sync.Mutex
	state   ConnState
	since   time.Time
	probing bool // true while a probe goroutine is running
}

// Get returns the current transport state.
func (t *ConnTracker) Get() ConnState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

// Set transitions the tracker to s and returns the previous state. The
// `since` timestamp only updates on an actual change so callers reading
// "how long have we been degraded?" see the original transition time.
// Adapters rely on the returned previous state to fire side effects (start a
// recovery probe, log) only on the edge into a new state.
func (t *ConnTracker) Set(s ConnState) ConnState {
	t.mu.Lock()
	defer t.mu.Unlock()
	prev := t.state
	if prev != s {
		t.state = s
		t.since = time.Now()
	}
	return prev
}

// ClaimProbe atomically marks a probe as running. Returns true if the caller
// is the one that should start the probe goroutine (idempotent across
// concurrent Disconnected transitions).
func (t *ConnTracker) ClaimProbe() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.probing {
		return false
	}
	t.probing = true
	return true
}

// ReleaseProbe clears the probing flag. Called from the probe goroutine
// before it exits.
func (t *ConnTracker) ReleaseProbe() {
	t.mu.Lock()
	t.probing = false
	t.mu.Unlock()
}
