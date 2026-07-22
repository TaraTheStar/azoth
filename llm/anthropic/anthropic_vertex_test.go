// SPDX-License-Identifier: AGPL-3.0-or-later

package anthropic

import (
	"context"
	"strings"
	"testing"

	llm "github.com/TaraTheStar/azoth/llm"
)

func TestAnthropicVertexClient_MissingRegionErrors(t *testing.T) {
	c := &AnthropicVertexClient{Model: "claude-3-5-sonnet-v2@20241022", Project: "p"}
	_, err := c.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("want error for missing region")
	}
	if !strings.Contains(err.Error(), "region") {
		t.Fatalf("error should name region: %v", err)
	}
}

// TestAnthropicVertexClient_MissingProjectErrors mirrors the region
// check — project is also required, also pre-validated.
func TestAnthropicVertexClient_MissingProjectErrors(t *testing.T) {
	c := &AnthropicVertexClient{Model: "claude-3-5-sonnet-v2@20241022", Region: "us-east5"}
	_, err := c.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("want error for missing project")
	}
	if !strings.Contains(err.Error(), "project") {
		t.Fatalf("error should name project: %v", err)
	}
}

// TestProviderFactory_AnthropicVertexPromptCaching pins the factory
// wiring on the anthropic-vertex adapter — same TOML key.
