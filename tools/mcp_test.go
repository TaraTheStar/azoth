// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

import (
	"context"
	"testing"
)

func TestMCPToolSatisfiesToolAndDelegates(t *testing.T) {
	var gotArgs map[string]any
	mt := MCPTool[testCtx]{
		ToolName: "remote_search",
		Desc:     "search the remote index",
		Schema:   Object(Str("q", "query").Req()),
		Invoke: func(ctx context.Context, args map[string]any) (Result, error) {
			gotArgs = args
			return Result{LLMOutput: "hit: " + args["q"].(string)}, nil
		},
	}

	// It registers alongside native tools in the same registry.
	r := NewRegistry[testCtx]()
	r.Register(mt)
	r.Register(tool("read"))

	got, ok := r.Get("remote_search")
	if !ok {
		t.Fatal("MCP tool not found in registry")
	}
	if got.Description() != "search the remote index" {
		t.Errorf("Description = %q", got.Description())
	}

	// Run delegates to Invoke and ignores the (nil) app context.
	res, err := got.Run(context.Background(), map[string]any{"q": "x"}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.LLMOutput != "hit: x" {
		t.Errorf("LLMOutput = %q", res.LLMOutput)
	}
	if gotArgs["q"] != "x" {
		t.Errorf("Invoke saw args %v", gotArgs)
	}

	// It contributes a sorted ToolDef like any other tool.
	defs := r.ToolDefs()
	if len(defs) != 2 || defs[0].Function.Name != "read" || defs[1].Function.Name != "remote_search" {
		t.Errorf("ToolDefs = %v, want [read, remote_search]", defs)
	}
}
