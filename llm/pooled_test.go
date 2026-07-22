// SPDX-License-Identifier: AGPL-3.0-or-later

package llm

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// gatedClient tracks concurrent in-flight Chat streams; each stream
// stays open until its gate channel is closed.
type gatedClient struct {
	mu       sync.Mutex
	inflight int32
	peak     int32
	gates    []chan struct{}
}

func (g *gatedClient) Chat(ctx context.Context, req ChatRequest) (<-chan Event, error) {
	gate := make(chan struct{})
	g.mu.Lock()
	g.gates = append(g.gates, gate)
	g.mu.Unlock()

	n := atomic.AddInt32(&g.inflight, 1)
	for {
		p := atomic.LoadInt32(&g.peak)
		if n <= p || atomic.CompareAndSwapInt32(&g.peak, p, n) {
			break
		}
	}

	ch := make(chan Event, 2)
	go func() {
		defer close(ch)
		defer atomic.AddInt32(&g.inflight, -1)
		<-gate
		ch <- Event{Type: EventDone, FinishReason: "stop"}
	}()
	return ch, nil
}

func (g *gatedClient) releaseAll() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, gate := range g.gates {
		close(gate)
	}
	g.gates = nil
}

func TestPooledClientBoundsConcurrency(t *testing.T) {
	inner := &gatedClient{}
	pooled := &PooledClient{Client: inner, Pool: NewPool(1)}

	const callers = 5
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, err := pooled.Chat(context.Background(), ChatRequest{})
			if err != nil {
				t.Errorf("Chat: %v", err)
				return
			}
			for range ch {
			}
		}()
	}

	// Let callers queue up, then drain them one gate at a time. The
	// pool must never admit a second stream while one is open.
	for i := 0; i < callers; i++ {
		deadline := time.After(2 * time.Second)
		for {
			inner.mu.Lock()
			ready := len(inner.gates) > 0
			inner.mu.Unlock()
			if ready {
				break
			}
			select {
			case <-deadline:
				t.Fatal("timed out waiting for a pooled stream to start")
			case <-time.After(time.Millisecond):
			}
		}
		inner.releaseAll()
	}
	wg.Wait()

	if peak := atomic.LoadInt32(&inner.peak); peak != 1 {
		t.Fatalf("peak concurrent streams = %d, want 1", peak)
	}
}

func TestPooledClientReleasesOnChatError(t *testing.T) {
	pooled := &PooledClient{Client: &chanClient{err: context.DeadlineExceeded}, Pool: NewPool(1)}
	for i := 0; i < 3; i++ {
		if _, err := pooled.Chat(context.Background(), ChatRequest{}); err == nil {
			t.Fatal("want error from inner client")
		}
	}
	if pooled.Pool.Inflight() != 0 {
		t.Fatalf("slot leaked: inflight = %d", pooled.Pool.Inflight())
	}
}
