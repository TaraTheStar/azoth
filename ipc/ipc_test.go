// SPDX-License-Identifier: AGPL-3.0-or-later

package ipc

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	type payload struct {
		N int    `json:"n"`
		S string `json:"s"`
	}
	env, err := Pack("submit", payload{N: 7, S: "hi"})
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	var buf bytes.Buffer
	if err := Write(&buf, env); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := Read(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Kind != "submit" {
		t.Fatalf("kind = %q, want submit", got.Kind)
	}
	var p payload
	if err := json.Unmarshal(got.Body, &p); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if p.N != 7 || p.S != "hi" {
		t.Fatalf("body = %+v, want {7 hi}", p)
	}
}

func TestPackNilBody(t *testing.T) {
	env, err := Pack("status", nil)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	if env.Body != nil {
		t.Fatalf("nil body should stay nil, got %q", env.Body)
	}
}

func TestReadCleanEOF(t *testing.T) {
	// An empty reader is a clean close at a frame boundary: io.EOF passes
	// through verbatim so callers distinguish it from a truncated frame.
	_, err := Read(bytes.NewReader(nil))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestReadRejectsBadLength(t *testing.T) {
	for _, n := range []uint32{0, MaxMessage + 1} {
		var prefix [4]byte
		binary.BigEndian.PutUint32(prefix[:], n)
		_, err := Read(bytes.NewReader(prefix[:]))
		if err == nil {
			t.Fatalf("length %d: expected error, got nil", n)
		}
	}
}

func TestReadShortFrame(t *testing.T) {
	// Prefix claims 10 bytes but only 3 follow → short frame, not a clean EOF.
	var prefix [4]byte
	binary.BigEndian.PutUint32(prefix[:], 10)
	r := bytes.NewReader(append(prefix[:], 'a', 'b', 'c'))
	if _, err := Read(r); err == nil {
		t.Fatal("expected short-frame error")
	}
}

func TestWriteRejectsOversize(t *testing.T) {
	big := bytes.Repeat([]byte("x"), MaxMessage)
	env, err := Pack("blob", string(big))
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	if err := Write(io.Discard, env); err == nil {
		t.Fatal("expected oversize write to be refused")
	}
}

func TestCheckPeerUIDNonUnix(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("CheckPeerUID is a no-op off linux")
	}
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	if err := CheckPeerUID(c1); err == nil {
		t.Fatal("expected non-unix connection to be refused")
	}
}

func TestCheckPeerUIDSameUID(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("CheckPeerUID is a no-op off linux")
	}
	sock := filepath.Join(t.TempDir(), "s.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			accepted <- nil
			return
		}
		accepted <- c
	}()

	client, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	srv := <-accepted
	if srv == nil {
		t.Fatal("accept failed")
	}
	defer srv.Close()

	// Same process → same uid → admitted.
	if err := CheckPeerUID(srv); err != nil {
		t.Fatalf("same-uid peer rejected: %v", err)
	}
}
