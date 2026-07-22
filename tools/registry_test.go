// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

import (
	"context"
	"reflect"
	"sync"
	"testing"
)

// testCtx is a stand-in for an app's request context.
type testCtx struct{ ran []string }

// fakeTool is a minimal Tool[testCtx] for registry tests.
type fakeTool struct {
	name   string
	desc   string
	params map[string]any
	run    func(context.Context, map[string]any, *testCtx) (Result, error)
}

func (f fakeTool) Name() string               { return f.name }
func (f fakeTool) Description() string        { return f.desc }
func (f fakeTool) Parameters() map[string]any { return f.params }
func (f fakeTool) Run(ctx context.Context, args map[string]any, tc *testCtx) (Result, error) {
	if f.run != nil {
		return f.run(ctx, args, tc)
	}
	return Result{LLMOutput: f.name}, nil
}

func tool(name string) fakeTool { return fakeTool{name: name, desc: name + " desc"} }

func TestRegistry_RegisterGetUnregister(t *testing.T) {
	r := NewRegistry[testCtx]()

	if _, ok := r.Get("read"); ok {
		t.Fatal("empty registry returned a tool")
	}

	r.Register(tool("read"))
	got, ok := r.Get("read")
	if !ok {
		t.Fatal("Get miss after Register")
	}
	if got.Name() != "read" {
		t.Fatalf("Get returned %q", got.Name())
	}

	// Register replaces same-named tool.
	r.Register(fakeTool{name: "read", desc: "v2"})
	got, _ = r.Get("read")
	if got.Description() != "v2" {
		t.Errorf("Register did not replace: desc = %q", got.Description())
	}

	r.Unregister("read", "absent") // absent name ignored
	if _, ok := r.Get("read"); ok {
		t.Error("tool present after Unregister")
	}
}

func TestRegistry_ToolDefsSortedAndMemoized(t *testing.T) {
	r := NewRegistry[testCtx]()
	for _, n := range []string{"write", "bash", "read", "edit"} {
		r.Register(tool(n))
	}

	defs := r.ToolDefs()
	got := make([]string, len(defs))
	for i, d := range defs {
		got[i] = d.Function.Name
	}
	want := []string{"bash", "edit", "read", "write"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToolDefs order = %v, want %v (sorted)", got, want)
	}

	// Memoized: same backing slice returned without a mutation.
	first := r.ToolDefs()
	second := r.ToolDefs()
	if &first[0] != &second[0] {
		t.Error("ToolDefs not memoized: different backing array on repeat call")
	}

	// Register busts the cache and picks up the new tool.
	r.Register(tool("grep"))
	if len(r.ToolDefs()) != 5 {
		t.Errorf("ToolDefs len after Register = %d, want 5", len(r.ToolDefs()))
	}

	// Unregister busts the cache too.
	r.Unregister("grep")
	if len(r.ToolDefs()) != 4 {
		t.Errorf("ToolDefs len after Unregister = %d, want 4", len(r.ToolDefs()))
	}
}

func TestRegistry_FilterIndependent(t *testing.T) {
	r := NewRegistry[testCtx]()
	for _, n := range []string{"read", "write", "bash"} {
		r.Register(tool(n))
	}

	child := r.Filter("read", "bash", "absent")
	got := map[string]bool{}
	for _, tl := range child.List() {
		got[tl.Name()] = true
	}
	if len(got) != 2 || !got["read"] || !got["bash"] {
		t.Fatalf("Filter set = %v, want {read, bash}", got)
	}

	// Child is independent: mutating it must not touch the parent.
	child.Unregister("read")
	if _, ok := r.Get("read"); !ok {
		t.Error("Unregister on Filter child mutated the parent")
	}
	// ...and mutating the parent must not touch the child.
	r.Register(tool("edit"))
	if _, ok := child.Get("edit"); ok {
		t.Error("Register on parent leaked into Filter child")
	}
}

func TestRegistry_Without(t *testing.T) {
	r := NewRegistry[testCtx]()
	for _, n := range []string{"read", "write", "bash", "edit"} {
		r.Register(tool(n))
	}
	child := r.Without("write", "edit", "absent")
	if _, ok := child.Get("write"); ok {
		t.Error("Without kept an excluded tool")
	}
	if _, ok := child.Get("read"); !ok {
		t.Error("Without dropped a retained tool")
	}
	if len(child.List()) != 2 {
		t.Errorf("Without child size = %d, want 2", len(child.List()))
	}
	// Parent untouched.
	if len(r.List()) != 4 {
		t.Errorf("Without mutated parent: size = %d, want 4", len(r.List()))
	}
}

func TestRegistry_ConcurrentRegisterAndToolDefs(t *testing.T) {
	r := NewRegistry[testCtx]()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			r.Register(tool(string(rune('a' + i))))
		}(i)
		go func() {
			defer wg.Done()
			_ = r.ToolDefs()
		}()
	}
	wg.Wait()
	if len(r.ToolDefs()) != 8 {
		t.Errorf("final tool count = %d, want 8", len(r.ToolDefs()))
	}
}
