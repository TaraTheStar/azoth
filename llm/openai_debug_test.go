// SPDX-License-Identifier: AGPL-3.0-or-later

package llm

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// grabDebugWriter snapshots the current package debug writer.
func grabDebugWriter() io.Writer {
	debugMu.RLock()
	defer debugMu.RUnlock()
	return debugWriter
}

// TestSetDebug_ClosesPreviousWriter guards the file-handle leak: each
// SetDebug call must close the file it previously opened, on both the
// re-enable and the disable path, and toggle the fast-path flag.
func TestSetDebug_ClosesPreviousWriter(t *testing.T) {
	t.Cleanup(func() {
		if err := SetDebug(""); err != nil {
			t.Errorf("cleanup SetDebug(\"\"): %v", err)
		}
	})

	dir := t.TempDir()

	if err := SetDebug(filepath.Join(dir, "a.log")); err != nil {
		t.Fatalf("SetDebug(a): %v", err)
	}
	if !debugEnabled.Load() {
		t.Fatal("debugEnabled should be true after enabling")
	}
	first, ok := grabDebugWriter().(*os.File)
	if !ok {
		t.Fatalf("debugWriter = %T, want *os.File", grabDebugWriter())
	}

	// Re-enable to a new path: the first handle must be closed.
	if err := SetDebug(filepath.Join(dir, "b.log")); err != nil {
		t.Fatalf("SetDebug(b): %v", err)
	}
	if _, err := first.WriteString("x"); !errors.Is(err, os.ErrClosed) {
		t.Errorf("first log file not closed on replace: write err = %v", err)
	}
	second := grabDebugWriter().(*os.File)

	// Disable: the second handle must be closed and the writer discarded.
	if err := SetDebug(""); err != nil {
		t.Fatalf("SetDebug(\"\"): %v", err)
	}
	if debugEnabled.Load() {
		t.Error("debugEnabled should be false after disabling")
	}
	if _, err := second.WriteString("x"); !errors.Is(err, os.ErrClosed) {
		t.Errorf("log file not closed on disable: write err = %v", err)
	}
	if grabDebugWriter() != io.Discard {
		t.Errorf("debugWriter = %T, want io.Discard", grabDebugWriter())
	}

	// Disabling again must be a no-op (io.Discard is never "closed").
	if err := SetDebug(""); err != nil {
		t.Fatalf("second SetDebug(\"\"): %v", err)
	}
}

// TestDebugf covers the exported one-call debug helper used by out-of-package
// adapters: it writes a formatted line when enabled, no-ops when disabled, and
// DebugEnabled tracks the toggle.
func TestDebugf(t *testing.T) {
	t.Cleanup(func() { _ = SetDebug("") })

	if DebugEnabled() {
		t.Fatal("DebugEnabled should be false before SetDebug")
	}

	// Disabled: Debugf must not panic and must write nothing.
	Debugf("dropped %d\n", 1)

	path := filepath.Join(t.TempDir(), "d.log")
	if err := SetDebug(path); err != nil {
		t.Fatalf("SetDebug: %v", err)
	}
	if !DebugEnabled() {
		t.Fatal("DebugEnabled should be true after SetDebug")
	}

	Debugf("hello %s %d\n", "world", 42)

	if err := SetDebug(""); err != nil { // flush+close before reading
		t.Fatalf("SetDebug(\"\"): %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if got, want := string(data), "hello world 42\n"; got != want {
		t.Errorf("log = %q, want %q (a disabled Debugf must not have written 'dropped')", got, want)
	}
}
