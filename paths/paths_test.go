// SPDX-License-Identifier: AGPL-3.0-or-later

package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// clearXDG unsets every XDG_* var this package reads and pins $HOME so the
// home-relative defaults are deterministic. Returns the pinned home.
func clearXDG(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	return home
}

func TestBaseDirs_HomeDefaults(t *testing.T) {
	home := clearXDG(t)
	l := Layout{App: "enso"}

	cases := []struct {
		name string
		got  func() (string, error)
		want string
	}{
		{"config", l.ConfigDir, filepath.Join(home, ".config", "enso")},
		{"data", l.DataDir, filepath.Join(home, ".local", "share", "enso")},
		{"state", l.StateDir, filepath.Join(home, ".local", "state", "enso")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.got()
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestBaseDirs_XDGEnvWins(t *testing.T) {
	clearXDG(t)
	t.Setenv("XDG_CONFIG_HOME", "/cfg")
	t.Setenv("XDG_DATA_HOME", "/data")
	t.Setenv("XDG_STATE_HOME", "/state")
	l := Layout{App: "enso"}

	for _, c := range []struct {
		name string
		got  func() (string, error)
		want string
	}{
		{"config", l.ConfigDir, "/cfg/enso"},
		{"data", l.DataDir, "/data/enso"},
		{"state", l.StateDir, "/state/enso"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.got()
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestRuntimeDir_XDGSet(t *testing.T) {
	clearXDG(t)
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	l := Layout{App: "namtar", RuntimeFallback: FallbackToTemp}
	got, err := l.RuntimeDir()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if want := "/run/user/1000/namtar"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRuntimeDir_DefaultFallbackIsState(t *testing.T) {
	home := clearXDG(t)
	l := Layout{App: "enso"} // nil RuntimeFallback → FallbackToState
	got, err := l.RuntimeDir()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if want := filepath.Join(home, ".local", "state", "enso"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRuntimeDir_TempFallback(t *testing.T) {
	clearXDG(t)
	l := Layout{App: "namtar", RuntimeFallback: FallbackToTemp}
	got, err := l.RuntimeDir()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	want := filepath.Join(os.TempDir(), fmt.Sprintf("namtar-%d", os.Getuid()))
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEmptyApp_IsError(t *testing.T) {
	clearXDG(t)
	var l Layout // App == ""
	for _, c := range []struct {
		name string
		got  func() (string, error)
	}{
		{"config", l.ConfigDir},
		{"data", l.DataDir},
		{"state", l.StateDir},
		{"runtime", l.RuntimeDir},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := c.got(); err == nil {
				t.Fatal("want error for empty App, got nil")
			}
		})
	}
}
