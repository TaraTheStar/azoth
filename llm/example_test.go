// SPDX-License-Identifier: AGPL-3.0-or-later

package llm_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/TaraTheStar/azoth/llm"
)

// Streaming a turn from a local model, with the guards that matter when the
// server is llama.cpp rather than a hosted API.
func ExampleOpenAIClient_Chat() {
	client := &llm.OpenAIClient{
		Endpoint: "http://localhost:8080/v1",
		Model:    "qwen3-coder-30b",

		// Fires on inter-token silence, not on slowness, so the same
		// value holds across a fast GPU and a slow large-context box.
		StallTimeout: 90 * time.Second,

		// Aborts a stream that collapses into a repeating cycle rather
		// than waiting for the max_tokens cap.
		LoopGuard: true,
	}

	events, err := client.Chat(context.Background(), llm.ChatRequest{
		Model:     client.Model,
		Messages:  []llm.Message{{Role: "user", Content: "Name three primes."}},
		Stream:    true,
		MaxTokens: 512,
	})
	if err != nil {
		log.Fatal(err)
	}

	for ev := range events {
		switch ev.Type {
		case llm.EventTextDelta:
			fmt.Print(ev.Text)
		case llm.EventDone:
			// "stop", "length", "tool_calls", or the synthetic
			// "stall" / "repetition" this client raises itself.
			fmt.Println("\nfinished:", ev.FinishReason)
		case llm.EventError:
			log.Fatal(ev.Error)
		}
	}
}

// Letting the model call a tool: define it, stream until the call arrives,
// run it, then feed the result back as a "tool" message on the next turn.
func ExampleOpenAIClient_Chat_tools() {
	client := &llm.OpenAIClient{
		Endpoint: "http://localhost:8080/v1",
		Model:    "qwen3-coder-30b",
	}

	msgs := []llm.Message{{Role: "user", Content: "What is in README.md?"}}
	tools := []llm.ToolDef{{
		Type: "function",
		Function: llm.ToolFunctionDef{
			Name:        "read_file",
			Description: "Read a UTF-8 text file from disk.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
				"required": []string{"path"},
			},
		},
	}}

	events, err := client.Chat(context.Background(), llm.ChatRequest{
		Model:    client.Model,
		Messages: msgs,
		Tools:    tools,
		Stream:   true,
	})
	if err != nil {
		log.Fatal(err)
	}

	for ev := range events {
		if ev.Type != llm.EventToolCallComplete {
			continue
		}
		for _, call := range ev.ToolCalls {
			// call.Function.Arguments is raw JSON, exactly as the
			// model emitted it; unmarshal into the tool's own type.
			result := runTool(call.Function.Name, call.Function.Arguments)

			// The ID ties the result back to the call. It is stable
			// across a retried turn, which keeps the server's
			// prompt-prefix cache warm.
			msgs = append(msgs,
				llm.Message{Role: "assistant", ToolCalls: []llm.ToolCall{call}},
				llm.Message{Role: "tool", ToolCallID: call.ID, Content: result},
			)
		}
	}
	// ...then send msgs back for the next turn.
}

// Stream adapts any ChatClient into a range-over-func iterator. Breaking out
// early cancels the request and drains the producer, so nothing leaks.
func ExampleStream() {
	var client llm.ChatClient = &llm.OpenAIClient{
		Endpoint: "http://localhost:8080/v1",
		Model:    "qwen3-coder-30b",
	}

	req := llm.ChatRequest{
		Model:    "qwen3-coder-30b",
		Messages: []llm.Message{{Role: "user", Content: "Summarize this changelog."}},
		Stream:   true,
	}

	var written int
	for ev, err := range llm.Stream(context.Background(), client, req) {
		if err != nil {
			log.Fatal(err)
		}
		if ev.Type == llm.EventTextDelta {
			fmt.Print(ev.Text)
			if written += len(ev.Text); written > 4096 {
				break // safe: context cancelled, stream drained
			}
		}
	}
}

// Complete is the one-call ask-and-answer for digests, summaries, and
// describe prompts — the pattern every consumer otherwise hand-rolls.
func ExampleComplete() {
	client := &llm.OpenAIClient{
		Endpoint: "http://localhost:8080/v1",
		Model:    "qwen3-coder-30b",
	}

	text, usage, err := llm.Complete(context.Background(), client, llm.ChatRequest{
		Model: client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: "Reply with one sentence."},
			{Role: "user", Content: "What changed in this diff?"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(text)
	// TotalTokens is authoritative; summing the fields by hand is wrong
	// on providers where cached input is a sub-line of the input count.
	fmt.Println("tokens:", usage.TotalTokens)
}

// One inference endpoint, several independent callers. The pool admits them
// in FIFO order and holds each slot until that request's stream closes.
func ExamplePooledClient() {
	endpoint := &llm.OpenAIClient{
		Endpoint: "http://localhost:8080/v1",
		Model:    "qwen3-coder-30b",
	}

	// One slot: a single-GPU box that serves one request at a time.
	// Queued callers wait up to 30s before ErrQueueTimeout.
	shared := &llm.PooledClient{
		Client: endpoint,
		Pool:   llm.NewPoolNamed("local", 1, 30*time.Second),
	}

	// The agent loop and a background digest can now both hold this
	// client without racing for the endpoint's capacity.
	go backgroundDigest(shared)
	runAgentLoop(shared)
}

func runTool(name, args string) string  { return "" }
func backgroundDigest(c llm.ChatClient) {}
func runAgentLoop(c llm.ChatClient)     {}
