// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

import "context"

// MCPTool adapts a remote MCP tool into a native Tool[Ctx] so built-in and
// MCP-backed tools share one Registry and are indistinguishable to the agent
// loop. It is the small skeleton both apps re-implemented; each keeps its own
// MCP client wiring and supplies it as the Invoke closure.
//
// Ctx is ignored by Run: an MCP tool executes on the remote server and has no
// use for the app's request context. The type parameter exists only so the
// adapter satisfies Tool[Ctx] and can live in the same registry as native
// tools. If a tool needs the app context, write a native Tool instead.
//
//	reg.Register(tools.MCPTool[AgentContext]{
//		ToolName: remote.Name,
//		Desc:     remote.Description,
//		Schema:   remote.InputSchema,
//		Invoke: func(ctx context.Context, args map[string]any) (tools.Result, error) {
//			return client.Call(ctx, remote.Name, args) // app's own MCP wiring
//		},
//	})
type MCPTool[Ctx any] struct {
	ToolName string
	Desc     string
	Schema   map[string]any
	// Invoke performs the remote call. Required; a nil Invoke panics on Run so
	// the wiring bug surfaces immediately rather than as a silent empty result.
	Invoke func(ctx context.Context, args map[string]any) (Result, error)
}

func (m MCPTool[Ctx]) Name() string               { return m.ToolName }
func (m MCPTool[Ctx]) Description() string        { return m.Desc }
func (m MCPTool[Ctx]) Parameters() map[string]any { return m.Schema }

// Run invokes the remote MCP tool, ignoring the app context.
func (m MCPTool[Ctx]) Run(ctx context.Context, args map[string]any, _ *Ctx) (Result, error) {
	return m.Invoke(ctx, args)
}
