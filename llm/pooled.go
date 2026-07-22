// SPDX-License-Identifier: AGPL-3.0-or-later

package llm

import "context"

// PooledClient bounds a ChatClient with a Pool: each Chat call holds
// one slot from the moment the request is admitted until its event
// stream closes. Use it when several independent callers (an agent
// loop, scheduled jobs, background digests) share one inference
// endpoint and must not race for its slots:
//
//	chat := &llm.PooledClient{Client: openai, Pool: llm.NewPool(1)}
//
// Acquire failures (context cancelled, ErrQueueTimeout) surface as the
// Chat error. The wrapper forwards events unmodified.
type PooledClient struct {
	Client ChatClient
	Pool   *Pool
}

// Chat implements ChatClient.
func (p *PooledClient) Chat(ctx context.Context, req ChatRequest) (<-chan Event, error) {
	release, err := p.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	ch, err := p.Client.Chat(ctx, req)
	if err != nil {
		release()
		return nil, err
	}
	out := make(chan Event, 32)
	go func() {
		defer close(out)
		defer release()
		for ev := range ch {
			out <- ev
		}
	}()
	return out, nil
}

// compile-time assertion
var _ ChatClient = (*PooledClient)(nil)
