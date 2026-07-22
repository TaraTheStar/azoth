# azoth

> *Azoth — in alchemy, the universal solvent and universal medicine: the
> one essence present in every work, the agent no transmutation can do
> without. The word spans A to Z — built from the first and last letters
> of the Latin, Greek, and Hebrew alphabets.*

Shared foundation library for [ensō](https://github.com/TaraTheStar/enso),
[namtar](https://github.com/TaraTheStar/namtar), and
[familiar](https://github.com/TaraTheStar/familiar)/grimoire — the common
substance inside every work, extracted so a fix lands once instead of
three times.

## Packages

### `llm`

An OpenAI-compatible streaming chat client, extracted from ensō's
battle-tested implementation:

- SSE stream parsing with truncated-stream detection (a cut connection
  is an error, never a silent "clean finish")
- streamed tool-call reassembly, index-ordered with deterministic
  synthesized IDs — keeps llama.cpp's prompt-prefix cache byte-stable
  across turns
- transport-only retry with backoff (500ms / 1.5s), friendly
  categorized network errors, `Retry-After`-aware API errors
- opt-in bounded retry of retryable statuses (429/502/503/504) that
  honors `Retry-After` — for unattended daemons; off by default
- `Pool` — FIFO slot bounding for shared inference hardware, and
  `PooledClient` to wrap any client with one
- `Complete()` — the one-call ask-and-answer helper for digests,
  summaries, and describe prompts
- stall watchdog (inter-token silence, prefill-safe), mid-stream
  repetition guard, optional reasoning budget — the local-model failure
  modes
- recovery of tool calls that GGUF chat templates leak into assistant
  text or the reasoning channel
- connection-state tracking for UI "reconnecting / disconnected"
  indicators, with a background recovery probe
- usage reporting (`stream_options.include_usage`, cache-read tokens
  surfaced separately)

Two consumption styles, one implementation:

```go
// channel (ensō style)
ch, err := client.Chat(ctx, req)

// iterator (namtar / grimoire style)
for ev, err := range llm.Stream(ctx, client, req) { ... }
```

`llm/llmtest` provides a programmable `ChatClient` mock for driving
agent loops in tests.

#### Cloud backends — `llm/anthropic`, `llm/bedrock`, `llm/vertex`

Non-OpenAI providers, each behind the same `llm.ChatClient` contract, so
a multi-provider host swaps a backend without touching its agent loop:

- `anthropic.Client` — the Anthropic Messages API directly, plus
  `anthropic.BedrockClient` and `anthropic.VertexClient` for the same
  wire protocol routed through Amazon Bedrock / GCP Vertex (prompt
  caching, extended thinking, guardrails carried through)
- `bedrock.Client` — Amazon Bedrock's multi-vendor Converse API
- `vertex.Client` — Google Vertex / Gemini generate-content

Each is a plain struct with exported fields — no constructors — so a host
builds one by literal and hands it around as an `llm.ChatClient`:

```go
c := &anthropic.Client{APIKey: key, Model: "claude-sonnet-4-5", MaxTokens: 16000}
ch, err := c.Chat(ctx, req)
```

These are **subpackages, not part of `llm` itself**, on purpose: the three
heavy cloud SDKs (`anthropic-sdk-go`, `aws-sdk-go-v2`,
`google.golang.org/genai`) stay out of the dependency graph of anything
that imports only `azoth/llm`. `go list -deps ./llm` pulls none of them;
each subpackage pulls only its own.

### `paths`

XDG Base Directory layout, parameterized by application name. A `Layout`
is bound to one app; each helper honors the matching `XDG_*` var and falls
back to the spec default under `$HOME`:

```go
p := paths.Layout{App: "enso"}
cfg, _ := p.ConfigDir()   // $XDG_CONFIG_HOME/enso (else ~/.config/enso)
```

`ConfigDir` / `DataDir` / `StateDir` / `RuntimeDir` are the shared
primitive; app-specific file paths (a db file, a socket, a key) are joined
onto them at the call site. `RuntimeDir`'s behavior when `$XDG_RUNTIME_DIR`
is unset is selectable — `FallbackToState` (default) or `FallbackToTemp`
(a uid-scoped `$TMPDIR/<app>-<uid>`, for a 0700 socket dir) — since the
XDG spec leaves that choice to the application.

### `store`

A schema-agnostic SQLite harness — the open-and-migrate plumbing, without
any application's tables:

```go
db, _ := store.Open(dbPath)              // WAL, foreign_keys, busy_timeout
_ = store.Migrate(db, migrationFS, "migrations")
```

- `Open` creates the parent dir 0700 (clamping a looser pre-existing one),
  opens the pure-Go modernc driver with the standard
  WAL/foreign-keys/busy-timeout pragmas, and Pings before returning the raw
  `*sql.DB` — schema and `Store` wrapper stay in the app.
- `Migrate` applies embedded `NNNN_name.sql` files newer than
  `PRAGMA user_version`, in ascending *numeric* version order (not
  directory order), each body plus its version bump in one transaction so a
  failure rolls back cleanly. Duplicate versions and non-numeric prefixes
  are rejected loudly. It takes an `fs.FS` (not a concrete `embed.FS`), so
  it's unit-testable with `fstest.MapFS` while the app passes its
  `//go:embed migrations/*.sql`.

`store` blank-imports `modernc.org/sqlite`, so callers don't — this is the
one core package that pulls a non-stdlib dependency into azoth.

### `netsec`

The SSRF address-class guard: `IsDeniedIP(net.IP) bool`, the single
decision of whether a model-supplied hostname may become an outbound
connection. It denies loopback, RFC1918 + RFC4193 ULA, link-local (incl.
cloud metadata 169.254.169.254), multicast, unspecified, CGNAT 100.64/10,
0.0.0.0/8, and broadcast; nil fails closed.

```go
if netsec.IsDeniedIP(resolved) { return errBlocked }
```

Only the stdlib `net` — no `llm` dependency — so a consumer that only
needs the classifier doesn't pull anything else in. The resolve-and-pin
dialer that *uses* it stays per-app (each host has its own allow-list and
egress policy); azoth shares only the classification.

### `tools`

The shared tool contract for agent hosts: a generic `Tool[Ctx any]`
interface, a unified `Result` (+ `ResultMeta` for paths-read/written and
cache keys), and a goroutine-safe `Registry[Ctx]` (`Register` /
`Unregister` / `Get(name) (T, bool)` / `List` / `Filter` / `Without` /
`ToolDefs`, with a memoized name-sorted `[]llm.ToolDef`). Each app adopts
via a type alias, supplying its own request-context type and tool set:

```go
type Tool = tools.Tool[AgentContext]
type Registry = tools.Registry[AgentContext]
```

An opt-in helper layer rides alongside — typed argument extractors
(`StrArg` / `IntArg` / `FloatArg` / `BoolArg` + `Opt*` variants, with a
typed `ArgError`), JSON-schema builders (`Object` / `Prop` / …), and an
`MCPTool[Ctx]` adapter shape — so tool authors share ergonomics, not just
the registry. MCP remains the runtime-plugin seam; azoth doesn't add a
second plugin loader.

## Not shared — on purpose

`bus` (in-process pub/sub) was evaluated and deliberately left per-app: the
two implementations share only a ~15-line fan-out idiom, and their valuable
machinery (namtar's replay ring + sequence stamping vs. ensō's typed
wire-form + slow-consumer accounting) can't sit on a common core cleanly.

Also intentionally out: config structs, store schemas and query surfaces,
agent loops, memory designs, and embeddings — those are per-app products,
not shared substance.

## Development

The siblings live side by side; use a `go.work` in the parent directory
to develop against the local copy without version churn.

## License

AGPL-3.0-or-later, same as the siblings. The `llm` package derives from
ensō's `internal/llm`.
