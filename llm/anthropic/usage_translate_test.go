// SPDX-License-Identifier: AGPL-3.0-or-later

package anthropic

import (
	"testing"

	llm "github.com/TaraTheStar/azoth/llm"
)

func TestAnthropicUsageFrom_SummingTotal(t *testing.T) {
	// Anthropic InputTokens is fresh-only; cache reads/writes are
	// separate. Total = sum of all four.
	got := anthropicUsageFrom(100, 50, 20, 5)
	want := llm.MessageUsage{
		InputTokens:      100,
		OutputTokens:     50,
		CacheReadTokens:  20,
		CacheWriteTokens: 5,
		TotalTokens:      175,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAnthropicUsageFrom_ZeroIsEmpty(t *testing.T) {
	got := anthropicUsageFrom(0, 0, 0, 0)
	if !got.Empty() {
		t.Errorf("all-zero translation should be Empty, got %+v", got)
	}
}
