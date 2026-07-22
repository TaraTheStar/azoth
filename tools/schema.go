// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

// The schema builders assemble the JSON-schema map a tool returns from
// Parameters(). Both apps hand-assemble these maps repeatedly; the builders
// dedup that boilerplate while producing the exact same shape:
//
//	func (t MyTool) Parameters() map[string]any {
//		return tools.Object(
//			tools.Str("path", "file to read").Req(),
//			tools.Int("limit", "max lines").With("default", 200),
//			tools.Enum("mode", "how to read", "text", "bytes"),
//		)
//	}
//
// Adoption is opt-in; a tool may still return a literal map.

// Prop is one property in an object schema. Construct with Str/Num/Int/Bool/
// Enum/Array, then chain Req/With. The zero value is not useful.
type Prop struct {
	name     string
	schema   map[string]any
	required bool
}

func prop(name, typ, desc string) Prop {
	s := map[string]any{"type": typ}
	if desc != "" {
		s["description"] = desc
	}
	return Prop{name: name, schema: s}
}

// Str is a string property.
func Str(name, desc string) Prop { return prop(name, "string", desc) }

// Num is a number (float) property.
func Num(name, desc string) Prop { return prop(name, "number", desc) }

// Int is an integer property.
func Int(name, desc string) Prop { return prop(name, "integer", desc) }

// Bool is a boolean property.
func Bool(name, desc string) Prop { return prop(name, "boolean", desc) }

// Enum is a string property constrained to the given values.
func Enum(name, desc string, values ...string) Prop {
	p := prop(name, "string", desc)
	vs := make([]any, len(values))
	for i, v := range values {
		vs[i] = v
	}
	p.schema["enum"] = vs
	return p
}

// Array is an array property whose items are of the given JSON-schema type
// (e.g. "string").
func Array(name, desc, itemType string) Prop {
	p := prop(name, "array", desc)
	p.schema["items"] = map[string]any{"type": itemType}
	return p
}

// Req marks the property required in its enclosing Object.
func (p Prop) Req() Prop {
	p.required = true
	return p
}

// With attaches an extra JSON-schema keyword to the property (e.g. "default",
// "minimum", "pattern"). It copies the property's schema so chained calls and
// reused Prop values stay independent.
func (p Prop) With(key string, val any) Prop {
	s := make(map[string]any, len(p.schema)+1)
	for k, v := range p.schema {
		s[k] = v
	}
	s[key] = val
	p.schema = s
	return p
}

// Object assembles props into a JSON-schema object for a tool's Parameters().
// The "required" list is emitted in the order the required props were passed,
// so the serialized schema is byte-stable across calls (the prompt-prefix
// cache depends on it). An object with no properties still returns a valid
// empty-object schema.
func Object(props ...Prop) map[string]any {
	properties := make(map[string]any, len(props))
	var required []string
	for _, p := range props {
		properties[p.name] = p.schema
		if p.required {
			required = append(required, p.name)
		}
	}
	obj := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		obj["required"] = required
	}
	return obj
}
