# Changelog

All notable changes to azoth are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Every release is an annotated tag whose message carries the same notes as
its entry below. Entries are written for the three consumers — ensō,
namtar, and familiar/grimoire — so anything that changes an import path, a
type name, or an egress/IPC behaviour is called out explicitly.

**Everything through v1.1.0 is retracted** (`retract [v0.1.0, v1.1.0]` in
`go.mod`): those tags carried a machine-specific absolute path inside a
test fixture, and the fix was a history rewrite rather than a patch.
v1.1.1 republishes v1.1.0's library content, scrubbed. Their tags no
longer exist in the repository; the entries below are reconstructed from
the module proxy and the rewritten history, kept for the record.

## [Unreleased]

_Nothing yet._

## [1.1.3] — 2026-08-01

Security release. No API change.

### Security

- `golang.org/x/text` 0.36.0 → 0.40.0. GO-2026-5970 was reachable from
  `llm/openai.go`'s chat path — every consumer's request reaches
  `norm.Form` through `http.Client.Do` — so this is callable, not merely
  present in the graph. Lands azoth on the version ensō and namtar already
  use.
- `golang.org/x/crypto` 0.50.0 → 0.52.0, closing thirteen advisories
  carried indirectly through the AWS and Google SDKs.
- `golang.org/x/net` 0.53.0 → 0.55.0.

### Other

- Dependency bumps collected since v1.1.1, all via Dependabot:
  `golang.org/x/sys` 0.47.0, `golang.org/x/sync` 0.22.0 (via `go mod
  tidy`), `modernc.org/sqlite` 1.55.0, `anthropic-sdk-go` 1.61.0,
  `aws-sdk-go-v2/service/bedrockruntime` 1.56.2, `aws-sdk-go-v2/config`
  1.32.32, and the transitive AWS SDK core packages.
- Lineage repair: v1.1.2 was cut on a pull-request branch and is not
  reachable from `main` (it lacks the `x/sys` 0.47.0 bump that merged
  alongside it). This tag sits on `main` and its content is a superset, so
  `main` describes itself correctly again from here.

## [1.1.2] — 2026-07-23

### Changed

- `go.mod` declares `retract [v0.1.0, v1.1.0]`, so `go list -m -versions`
  and `go get` stop offering the pre-scrub line. No code change.

> Not reachable from `main` — see the lineage note under v1.1.3. Prefer
> v1.1.3.

## [1.1.1] — 2026-07-23

Republication of v1.1.0 with the offending test fixture path scrubbed.
Library content is otherwise identical to v1.1.0 — no API or behaviour
change between the two.

## 1.1.0 — 2026-07-23 — retracted

Three additions, each extracted from a place where ensō and namtar had
independently duplicated it.

### Added

- `netsec.Dialer` / `netsec.GuardedClient` / `netsec.DeniedError` — the
  resolve-check-pin dial path that turns `IsDeniedIP` into an enforcing
  dialer. Refuses if any resolved address is denied, then dials the vetted
  IP literals in order, so DNS cannot rebind between check and connect.
  `Exempt` is the operator-configured opt-out seam; `DeniedError` lets a
  caller reword the refusal with app-specific remediation. `GuardedClient`
  wraps a no-exemptions `Dialer` in a ready `*http.Client`.
- `ipc` — length-prefixed `{kind, body}` `Envelope` framing (`Pack` /
  `Write` / `Read`, 8 MiB cap, truncated-frame detection) plus
  `CheckPeerUID`, the unix-socket `SO_PEERCRED` admission check (no-op off
  Linux). Apps keep their own typed `Kind` vocabulary and adapt at the
  seam; only the byte-level framing is shared.
- `llm/llmtest.Call(id, name, args)` — compact tool-call literal
  constructor for scripted-mock tests.

### Changed

- `golang.org/x/sys` is promoted from indirect to a direct dependency
  (required by `ipc`'s peercred path).

### Other

- CI (gofmt / vet / test with read-only workflow permissions), weekly
  Dependabot for gomod and github-actions, and a `SECURITY.md` scoped to
  the library's packages.
- Dependency bumps: `google.golang.org/genai` 1.65.0,
  `anthropic-sdk-go` 1.59.0, `modernc.org/sqlite` 1.54.0,
  `aws-sdk-go-v2/service/bedrockruntime` 1.56.0.
- README records why prefix-gated config env-expansion was evaluated for
  sharing and held back (common threat model, divergent integration).

## 1.0.0 — 2026-07-22 — retracted

API freeze. Pure rename plus documentation — no dependency or behaviour
change.

### Changed

- **Breaking.** Cloud clients take package-qualified names:
  `AnthropicClient` → `anthropic.Client`, `AnthropicBedrockClient` →
  `anthropic.BedrockClient`, `AnthropicVertexClient` →
  `anthropic.VertexClient`, `BedrockClient` → `bedrock.Client`,
  `VertexClient` → `vertex.Client`. Inside `llm/anthropic` the SDK import
  is aliased to `anthropicsdk` so the vendor's own `anthropic.Client`
  never reads ambiguously against ours.
- README promotes `netsec` and `tools` out of "Planned" (both shipped),
  documents the `llm/{anthropic,bedrock,vertex}` backends and their
  SDK-isolation design, and records the bus skip under "not shared".

## 0.6.1 — 2026-07-22 — retracted

### Security

- `google.golang.org/grpc` → v1.82.1 (CVE fix), reached transitively
  through the Vertex backend.

## 0.6.0 — 2026-07-22 — retracted

### Added

- `llm/anthropic` — direct Messages API, plus the anthropic-bedrock and
  anthropic-vertex routing variants (all `anthropic-sdk-go` based).
- `llm/bedrock` — native Converse API (`aws bedrockruntime`).
- `llm/vertex` — Gemini generate-content (`google genai`).

  Subpackages, not `llm` proper, so the three heavy cloud SDKs stay out of
  the dependency graph of consumers that import only `azoth/llm`: `go list
  -deps ./llm` pulls none of them, and each subpackage pulls only its own
  SDK.

## 0.5.0 — 2026-07-22 — retracted

### Added

- `tools` — the shared tool contract, generic over each app's request
  context: `Tool[Ctx]`, `Result`, `ResultMeta`, and a goroutine-safe
  `Registry[Ctx]` (`Register` / `Unregister` / `Get` / `List` / `Filter` /
  `Without` / `ToolDefs`) whose `ToolDefs` is memoized and name-sorted for
  prompt-prefix-cache stability. Opt-in helpers: typed arg extractors
  (`StrArg` / `IntArg` / `FloatArg` / `BoolArg` and `Opt` variants) with a
  typed `ArgError`, JSON-schema builders, and an `MCPTool` adapter shape.
  Imports only `azoth/llm` — no new external dependencies.

### Changed

- **Breaking.** `llm` exports the transport machinery out-of-package
  adapters lean on: `connTracker` → `ConnTracker` (methods `Get` / `Set` /
  `ClaimProbe` / `ReleaseProbe`; fields stay unexported and `OpenAIClient`
  embeds the exported type), plus `ClassifyTransportError`,
  `FriendlyHTTPError`, `Debugf`, and `DebugEnabled`.

## 0.4.0 — 2026-07-22 — retracted

### Added

- `netsec.IsDeniedIP` — the address-class classification deciding whether
  a model-supplied name may become an outbound TCP connection, extracted
  from the copies in ensō and namtar so a newly recognised dangerous range
  closes on every egress path at once. Covers loopback, RFC1918 and
  RFC4193 ULA, link-local including cloud metadata, multicast,
  unspecified, CGNAT 100.64/10, 0.0.0.0/8, and broadcast; nil fails
  closed. Resolve-and-pin stays per-app at this point — it carries
  application-specific policy.

## 0.3.0 — 2026-07-22 — retracted

### Added

- `store` — `Open` creates the parent directory 0700 (clamping a looser
  pre-existing one), opens the pure-Go modernc driver with the standard
  WAL / foreign_keys / busy_timeout pragmas, pings, and returns a raw
  `*sql.DB`. `Migrate` applies embedded `NNNN_name.sql` files newer than
  `PRAGMA user_version` in ascending numeric order, each body plus its
  version bump in one transaction; duplicate versions and non-numeric
  prefixes are rejected. Takes an `fs.FS`, so the runner is
  `fstest`-testable. Schema-agnostic by design: each app keeps its own
  wrapper, query surface, and migrations tree.
- First external dependency: `modernc.org/sqlite` (pure Go, no cgo).

## 0.2.0 — 2026-07-22 — retracted

### Added

- `paths` — XDG base-directory resolver bound to one application name that
  suffixes every base dir; each helper honours the matching `XDG_*` env
  var first and falls back to the spec default under `$HOME`.
  `ConfigDir` / `DataDir` / `StateDir` / `RuntimeDir`. `RuntimeDir`'s
  behaviour when `$XDG_RUNTIME_DIR` is unset — which the spec leaves to
  the application — is selectable: `FallbackToState` (default) or
  `FallbackToTemp` (`$TMPDIR/<app>-<uid>`). An empty `Layout.App` is a
  hard error, never a silent app-less XDG path.

## 0.1.0 — 2026-07-22 — retracted

Initial extraction of ensō's OpenAI-compatible LLM client.

### Added

- `llm` — SSE streaming chat with truncated-stream detection, streamed
  tool-call reassembly with deterministic synthesized IDs, transport-only
  retry with backoff, categorized network errors, `Retry-After`-aware API
  errors, stall watchdog, mid-stream repetition guard, recovery of tool
  calls leaked into assistant text or the reasoning channel,
  connection-state tracking with a background recovery probe, and usage
  reporting.
- `llm.Stream` — an `iter.Seq2` surface over any `ChatClient`,
  cancel-and-drain safe on early break, alongside the channel-based
  `Chat`.
- `ConnectError.Hint`, fed by `OpenAIClient.ConnectHint`, so each app
  supplies its own config-path remediation text.
- `Complete(ctx, client, req)` — stream internally, return collected text
  and usage; rejects tool calls as misuse rather than dropping them.
- `Pool` (FIFO direct hand-off with queue timeout) and `PooledClient`,
  which wraps any `ChatClient` so each `Chat` holds a slot until its event
  stream closes.
- `OpenAIClient.StatusRetries` / `StatusRetryMaxWait` — bounded retry of
  429/502/503/504 honouring `Retry-After` (capped, default 30s; over-cap
  surfaces the `APIError` instead of sleeping). 500 is deliberately
  excluded. Off by default, so interactive apps keep fail-fast behaviour.
- `OpenAIClient.Temperature *float64` — per-client default applied to
  requests that don't set their own; nil sends nothing.
- `ToolCall.Function` gets a named `FunctionCall` type so downstream code
  can write tool-call literals.
- `llm/llmtest` — programmable `ChatClient` mock: scheduled `Script`
  turns, recorded `Calls`, and `NewT` for leftover-script assertions.

[Unreleased]: https://github.com/TaraTheStar/azoth/compare/v1.1.3...HEAD
[1.1.3]: https://github.com/TaraTheStar/azoth/releases/tag/v1.1.3
[1.1.2]: https://github.com/TaraTheStar/azoth/releases/tag/v1.1.2
[1.1.1]: https://github.com/TaraTheStar/azoth/releases/tag/v1.1.1
