// SPDX-License-Identifier: AGPL-3.0-or-later

package netsec

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// listenLoopback starts a throwaway TCP listener on loopback and returns its
// address; the listener accepts and immediately closes, which is enough to
// prove a dial connected.
func listenLoopback(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	return ln.Addr().String()
}

func TestDialerRefusesDeniedTarget(t *testing.T) {
	addr := listenLoopback(t)
	// Default Deny is IsDeniedIP, which denies loopback — the literal host
	// resolves to itself, so the dial must be refused before connecting.
	d := &Dialer{}
	_, err := d.DialContext(context.Background(), "tcp", addr)
	if err == nil {
		t.Fatal("expected refusal dialing loopback, got nil")
	}
	if !strings.Contains(err.Error(), "denied address") {
		t.Fatalf("error = %v, want it to mention a denied address", err)
	}
}

func TestDialerAllowsWhenNothingDenied(t *testing.T) {
	addr := listenLoopback(t)
	// Override the class predicate so loopback is treated as allowed; the
	// dial should then reach the listener.
	d := &Dialer{Deny: func(net.IP) bool { return false }}
	conn, err := d.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = conn.Close()
}

func TestDialerExemptBypassesDenylist(t *testing.T) {
	addr := listenLoopback(t)
	// Default Deny denies loopback, but an Exempt matching this target skips
	// the check — the operator-configured opt-out. Dial should connect.
	d := &Dialer{Exempt: func(hostport string) bool { return hostport == addr }}
	conn, err := d.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		t.Fatalf("dial with exemption: %v", err)
	}
	_ = conn.Close()

	// A non-matching exemption leaves the denylist in force.
	d2 := &Dialer{Exempt: func(hostport string) bool { return false }}
	if _, err := d2.DialContext(context.Background(), "tcp", addr); err == nil {
		t.Fatal("expected refusal when exemption does not match")
	}
}

func TestDialerBadTarget(t *testing.T) {
	d := &Dialer{}
	if _, err := d.DialContext(context.Background(), "tcp", "no-port-here"); err == nil {
		t.Fatal("expected error for target without a port")
	}
}

func TestGuardedClientRefusesLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// httptest listens on loopback; the guarded client must refuse it,
	// proving the denylist dialer is actually wired into the transport.
	_, err := GuardedClient(0).Get(srv.URL)
	if err == nil {
		t.Fatal("GuardedClient reached a loopback server; SSRF guard not wired")
	}
	if !strings.Contains(err.Error(), "denied address") {
		t.Fatalf("error = %v, want a denied-address refusal", err)
	}
}
