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

## Planned

Roughly in order — see the sibling repos for the current copies:

- `store` — modernc-sqlite open + embedded-migration harness
  (`user_version` cursor, WAL/foreign-keys/busy-timeout pragmas)
- `tools` — the shared `Tool` / `Result` / `Registry` contract
- `netsec` — SSRF/private-range guard
- `bus` — in-process pub/sub

Deliberately *not* here: config structs, store schemas, agent loops,
memory designs — those are per-app products, not shared substance.

## Development

The siblings live side by side; use a `go.work` in the parent directory
to develop against the local copy without version churn.

## License

AGPL-3.0-or-later, same as the siblings. The `llm` package derives from
ensō's `internal/llm`.
