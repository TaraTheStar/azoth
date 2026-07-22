# Security policy

## Reporting a vulnerability

Please report security issues **privately**. Do not open a public
issue, PR, or discussion thread for anything that looks like a
vulnerability.

Two ways to reach the maintainer:

1. **GitHub private vulnerability reporting** (preferred) — on the
   repo, click **Security → Report a vulnerability**. This opens a
   private advisory only the maintainer can see.
2. **Email** — <TaraTheStar@proton.me>. PGP not currently offered;
   if you need encrypted transport, say so in a first-contact email
   and we'll arrange a key exchange.

Please include:

- A description of the issue and its impact.
- Steps to reproduce, or a proof-of-concept.
- The azoth version (`go list -m github.com/TaraTheStar/azoth`) and
  Go version.
- Whether you'd like credit in the eventual advisory.

## Scope

azoth is a library consumed by other programs, so most of what
matters is how its packages behave on untrusted input.

In scope:

- The `llm` package and its provider subpackages (anthropic,
  bedrock, vertex) — including request/response parsing, streaming
  (SSE) handling, and the loop guard.
- The `netsec` package — anything that weakens the network-security
  posture it's meant to enforce.
- The `store` package — SQL/migration handling and data at rest.
- The `tools` package — schema/argument handling and the MCP client.
- The `paths` package — path construction that could escape an
  intended directory.

Out of scope:

- Third-party MCP servers, LSP servers, or model providers a
  consuming program configures itself.
- Issues that require an attacker who already has code execution as
  the user running the consuming program.
- Misuse of the library by a consumer in a way the docs advise
  against.

## Disclosure

We aim to acknowledge reports within 7 days and ship a fix or a
mitigation plan within 30 days for confirmed issues. Coordinated
disclosure timelines are negotiable for complex bugs; please say so
in your report.
