// SPDX-License-Identifier: AGPL-3.0-or-later

package netsec

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Dialer is the resolve-and-pin dial path that turns the IsDeniedIP
// classification into an enforcing DialContext. It resolves the target host,
// refuses the connection if ANY resolved address is denied, and then dials the
// vetted IP literals directly so the stack never re-resolves between the check
// and the connect — the DNS-rebind defence. Sharing this composition means an
// SSRF hole found in one egress path (a fetch client, a host-side proxy) is
// closed for every application at once, not fixed three times.
//
// The "refuse if ANY resolved IP is denied" stance (rather than "dial the first
// allowed IP") is deliberate: a name that resolves to a mix of public and
// loopback addresses must not be dialable at all, or an attacker who controls
// the name's DNS could smuggle a private target past a per-address filter.
//
// The zero value is a usable, no-exemptions dialer: every target is filtered,
// each TCP dial is bounded at 10s, resolution uses net.DefaultResolver, and the
// class predicate is IsDeniedIP.
type Dialer struct {
	// Timeout bounds each individual TCP dial (not host resolution, which
	// honours the context). Zero selects 10s.
	Timeout time.Duration

	// Exempt, when non-nil, is consulted with the ORIGINAL dial target
	// ("host:port") before the denylist runs; returning true skips the
	// denylist for that one target. This is the operator-configured opt-out
	// for a deliberately-named loopback/LAN service (e.g. a local model
	// server on 127.0.0.1) — the operator typed the name, so it is trusted to
	// resolve into an otherwise-denied class. A nil Exempt (the zero value)
	// never exempts: every target is filtered, which is the right posture for
	// a client dialing worker- or model-chosen names.
	Exempt func(hostport string) bool

	// Resolver overrides host resolution. Zero uses net.DefaultResolver.
	// Primarily a test seam.
	Resolver *net.Resolver

	// Deny overrides the address-class predicate. Zero uses IsDeniedIP.
	// Primarily a test seam (e.g. permit loopback to point a target at an
	// httptest server while keeping the other classes denied).
	Deny func(net.IP) bool
}

func (d *Dialer) resolver() *net.Resolver {
	if d.Resolver != nil {
		return d.Resolver
	}
	return net.DefaultResolver
}

func (d *Dialer) deny() func(net.IP) bool {
	if d.Deny != nil {
		return d.Deny
	}
	return IsDeniedIP
}

func (d *Dialer) timeout() time.Duration {
	if d.Timeout > 0 {
		return d.Timeout
	}
	return 10 * time.Second
}

// DeniedError reports that a resolved address was refused by the denylist. It
// is returned by Dialer.DialContext so a caller can errors.As for it and
// replace the message with application-specific remediation — e.g. naming the
// allowlist entry that would make the target legal — while the default text
// stays actionable on its own. Host is the name that was dialed; IP is the
// specific resolved address that tripped the check.
type DeniedError struct {
	Host string
	IP   net.IP
}

func (e *DeniedError) Error() string {
	return fmt.Sprintf("netsec: refusing %s: resolves to denied address %s", e.Host, e.IP)
}

// DialContext resolves addr's host, refuses the connection if any resolved IP
// is a denied class (unless the target is Exempt), then dials the surviving IP
// literals in order until one connects. It satisfies the DialContext shape used
// by http.Transport and any code that dials by "host:port".
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("netsec: bad target %q: %w", addr, err)
	}
	ips, err := d.resolver().LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("netsec: resolve %s: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("netsec: resolve %s: no addresses", host)
	}
	if d.Exempt == nil || !d.Exempt(addr) {
		deny := d.deny()
		for _, ip := range ips {
			if deny(ip) {
				return nil, &DeniedError{Host: host, IP: ip}
			}
		}
	}
	dialer := net.Dialer{Timeout: d.timeout()}
	var lastErr error
	for _, ip := range ips {
		conn, derr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if derr == nil {
			return conn, nil
		}
		lastErr = derr
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("netsec: no addresses to dial for %s", host)
	}
	return nil, lastErr
}

// GuardedClient returns an *http.Client that resolves, denylist-checks, and
// pins every host it dials through a zero-value Dialer (no exemptions — every
// target is filtered), and caps redirects at 10 so each hop is re-checked
// rather than followed unboundedly. timeout is the whole-request timeout
// (net/http's Client.Timeout); a zero timeout means no whole-request bound,
// exactly as net/http defines it. This is the batteries-included client for a
// model-driven fetch tool; callers needing an exemption hook or a custom
// transport should compose a Dialer directly.
func GuardedClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: (&Dialer{}).DialContext},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
}
