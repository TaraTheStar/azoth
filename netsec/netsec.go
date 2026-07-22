// SPDX-License-Identifier: AGPL-3.0-or-later

// Package netsec holds the SSRF address denylist shared by every place a
// model-supplied name is turned into an outbound TCP connection. Centralising
// the address-class classification means a newly recognised dangerous range is
// closed on every egress path at once — a resolve-and-pin fetch client, a
// host-side egress proxy, a sealed worker pointed at an HTTPS_PROXY — rather
// than drifting between per-application copies.
//
// The guard only classifies addresses. Resolving a host and pinning the vetted
// IP against DNS rebinding is the caller's job: that dial path carries
// application-specific policy (per-host allowlists, redirect budgets) that does
// not belong here. Each application composes IsDeniedIP into its own dialer.
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
