// SPDX-License-Identifier: AGPL-3.0-or-later

// Package paths resolves an application's XDG Base Directory layout so call
// sites don't re-derive ~/.config/<app>, ~/.local/share/<app>, and friends.
// Each helper honours the matching XDG_* environment variable first and
// falls back to the spec-defined default under $HOME.
//
// A Layout is bound to one application name, which suffixes every base dir:
//
//	p := paths.Layout{App: "enso"}
//	cfg, err := p.ConfigDir()   // $XDG_CONFIG_HOME/enso (else ~/.config/enso)
//
//	ConfigDir  → $XDG_CONFIG_HOME/<app>  (else ~/.config/<app>)
//	DataDir    → $XDG_DATA_HOME/<app>    (else ~/.local/share/<app>)
//	StateDir   → $XDG_STATE_HOME/<app>   (else ~/.local/state/<app>)
//	RuntimeDir → $XDG_RUNTIME_DIR/<app>  (else RuntimeFallback, see below)
//
// The four helpers are the shared primitive. Application-specific file paths
// (a database file, a unix socket, a key) are derived by joining onto these
// at the call site — that composition stays in each application, not here.
package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Layout resolves the XDG base directories for a single application. The
// zero value is unusable — App is required; construct it as a literal:
//
//	paths.Layout{App: "enso"}                                    // RuntimeDir → StateDir when unset
//	paths.Layout{App: "namtar", RuntimeFallback: paths.FallbackToTemp}
type Layout struct {
	// App is the per-application directory suffix (e.g. "enso" →
	// ~/.config/enso). Required; every helper returns an error when App
	// is empty rather than resolving a bare, app-less XDG directory.
	App string

	// RuntimeFallback resolves RuntimeDir when $XDG_RUNTIME_DIR is unset —
	// a case the XDG spec leaves to the application (systems without
	// pam_systemd, non-interactive sessions, containers). nil defaults to
	// FallbackToState, placing runtime files beside logs/state.
	RuntimeFallback func(Layout) (string, error)
}

// FallbackToState resolves RuntimeDir to StateDir. The default policy; it
// suits applications whose runtime files (sockets, pidfiles) can live
// beside their persistent state.
func FallbackToState(l Layout) (string, error) { return l.StateDir() }

// FallbackToTemp resolves RuntimeDir to a uid-scoped directory under the OS
// temp dir ($TMPDIR/<app>-<uid>). Suits applications that stage a 0700
// socket directory there when no session runtime dir exists — the mode is
// the caller's to set; this only computes the path.
func FallbackToTemp(l Layout) (string, error) {
	if l.App == "" {
		return "", errAppEmpty
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", l.App, os.Getuid())), nil
}

var errAppEmpty = errors.New("paths: Layout.App is empty")

// ConfigDir returns the directory for user-editable configuration.
func (l Layout) ConfigDir() (string, error) { return l.homeDir("XDG_CONFIG_HOME", ".config") }

// DataDir returns the directory for app-managed persistent data.
func (l Layout) DataDir() (string, error) { return l.homeDir("XDG_DATA_HOME", ".local", "share") }

// StateDir returns the directory for logs and similar state that survives
// restarts but isn't portable.
func (l Layout) StateDir() (string, error) { return l.homeDir("XDG_STATE_HOME", ".local", "state") }

// RuntimeDir returns the directory for ephemeral runtime files. When
// $XDG_RUNTIME_DIR is set it wins; otherwise RuntimeFallback decides (or
// FallbackToState when RuntimeFallback is nil).
func (l Layout) RuntimeDir() (string, error) {
	if l.App == "" {
		return "", errAppEmpty
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, l.App), nil
	}
	fb := l.RuntimeFallback
	if fb == nil {
		fb = FallbackToState
	}
	return fb(l)
}

// homeDir resolves one XDG base dir: $envVar/<app> when the variable is
// set, else $HOME joined with homeRel... and the app suffix.
func (l Layout) homeDir(envVar string, homeRel ...string) (string, error) {
	if l.App == "" {
		return "", errAppEmpty
	}
	if d := os.Getenv(envVar); d != "" {
		return filepath.Join(d, l.App), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	parts := make([]string, 0, len(homeRel)+2)
	parts = append(parts, home)
	parts = append(parts, homeRel...)
	parts = append(parts, l.App)
	return filepath.Join(parts...), nil
}
