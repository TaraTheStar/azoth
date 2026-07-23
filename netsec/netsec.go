// SPDX-License-Identifier: AGPL-3.0-or-later

// Package netsec holds the SSRF address denylist shared by every place a
// model-supplied name is turned into an outbound TCP connection. Centralising
// the address-class classification means a newly recognised dangerous range is
// closed on every egress path at once — a resolve-and-pin fetch client, a
// host-side egress proxy, a sealed worker pointed at an HTTPS_PROXY — rather
// than drifting between per-application copies.
//
// IsDeniedIP classifies a single address. Dialer composes that classification
// into the enforcing resolve-and-pin dial path (refuse if any resolved IP is
// denied, then dial the vetted literal so DNS can't rebind between check and
// connect), and GuardedClient wraps a zero-value Dialer in a ready *http.Client
// for a model-driven fetch tool. Application-specific policy that sits ABOVE
// the address check — per-host allowlists, interactive egress prompts, proxy
// framing — stays in the application; a Dialer.Exempt hook is the one seam
// provided for an operator-configured opt-out.
package netsec

import "net"

// IsDeniedIP reports whether an address is off-limits for a model-driven
// outbound connection. It fails closed: a nil IP is denied. Covered classes:
//
//   - loopback           127.0.0.0/8, ::1
//   - RFC1918 + RFC4193  10/8, 172.16/12, 192.168/16, fc00::/7   (net.IP.IsPrivate)
//   - link-local unicast 169.254.0.0/16 (incl. cloud metadata 169.254.169.254), fe80::/10
//   - multicast          224.0.0.0/4, ff00::/8
//   - unspecified        0.0.0.0, ::
//   - CGNAT              100.64.0.0/10
//   - "this network"     0.0.0.0/8   (IsUnspecified only catches the single 0.0.0.0)
//   - broadcast          255.255.255.255
//
// IsPrivate already covers the IPv6 unique-local range (fc00::/7), and
// IsMulticast subsumes link-local multicast, so neither needs a separate check.
func IsDeniedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		switch {
		case v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127: // CGNAT 100.64.0.0/10
			return true
		case v4[0] == 0: // 0.0.0.0/8 "this network"
			return true
		case v4.Equal(net.IPv4bcast): // 255.255.255.255
			return true
		}
	}
	return false
}
