// SPDX-License-Identifier: AGPL-3.0-or-later

// Package tools is the shared tool contract for the azoth siblings. It owns
// the pieces that were byte-for-byte the same across the apps — the Tool
// interface shape, the Result an invocation returns, and the goroutine-safe
// Registry that turns a tool set into the model-facing []llm.ToolDef — while
// leaving each app its own request context, its own tool implementations, and
// its own default-set wiring.
//
// The one thing that genuinely differs between apps is the request-scoped
// value threaded into every Run: enso's ~34-field AgentContext vs namtar's
// small Context. So Tool and Registry are parameterized on that type:
//
//	// enso
//	type Tool = tools.Tool[AgentContext]
//	type Registry = tools.Registry[AgentContext]
//	// namtar
//	type Tool = tools.Tool[Context]
//	type Registry = tools.Registry[Context]
//
// Each app aliases these, supplies its own Ctx, and keeps its built-in tools,
// its BuildDefault, and its MCP client wiring. Because the tool contexts are
// near-disjoint, tools are NOT portable between apps — that is not the goal.
// The goal is one contract (so a tool reads the same in either repo), one
// correct Registry (identical byte-stable ToolDefs sort, which the prompt
// prefix cache depends on), and a shared authoring-helper library (see
// args.go / schema.go).
//
// MCP remains the runtime-plugin seam: external tools enter through each app's
// MCP client, wrapped as native Tools via the shared adapter shape in mcp.go.
// This package is not a second cross-app plugin loader.
package tools

import (
	"context"

	"github.com/TaraTheStar/azoth/llm"
)

// Tool is one callable capability offered to the model. Ctx is the app's
// request-scoped context type, passed by pointer to Run. Built-in tools and
// MCP-adapted tools implement the same interface, so the agent loop can't tell
// them apart.
type Tool[Ctx any] interface {
	Name() string
	Description() string
	Parameters() map[string]any // JSON Schema for the tool's arguments
	Run(ctx context.Context, args map[string]any, tc *Ctx) (Result, error)
}

// Result separates what the model sees from what the app keeps. It is the
// superset of the apps' result shapes: an app that doesn't use a field simply
// leaves it zero (e.g. namtar sets only LLMOutput/FullOutput).
type Result struct {
	// LLMOutput is the (possibly truncated) text fed back to the model.
	LLMOutput string
	// FullOutput is the complete output persisted by the app; when empty the
	// app falls back to LLMOutput.
	FullOutput string
	// DisplayOutput is optional terse line(s) for scrollback; falls back to
	// LLMOutput when empty.
	DisplayOutput string
	// Display carries rich display data for the UI (e.g. a diff for a
	// permission modal). Opaque to this package.
	Display any
	// Meta carries side-channel metadata for the app's context-pruning
	// machinery. Zero values are safe (no effect).
	Meta ResultMeta
	// Parts carries non-text content the tool wants the model to see (e.g. an
	// image read off disk). When populated, the agent loop wraps the
	// tool-result message with these parts so the vendor adapter can translate
	// them to the provider's multimodal shape. LLMOutput is still set for
	// adapters that don't speak images and for text-only persistence.
	Parts []llm.MessagePart
}

// ResultMeta carries optional side-channel metadata a tool can populate to
// drive an app's context-pruning. All fields are optional; the zero value has
// no effect.
type ResultMeta struct {
	// PathsRead is the set of absolute paths whose contents this tool
	// surfaced to the model. Used to invalidate stale reads after a write and
	// to mark referenced paths as pinned against compaction.
	PathsRead []string
	// PathsWritten is the set of absolute paths this tool modified. Any prior
	// read of a path in this set can be stubbed when the write is appended.
	PathsWritten []string
	// CacheKey is a normalized identifier for same-call dedup: when two tool
	// results share a CacheKey the older can be stubbed. E.g.
	// "read:/abs/path:1-200", "bash:git status".
	CacheKey string
}
