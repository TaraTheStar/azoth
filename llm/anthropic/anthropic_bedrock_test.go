// SPDX-License-Identifier: AGPL-3.0-or-later

package anthropic

import (
	"encoding/json"
	"strings"
	"testing"

	llm "github.com/TaraTheStar/azoth/llm"
)

func TestAnthropicBedrock_BuildParamsReusesAnthropicTranslator(t *testing.T) {
	params, err := buildAnthropicParams(
		llm.ChatRequest{
			Messages: []llm.Message{
				{Role: "system", Content: "be brief"},
				{Role: "user", Content: "hi"},
			},
		},
		"anthropic.claude-3-5-sonnet-20241022-v2:0",
		16000,
		true,  // extended thinking
		8000,  // budget
		false, // prompt caching
	)
	if err != nil {
		t.Fatalf("buildAnthropicParams: %v", err)
	}
	data, _ := json.Marshal(params)
	js := string(data)
	if !strings.Contains(js, `"system":[{"text":"be brief"`) {
		t.Fatalf("system not hoisted: %s", js)
	}
	if !strings.Contains(js, `"thinking"`) || !strings.Contains(js, `"budget_tokens":8000`) {
		t.Fatalf("thinking not applied: %s", js)
	}
	if !strings.Contains(js, `"model":"anthropic.claude-3-5-sonnet-20241022-v2:0"`) {
		t.Fatalf("bedrock model id not preserved: %s", js)
	}
}

// TestProviderFactory_AnthropicBedrockType checks that
// type = "anthropic-bedrock" constructs an AnthropicBedrockClient with
// AWS-specific config (region, profile) threaded. Distinct from
// type = "bedrock" (Converse) — both can coexist in one config.
