// SPDX-License-Identifier: AGPL-3.0-or-later

package llm

import (
	"context"
	"iter"
)

// Stream adapts any ChatClient's channel-based Chat into a range-over-func
// iterator:
//
//	for ev, err := range llm.Stream(ctx, client, req) {
//	    if err != nil { ... }
//	    switch ev.Type { ... }
//	}
//
// EventError events and a failed Chat call both surface on the error side
// of the pair; every other event arrives with a nil error. EventDone is
// yielded like any other event so consumers can read FinishReason.
//
// Breaking out of the loop early is safe: the request context is
// cancelled and the producer goroutine drained in the background, so
// neither the goroutine nor the HTTP stream leaks.
func Stream(ctx context.Context, c ChatClient, req ChatRequest) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		ctx, cancel := context.WithCancel(ctx)
		ch, err := c.Chat(ctx, req)
		if err != nil {
			cancel()
			yield(Event{}, err)
			return
		}
		defer func() {
			// Cancel first so an in-flight stream tears down promptly,
			// then drain so the producer's pending sends can complete and
			// the goroutine exits. After a normal finish both are no-ops
			// (the channel is already closed and empty).
			cancel()
			go func() {
				for range ch {
				}
			}()
		}()
		for ev := range ch {
			if ev.Type == EventError {
				if !yield(Event{}, ev.Error) {
					return
				}
				continue
			}
			if !yield(ev, nil) {
				return
			}
		}
	}
}
