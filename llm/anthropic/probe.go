// SPDX-License-Identifier: AGPL-3.0-or-later

package anthropic

import "time"

// Probe timing for the Anthropic-family adapters' recovery loops. Each
// client type defines its own probe goroutine but shares these timings;
// azoth/llm keeps the equivalent constants unexported for its OpenAIClient,
// so these cloud adapters carry their own copy. Values match the
// OpenAIClient defaults: recovery-aware without being chatty.
const (
	// probeInterval is how often a disconnected adapter re-pings its
	// endpoint while waiting to recover.
	probeInterval = 5 * time.Second
	// probeTimeout caps each probe attempt so a stuck DNS/connect can't pile
	// up goroutines across cycles.
	probeTimeout = 3 * time.Second
)
