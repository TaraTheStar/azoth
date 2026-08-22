// SPDX-License-Identifier: AGPL-3.0-or-later

// Package vertex speaks Google Vertex AI's generate-content API behind the
// [github.com/TaraTheStar/azoth/llm.ChatClient] interface, so a host swaps it
// for the OpenAI client without touching its agent loop.
//
// [Client] reaches Gemini models by ID within a project and location,
// authenticating with Google application default credentials:
//
//	c := &vertex.Client{Model: "gemini-2.5-pro", Project: "my-project", Location: "us-central1"}
//	events, err := c.Chat(ctx, req)
//
// Gemini's thinking output is surfaced as reasoning deltas — the same event
// stream an OpenAI reasoning model or Claude extended thinking arrives on — so
// a UI that already renders reasoning needs no per-provider branch. Unlike
// Anthropic's, Gemini's thinking budget has no floor or ceiling: zero leaves
// its dynamic mode in effect.
//
// This is a separate package from llm on purpose — importing it pulls
// google.golang.org/genai, which anything using only the OpenAI path should
// not pay for.
package vertex
