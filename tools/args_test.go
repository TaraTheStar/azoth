// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

import "testing"

func TestStrArg(t *testing.T) {
	args := map[string]any{"path": "/x", "n": 3.0}

	if got, err := StrArg(args, "path"); err != nil || got != "/x" {
		t.Fatalf("StrArg = %q, %v", got, err)
	}
	if _, err := StrArg(args, "missing"); !IsArgError(err) {
		t.Errorf("missing key: err = %v, want ArgError", err)
	} else if ae := err.(*ArgError); !ae.Missing {
		t.Error("missing key: ArgError.Missing = false")
	}
	if _, err := StrArg(args, "n"); !IsArgError(err) {
		t.Errorf("wrong type: err = %v, want ArgError", err)
	} else if ae := err.(*ArgError); ae.Missing {
		t.Error("wrong type: ArgError.Missing = true, want false")
	}
}

func TestOptStrArg(t *testing.T) {
	args := map[string]any{"present": "v", "num": 1.0}
	if got, err := OptStrArg(args, "absent", "def"); err != nil || got != "def" {
		t.Fatalf("absent: got %q, %v; want def", got, err)
	}
	if got, err := OptStrArg(args, "present", "def"); err != nil || got != "v" {
		t.Fatalf("present: got %q, %v; want v", got, err)
	}
	if _, err := OptStrArg(args, "num", "def"); err == nil {
		t.Error("present-but-wrong-type should error, not fall back to default")
	}
}

func TestIntArg(t *testing.T) {
	cases := []struct {
		name string
		v    any
		want int
		ok   bool
	}{
		{"json-whole-float", 42.0, 42, true},
		{"native-int", 7, 7, true},
		{"int64", int64(9), 9, true},
		{"fractional", 1.5, 0, false},
		{"string", "3", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := IntArg(map[string]any{"k": c.v}, "k")
			if c.ok {
				if err != nil || got != c.want {
					t.Fatalf("IntArg = %d, %v; want %d", got, err, c.want)
				}
			} else if err == nil {
				t.Fatalf("IntArg accepted %v (=%d), want error", c.v, got)
			}
		})
	}
	if _, err := IntArg(map[string]any{}, "k"); !IsArgError(err) {
		t.Errorf("missing int: err = %v, want ArgError", err)
	}
}

func TestFloatAndBoolArg(t *testing.T) {
	if got, err := FloatArg(map[string]any{"t": 0.7}, "t"); err != nil || got != 0.7 {
		t.Fatalf("FloatArg = %v, %v", got, err)
	}
	if got, err := OptFloatArg(map[string]any{}, "t", 1.0); err != nil || got != 1.0 {
		t.Fatalf("OptFloatArg absent = %v, %v; want 1.0", got, err)
	}
	if got, err := BoolArg(map[string]any{"b": true}, "b"); err != nil || !got {
		t.Fatalf("BoolArg = %v, %v", got, err)
	}
	if _, err := BoolArg(map[string]any{"b": "true"}, "b"); !IsArgError(err) {
		t.Errorf("string-for-bool: err = %v, want ArgError", err)
	}
	if got, err := OptBoolArg(map[string]any{}, "b", true); err != nil || !got {
		t.Fatalf("OptBoolArg absent = %v, %v; want true", got, err)
	}
}

func TestArgErrorMessages(t *testing.T) {
	missing := (&ArgError{Key: "path", Want: "string", Missing: true}).Error()
	if missing != `missing required argument "path"` {
		t.Errorf("missing msg = %q", missing)
	}
	wrongStr := (&ArgError{Key: "path", Want: "string"}).Error()
	if wrongStr != `argument "path" must be a string` {
		t.Errorf("wrong-string msg = %q", wrongStr)
	}
	// "integer" takes "an".
	wrongInt := (&ArgError{Key: "n", Want: "integer"}).Error()
	if wrongInt != `argument "n" must be an integer` {
		t.Errorf("wrong-int msg = %q", wrongInt)
	}
}
