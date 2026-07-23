// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !linux

package ipc

import "net"

// CheckPeerUID is a no-op off Linux, where SO_PEERCRED is unavailable; the
// 0700 runtime directory and 0600 socket carry the access control alone. It is
// defined for every platform so daemon code calls it unconditionally.
func CheckPeerUID(conn net.Conn) error { return nil }
