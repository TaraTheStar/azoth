// SPDX-License-Identifier: AGPL-3.0-or-later

// Package llm is a streaming chat client for OpenAI-compatible endpoints,
// built for the failure modes that appear when the server is a local model
// rather than a hosted API.
//
// The wire format is ordinary OpenAI chat completions, so it speaks to
// llama.cpp, vLLM, litellm, Ollama, or OpenAI itself. What it adds is the
// handling around that format:
//
//   - A truncated SSE stream is an error, never a silent clean finish.
//   - Streamed tool-call fragments are reassembled in index order with
//     deterministic synthesized IDs, so a retried turn produces byte-identical
//     history and a server's prompt-prefix cache stays warm.
//   - Tool calls that a GGUF chat template leaks into assistant text or the
//     reasoning channel are recovered rather than shown to the user.
//   - A stall watchdog fires on inter-token silence rather than slowness, so
//     one timeout works across a fast GPU and a slow large-context box.
//   - A repetition guard aborts a stream that collapses into a short repeating
//     cycle, before the max_tokens cap is reached.
//   - Transport failures retry with backoff and are categorized into messages
//     a user can act on; retryable HTTP statuses (429, 502, 503, 504) honor
//     Retry-After behind an opt-in counter that stays off for interactive apps.
//
// # Consuming a stream
//
// Chat returns a channel; [Stream] adapts any [ChatClient] into a
// range-over-func iterator over the same events. Both consume one request:
//
//	events, err := client.Chat(ctx, req)      // channel
//	for ev, err := range llm.Stream(ctx, client, req) { ... }   // iterator
//
// For a single ask-and-answer with no tools, [Complete] collects the text and
// returns it with the turn's usage. Agent loops should consume Chat or Stream
// directly, since Complete rejects tool calls rather than dropping them.
//
// # Sharing an endpoint
//
// [Pool] bounds concurrency with a FIFO queue, and [PooledClient] wraps any
// ChatClient with one. Use it when several independent callers — an agent
// loop, scheduled jobs, background digests — share inference hardware that
// can only serve so many requests at once.
//
// # Other providers
//
// Non-OpenAI backends live in subpackages that satisfy the same [ChatClient]
// interface: llm/anthropic, llm/bedrock, and llm/vertex. They are separate
// packages so their heavy vendor SDKs stay out of the dependency graph of
// anything importing only llm — go list -deps ./llm pulls none of them.
//
// # Testing
//
// llm/llmtest provides a scriptable ChatClient for driving agent loops in
// tests without standing up a server.
package llm
