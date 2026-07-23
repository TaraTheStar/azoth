// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build linux

package ipc

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// CheckPeerUID rejects a unix-socket connection whose peer runs under a
// different uid than this process. Filesystem permissions on the socket
// directory (0700) and the socket (0600) already enforce this; the
// SO_PEERCRED check is the second lock on the same door — cheap defence in
// depth that also catches a socket accidentally created with looser perms.
//
// It is meaningful only for a *net.UnixConn; a non-unix connection is refused
// so a caller can't mistake a TCP peer for a vouched-for local one. Off Linux
// the check is a no-op (see the !linux build) and the directory perms stand
// alone.
func CheckPeerUID(conn net.Conn) error {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("ipc: not a unix socket connection")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return err
	}
	var cred *unix.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return err
	}
	if credErr != nil {
		return fmt.Errorf("ipc: peercred: %w", credErr)
	}
	if int(cred.Uid) != unix.Getuid() {
		return fmt.Errorf("ipc: peer uid %d is not our uid %d", cred.Uid, unix.Getuid())
	}
	return nil
}
