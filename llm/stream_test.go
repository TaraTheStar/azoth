// SPDX-License-Identifier: AGPL-3.0-or-later

package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

// chanClient is a minimal ChatClient backed by a pre-filled channel.
type chanClient struct {
	ch  chan Event
	err error
}

func (c *chanClient) Chat(ctx context.Context, req ChatRequest) (<-chan Event, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.ch, nil
}

func TestStreamYieldsEventsInOrder(t *testing.T) {
	ch := make(chan Event, 4)
	ch <- Event{Type: EventTextDelta, Text: "hel"}
	ch <- Event{Type: EventTextDelta, Text: "lo"}
	ch <- Event{Type: EventDone, FinishReason: "stop"}
	close(ch)

	var text, finish string
	for ev, err := range Stream(context.Background(), &chanClient{ch: ch}, ChatRequest{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		switch ev.Type {
		case EventTextDelta:
			text += ev.Text
		case EventDone:
			finish = ev.FinishReason
		}
	}
	if text != "hello" || finish != "stop" {
		t.Fatalf("got text %q finish %q", text, finish)
	}
}

func TestStreamSurfacesChatError(t *testing.T) {
	boom := errors.New("boom")
	seen := 0
	for _, err := range Stream(context.Background(), &chanClient{err: boom}, ChatRequest{}) {
		seen++
		if !errors.Is(err, boom) {
			t.Fatalf("want boom, got %v", err)
		}
	}
	if seen != 1 {
		t.Fatalf("want exactly one yield, got %d", seen)
	}
}

func TestStreamSurfacesEventError(t *testing.T) {
	ch := make(chan Event, 2)
	boom := errors.New("mid-stream")
	ch <- Event{Type: EventError, Error: boom}
	ch <- Event{Type: EventDone}
	close(ch)

	var got error
	var done bool
	for ev, err := range Stream(context.Background(), &chanClient{ch: ch}, ChatRequest{}) {
		if err != nil {
			got = err
			continue
		}
		if ev.Type == EventDone {
			done = true
		}
	}
	if !errors.Is(got, boom) || !done {
		t.Fatalf("got err %v done %v", got, done)
	}
}

func TestStreamEarlyBreakDrainsProducer(t *testing.T) {
	ch := make(chan Event) // unbuffered: producer blocks until someone reads
	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		defer close(ch)
		for i := 0; i < 100; i++ {
			ch <- Event{Type: EventTextDelta, Text: "x"}
		}
	}()

	for ev, err := range Stream(context.Background(), &chanClient{ch: ch}, ChatRequest{}) {
		_ = err
		if ev.Type == EventTextDelta {
			break // bail after the first delta
		}
	}

	select {
	case <-producerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("producer goroutine leaked after early break")
	}
}
