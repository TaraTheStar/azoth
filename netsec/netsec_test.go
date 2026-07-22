// SPDX-License-Identifier: AGPL-3.0-or-later

package netsec

import (
	"net"
	"testing"
)

func TestIsDeniedIP(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		deny bool
	}{
		// Denied classes.
		{"nil", "", true}, // parsed nil, fail closed
		{"loopback v4", "127.0.0.1", true},
		{"loopback v4 range", "127.9.9.9", true},
		{"loopback v6", "::1", true},
		{"private 10/8", "10.0.0.1", true},
		{"private 172.16/12", "172.16.5.5", true},
		{"private 192.168/16", "192.168.1.1", true},
		{"ipv6 ula fc00::/7", "fc00::1", true},
		{"ipv6 ula fd", "fd12:3456:789a::1", true},
		{"link-local v4", "169.254.1.1", true},
		{"cloud metadata", "169.254.169.254", true},
		{"link-local v6", "fe80::1", true},
		{"multicast v4", "224.0.0.1", true},
		{"multicast v4 high", "239.255.255.250", true},
		{"multicast v6", "ff02::1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"unspecified v6", "::", true},
		{"cgnat low", "100.64.0.1", true},
		{"cgnat high", "100.127.255.255", true},
		{"this network 0/8", "0.1.2.3", true},
		{"broadcast", "255.255.255.255", true},

		// Allowed: routable public addresses, including near-miss edges.
		{"public v4", "1.1.1.1", false},
		{"public v4 dns", "8.8.8.8", false},
		{"public v4 above cgnat", "100.128.0.1", false},    // 100.128/9 is public
		{"public v4 below cgnat", "100.63.255.255", false}, // 100.0/10-ish, below 100.64
		{"public v4 near private", "172.15.255.255", false},
		{"public v4 near private hi", "172.32.0.1", false},
		{"public v6", "2606:4700:4700::1111", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ip := net.ParseIP(c.ip) // nil for the "nil" case
			if got := IsDeniedIP(ip); got != c.deny {
				t.Fatalf("IsDeniedIP(%q) = %v, want %v", c.ip, got, c.deny)
			}
		})
	}
}
