// SPDX-License-Identifier: AGPL-3.0-or-later

// Package ipc frames the local-socket control protocol an application's CLI
// and daemon speak to each other: a 4-byte big-endian length prefix followed
// by one JSON envelope {kind, body}. Bodies stay json.RawMessage so each side
// decodes lazily per kind — no JSON-RPC machinery and no dependency beyond the
// standard library.
//
// The framing is transport-agnostic substance; the SET of kinds and the typed
// payload structs are per-application vocabulary and stay in the application.
// An app keeps its own typed Kind and Message and adapts to Envelope in two
// lines each way, so the byte-level framing (the length prefix, the size cap,
// the truncated-frame handling) lives — and is audited — in exactly one place.
//
// CheckPeerUID adds the unix-socket peer-credential admission check that
// belongs with this transport: the second lock, behind the socket directory's
// filesystem permissions, that refuses a connection from another user's uid.
package ipc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// MaxMessage bounds a single frame in bytes. A prefix advertising more, or a
// zero-length frame, is a protocol error rather than an allocation: an
// unbounded length prefix from a hostile or corrupt peer would otherwise be an
// out-of-memory lever.
const MaxMessage = 8 << 20

// Envelope is the framed unit: a discriminator plus the kind-specific payload
// as raw JSON. Kind is a plain string so each application can keep its own
// typed Kind constant vocabulary and convert at the seam.
type Envelope struct {
	Kind string          `json:"kind"`
	Body json.RawMessage `json:"body,omitempty"`
}

// Pack builds an Envelope, marshalling body into Body. A nil body yields an
// envelope with no body (an empty-payload verb).
func Pack(kind string, body any) (Envelope, error) {
	if body == nil {
		return Envelope{Kind: kind}, nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Envelope{}, fmt.Errorf("ipc: marshal %q body: %w", kind, err)
	}
	return Envelope{Kind: kind, Body: raw}, nil
}

// Write frames and writes one envelope: a 4-byte big-endian length prefix then
// the JSON. It refuses to emit a frame larger than MaxMessage so a sender
// never puts on the wire something the reader is bound to reject.
func Write(w io.Writer, e Envelope) error {
	payload, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("ipc: marshal envelope: %w", err)
	}
	if len(payload) > MaxMessage {
		return fmt.Errorf("ipc: message too large: %d bytes", len(payload))
	}
	var prefix [4]byte
	binary.BigEndian.PutUint32(prefix[:], uint32(len(payload)))
	if _, err := w.Write(prefix[:]); err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}

// Read reads one framed envelope. A clean EOF at a frame boundary passes
// through verbatim (io.EOF) so callers can tell an orderly close from a
// truncated frame; a partial frame after the prefix reports as a short frame.
func Read(r io.Reader) (Envelope, error) {
	var prefix [4]byte
	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return Envelope{}, err // io.EOF / ErrUnexpectedEOF pass through for a clean close
	}
	n := binary.BigEndian.Uint32(prefix[:])
	if n == 0 || n > MaxMessage {
		return Envelope{}, fmt.Errorf("ipc: bad frame length %d", n)
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Envelope{}, fmt.Errorf("ipc: short frame: %w", err)
	}
	var e Envelope
	if err := json.Unmarshal(payload, &e); err != nil {
		return Envelope{}, fmt.Errorf("ipc: bad envelope: %w", err)
	}
	return e, nil
}
