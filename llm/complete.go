// SPDX-License-Identifier: AGPL-3.0-or-later

package llm

import (
	"context"
	"fmt"
	"strings"
)

// Complete runs one ask-and-answer completion: it streams internally,
// collects the assistant text, and returns it with the turn's usage
// (zero-valued when the provider reported none). Reasoning deltas are
// discarded.
//
// It is a convenience for the digest/summary/describe pattern every
// consumer otherwise hand-rolls. It is NOT for tool-enabled requests —
// a tool call arriving mid-stream is returned as an error, since
// silently dropping the model's chosen action would corrupt the
// exchange. Agent loops should consume Chat or Stream directly.
func Complete(ctx context.Context, c ChatClient, req ChatRequest) (string, MessageUsage, error) {
	var text strings.Builder
	var usage MessageUsage
	for ev, err := range Stream(ctx, c, req) {
		if err != nil {
			return "", MessageUsage{}, err
		}
		switch ev.Type {
		case EventTextDelta:
			text.WriteString(ev.Text)
		case EventToolCallComplete:
			return "", MessageUsage{}, fmt.Errorf("llm: Complete got a tool call (%s); use Chat or Stream for tool-enabled requests", ev.ToolCalls[0].Function.Name)
		case EventUsage:
			usage = ev.Usage
		}
	}
	return text.String(), usage, nil
}
