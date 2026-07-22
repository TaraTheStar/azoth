// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

import (
	"reflect"
	"testing"
)

func TestObject(t *testing.T) {
	schema := Object(
		Str("path", "file to read").Req(),
		Int("limit", "max lines").With("default", 200),
		Enum("mode", "read mode", "text", "bytes").Req(),
	)

	if schema["type"] != "object" {
		t.Fatalf("type = %v, want object", schema["type"])
	}

	props := schema["properties"].(map[string]any)
	if len(props) != 3 {
		t.Fatalf("properties len = %d, want 3", len(props))
	}

	path := props["path"].(map[string]any)
	if path["type"] != "string" || path["description"] != "file to read" {
		t.Errorf("path prop = %v", path)
	}

	limit := props["limit"].(map[string]any)
	if limit["type"] != "integer" || limit["default"] != 200 {
		t.Errorf("limit prop = %v", limit)
	}

	mode := props["mode"].(map[string]any)
	if !reflect.DeepEqual(mode["enum"], []any{"text", "bytes"}) {
		t.Errorf("mode enum = %v", mode["enum"])
	}

	// required is in prop order: path then mode (limit is not required).
	if got, want := schema["required"], []string{"path", "mode"}; !reflect.DeepEqual(got, want) {
		t.Errorf("required = %v, want %v", got, want)
	}
}

func TestObjectNoRequired(t *testing.T) {
	schema := Object(Str("q", "query"))
	if _, has := schema["required"]; has {
		t.Error("required key present when no prop is required")
	}
}

func TestPropWithIsCopyOnWrite(t *testing.T) {
	base := Int("n", "count")
	a := base.With("minimum", 0)
	b := base.With("maximum", 10)
	// base must be unaffected by either derivation.
	if _, has := base.schema["minimum"]; has {
		t.Error("With mutated the base prop's schema")
	}
	if _, has := a.schema["maximum"]; has {
		t.Error("derivation a leaked b's keyword")
	}
	if b.schema["maximum"] != 10 {
		t.Error("derivation b lost its keyword")
	}
}

func TestArrayProp(t *testing.T) {
	schema := Object(Array("tags", "labels", "string").Req())
	tags := schema["properties"].(map[string]any)["tags"].(map[string]any)
	if tags["type"] != "array" {
		t.Fatalf("tags type = %v, want array", tags["type"])
	}
	items := tags["items"].(map[string]any)
	if items["type"] != "string" {
		t.Errorf("items type = %v, want string", items["type"])
	}
}
