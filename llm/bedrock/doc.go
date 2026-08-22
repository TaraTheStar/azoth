// SPDX-License-Identifier: AGPL-3.0-or-later

// Package bedrock speaks Amazon Bedrock's multi-vendor Converse API behind the
// [github.com/TaraTheStar/azoth/llm.ChatClient] interface, so a host swaps it
// for the OpenAI client without touching its agent loop.
//
// Converse is the vendor-neutral surface: one [Client] reaches Claude, Nova,
// Llama, and the rest by model ID, authenticating with the AWS credential
// chain (Profile, or the ambient environment when empty).
//
//	c := &bedrock.Client{Model: "anthropic.claude-sonnet-4-5-v1:0", Region: "us-east-1"}
//	events, err := c.Chat(ctx, req)
//
// Bedrock Guardrails are configured by field, and extended thinking is applied
// through additionalModelRequestFields with its constraints handled for you.
// Both are model-specific: pairing a thinking budget with a model that has no
// thinking mode surfaces as a Bedrock 400, not a silent no-op.
//
// For Claude specifically, llm/anthropic's BedrockClient is the alternative —
// the Messages API routed through Bedrock rather than the Converse
// abstraction over it. Choose this package for multi-vendor reach, that one
// for full Messages-API fidelity.
//
// This is a separate package from llm on purpose — importing it pulls
// aws-sdk-go-v2, which anything using only the OpenAI path should not pay for.
package bedrock
