// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

import (
	"errors"
	"fmt"
	"math"
)

// ArgError is returned by the argument extractors when a required argument is
// missing or has the wrong type. It is typed so a tool can errors.As it to
// tell a model-input problem (worth returning to the model as a correction)
// apart from an internal failure. Adopting the extractors is opt-in — tools
// may still assert args by hand.
type ArgError struct {
	Key     string // the argument name
	Want    string // expected type, e.g. "string", "integer"
	Missing bool   // true if absent entirely (vs present-but-wrong-type)
}

func (e *ArgError) Error() string {
	if e.Missing {
		return fmt.Sprintf("missing required argument %q", e.Key)
	}
	return fmt.Sprintf("argument %q must be %s %s", e.Key, article(e.Want), e.Want)
}

// article picks "a"/"an" for the type name in the error message.
func article(want string) string {
	if want == "" {
		return "a"
	}
	switch want[0] {
	case 'a', 'e', 'i', 'o', 'u':
		return "an"
	default:
		return "a"
	}
}

// IsArgError reports whether err is (or wraps) an *ArgError.
func IsArgError(err error) bool {
	var ae *ArgError
	return errors.As(err, &ae)
}

// StrArg extracts a required string argument.
func StrArg(args map[string]any, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", &ArgError{Key: key, Want: "string", Missing: true}
	}
	s, ok := v.(string)
	if !ok {
		return "", &ArgError{Key: key, Want: "string"}
	}
	return s, nil
}

// OptStrArg extracts an optional string argument, returning def when the key
// is absent. A present-but-non-string value is still an error.
func OptStrArg(args map[string]any, key, def string) (string, error) {
	if _, ok := args[key]; !ok {
		return def, nil
	}
	return StrArg(args, key)
}

// FloatArg extracts a required number argument as a float64. JSON numbers
// decode as float64, so this is the natural numeric extractor.
func FloatArg(args map[string]any, key string) (float64, error) {
	v, ok := args[key]
	if !ok {
		return 0, &ArgError{Key: key, Want: "number", Missing: true}
	}
	f, ok := toFloat(v)
	if !ok {
		return 0, &ArgError{Key: key, Want: "number"}
	}
	return f, nil
}

// OptFloatArg extracts an optional number argument, returning def when absent.
func OptFloatArg(args map[string]any, key string, def float64) (float64, error) {
	if _, ok := args[key]; !ok {
		return def, nil
	}
	return FloatArg(args, key)
}

// IntArg extracts a required integer argument. JSON numbers arrive as float64,
// so a whole-valued float (42.0) is accepted; a fractional value (1.5) is an
// error. Native int/int64 values are accepted too.
func IntArg(args map[string]any, key string) (int, error) {
	v, ok := args[key]
	if !ok {
		return 0, &ArgError{Key: key, Want: "integer", Missing: true}
	}
	i, ok := toInt(v)
	if !ok {
		return 0, &ArgError{Key: key, Want: "integer"}
	}
	return i, nil
}

// OptIntArg extracts an optional integer argument, returning def when absent.
func OptIntArg(args map[string]any, key string, def int) (int, error) {
	if _, ok := args[key]; !ok {
		return def, nil
	}
	return IntArg(args, key)
}

// BoolArg extracts a required boolean argument.
func BoolArg(args map[string]any, key string) (bool, error) {
	v, ok := args[key]
	if !ok {
		return false, &ArgError{Key: key, Want: "boolean", Missing: true}
	}
	b, ok := v.(bool)
	if !ok {
		return false, &ArgError{Key: key, Want: "boolean"}
	}
	return b, nil
}

// OptBoolArg extracts an optional boolean argument, returning def when absent.
func OptBoolArg(args map[string]any, key string, def bool) (bool, error) {
	if _, ok := args[key]; !ok {
		return def, nil
	}
	return BoolArg(args, key)
}

// toFloat coerces the numeric shapes a decoded JSON value may take.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// toInt coerces to int, rejecting fractional or out-of-range values.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		if n != math.Trunc(n) || n > math.MaxInt64 || n < math.MinInt64 {
			return 0, false
		}
		return int(n), true
	case float32:
		return toInt(float64(n))
	default:
		return 0, false
	}
}
