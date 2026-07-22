// SPDX-License-Identifier: AGPL-3.0-or-later

package vertex

import (
	"testing"

	llm "github.com/TaraTheStar/azoth/llm"
)

func TestVertexUsageFrom_TotalIsAuthoritative(t *testing.T) {
	// Gemini reports TotalTokenCount separately; we use it verbatim
	// rather than re-summing. CachedContentTokenCount is a sub-line
	// of PromptTokenCount, not additive.
	got := vertexUsageFrom(150, 75, 30, 225)
	want := llm.MessageUsage{
		InputTokens:     150, // prompt
		OutputTokens:    75,  // candidates
		CacheReadTokens: 30,  // cached (sub-line of prompt)
		TotalTokens:     225, // authoritative from Gemini
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestVertexUsageFrom_TotalIsNotRecomputed(t *testing.T) {
	// Verify TotalTokens uses the API-reported value even when it
	// disagrees with our local sum — Gemini's number is authoritative
	// and includes things we don't model (e.g. tool-use prompt tokens).
	got := vertexUsageFrom(100, 50, 0, 999)
	if got.TotalTokens != 999 {
		t.Errorf("TotalTokens = %d, want 999 (API-reported, not recomputed)",
			got.TotalTokens)
	}
}
