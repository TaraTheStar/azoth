// SPDX-License-Identifier: AGPL-3.0-or-later

package bedrock

import (
	"testing"

	llm "github.com/TaraTheStar/azoth/llm"
)

func TestBedrockUsageFrom_SummingTotal(t *testing.T) {
	// Bedrock Converse mirrors Anthropic accounting.
	got := bedrockUsageFrom(120, 60, 25, 7)
	want := llm.MessageUsage{
		InputTokens:      120,
		OutputTokens:     60,
		CacheReadTokens:  25,
		CacheWriteTokens: 7,
		TotalTokens:      212,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestBedrockUsageFrom_ZeroIsEmpty(t *testing.T) {
	got := bedrockUsageFrom(0, 0, 0, 0)
	if !got.Empty() {
		t.Errorf("all-zero translation should be Empty, got %+v", got)
	}
}
