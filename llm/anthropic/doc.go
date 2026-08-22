// SPDX-License-Identifier: AGPL-3.0-or-later

// Package anthropic speaks the Anthropic Messages API behind the
// [github.com/TaraTheStar/azoth/llm.ChatClient] interface, so a host swaps it
// for the OpenAI client without touching its agent loop.
//
// Three clients share one wire protocol and differ only in how they are
// routed and authenticated:
//
//   - [Client] calls api.anthropic.com directly with an API key.
//   - [BedrockClient] routes the same protocol through Amazon Bedrock,
//     authenticating with the AWS credential chain.
//   - [VertexClient] routes it through GCP Vertex AI, authenticating with
//     Google application default credentials.
//
// Each is a plain struct with exported fields — no constructors — so a host
// builds one by literal and hands it around as a ChatClient:
//
//	c := &anthropic.Client{APIKey: key, Model: "claude-sonnet-4-5", MaxTokens: 16000}
//	events, err := c.Chat(ctx, req)
//
// Prompt caching, extended thinking, and their per-provider constraints are
// carried through: enabling ExtendedThinking applies the temperature the API
// requires and clamps the thinking budget, so callers set one field rather
// than three.
//
// This is a separate package from llm on purpose — importing it pulls
// anthropic-sdk-go, which anything using only the OpenAI path should not pay
// for.
package anthropic
