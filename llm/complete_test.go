// SPDX-License-Identifier: AGPL-3.0-or-later

package llm

import (
	"context"
	"strings"
	"testing"
)

func TestCompleteCollectsTextAndUsage(t *testing.T) {
	ch := make(chan Event, 4)
	ch <- Event{Type: EventTextDelta, Text: "morning "}
	ch <- Event{Type: EventTextDelta, Text: "tablet"}
	ch <- Event{Type: EventUsage, Usage: MessageUsage{InputTokens: 10, OutputTokens: 2, TotalTokens: 12}}
	ch <- Event{Type: EventDone, FinishReason: "stop"}
	close(ch)

	text, usage, err := Complete(context.Background(), &chanClient{ch: ch}, ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "morning tablet" {
		t.Fatalf("text = %q", text)
	}
	if usage.TotalTokens != 12 {
		t.Fatalf("usage = %+v", usage)
	}
}

func TestCompleteRejectsToolCalls(t *testing.T) {
	ch := make(chan Event, 2)
	ch <- Event{Type: EventToolCallComplete, ToolCalls: []ToolCall{
		{ID: "c1", Type: "function", Function: FunctionCall{Name: "surprise", Arguments: "{}"}},
	}}
	close(ch)

	_, _, err := Complete(context.Background(), &chanClient{ch: ch}, ChatRequest{})
	if err == nil || !strings.Contains(err.Error(), "surprise") {
		t.Fatalf("want tool-call rejection naming the tool, got %v", err)
	}
}

func TestCompleteSurfacesStreamError(t *testing.T) {
	ch := make(chan Event, 1)
	ch <- Event{Type: EventError, Error: context.DeadlineExceeded}
	close(ch)

	_, _, err := Complete(context.Background(), &chanClient{ch: ch}, ChatRequest{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}
