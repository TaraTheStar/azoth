// SPDX-License-Identifier: AGPL-3.0-or-later

package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func sseOK() *http.Response {
	body := "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n" +
		"data: [DONE]\n\n"
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func statusResp(code int, retryAfter string) *http.Response {
	h := http.Header{}
	if retryAfter != "" {
		h.Set("Retry-After", retryAfter)
	}
	return &http.Response{
		StatusCode: code,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader("busy")),
	}
}

func collectText(t *testing.T, c *OpenAIClient) (string, error) {
	t.Helper()
	ch, err := c.Chat(context.Background(), ChatRequest{})
	if err != nil {
		return "", err
	}
	var text strings.Builder
	for ev := range ch {
		if ev.Type == EventTextDelta {
			text.WriteString(ev.Text)
		}
		if ev.Type == EventError {
			return "", ev.Error
		}
	}
	return text.String(), nil
}

func TestStatusRetryDisabledByDefault(t *testing.T) {
	var calls atomic.Int32
	c := &OpenAIClient{Endpoint: "http://x", HTTPClient: &http.Client{Transport: roundTripFunc(
		func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return statusResp(429, ""), nil
		})}}
	_, err := collectText(t, c)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 429 {
		t.Fatalf("want APIError 429, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("want 1 attempt with retries disabled, got %d", calls.Load())
	}
}

func TestStatusRetryRecoversAfter429(t *testing.T) {
	var calls atomic.Int32
	c := &OpenAIClient{
		Endpoint:      "http://x",
		StatusRetries: 2,
		RetryBackoff:  func(int) time.Duration { return time.Millisecond },
		HTTPClient: &http.Client{Transport: roundTripFunc(
			func(req *http.Request) (*http.Response, error) {
				if calls.Add(1) <= 2 {
					return statusResp(429, "0"), nil
				}
				return sseOK(), nil
			})},
	}
	text, err := collectText(t, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "ok" || calls.Load() != 3 {
		t.Fatalf("text %q after %d calls", text, calls.Load())
	}
}

func TestStatusRetryDoesNotRetryNonRetryable(t *testing.T) {
	var calls atomic.Int32
	c := &OpenAIClient{Endpoint: "http://x", StatusRetries: 3,
		HTTPClient: &http.Client{Transport: roundTripFunc(
			func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				return statusResp(500, ""), nil
			})}}
	_, err := collectText(t, c)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 500 {
		t.Fatalf("want APIError 500, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("500 must not retry; got %d attempts", calls.Load())
	}
}

func TestStatusRetryRespectsMaxWait(t *testing.T) {
	var calls atomic.Int32
	c := &OpenAIClient{Endpoint: "http://x", StatusRetries: 3,
		StatusRetryMaxWait: time.Second,
		HTTPClient: &http.Client{Transport: roundTripFunc(
			func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				return statusResp(429, "120"), nil
			})}}
	start := time.Now()
	_, err := collectText(t, c)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != 120*time.Second {
		t.Fatalf("want APIError carrying Retry-After, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("over-cap Retry-After must surface immediately; got %d attempts", calls.Load())
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("slept toward an over-cap Retry-After instead of surfacing")
	}
}

func TestStatusRetryHonorsContextCancel(t *testing.T) {
	c := &OpenAIClient{Endpoint: "http://x", StatusRetries: 5,
		HTTPClient: &http.Client{Transport: roundTripFunc(
			func(req *http.Request) (*http.Response, error) {
				return statusResp(503, "10"), nil
			})}}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Chat(ctx, ChatRequest{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want ctx deadline error, got %v", err)
	}
}
