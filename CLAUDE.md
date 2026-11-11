# CLAUDE.md

Notes for coding agents working in go-stt: a single-package Go library (`package stt`, module
`github.com/anatolykoptev/go-stt`) that talks to OpenAI-compatible speech-to-text servers over HTTP
and WebSocket. Every document is indexed in [docs/INDEX.md](docs/INDEX.md).

## Layout

One package at the repository root, split by concern: `client.go` (Client, options, `New`,
`IsAvailable`), `transcribe.go` and `multipart.go` (batch calls), `telegram.go` (`TranscribeURL`),
`retry.go` (retry with backoff, circuit breaker), `errors.go`, `types.go`, and the streaming files
`stream.go` (`StreamClient`), `stream_channel.go`, `stream_file.go`, `stream_types.go`. Tests sit
next to the code; `goleak_test.go` fails the suite on a leaked goroutine.

## Gate

CI (`.github/workflows/preflight.yml`, job `preflight`, `ubuntu-24.04-arm`, Go `stable`) runs
`make preflight`: `gofmt -l -s` must print nothing, `go vet .`, `go build .`,
`GOWORK=off go test -race -count=1 .`. Build, vet and test target `.` only, never `./...`.
A nightly workflow runs `go test -race -count=3 -timeout 300s .`. `make lint` (errcheck and
staticcheck at pinned versions) is deliberately not in preflight, because errcheck flags many
`_, _ =` patterns in tests. A docs-only PR (Markdown, `docs/`, licence, issue and PR templates)
skips the `preflight` job through the `docs-scope` job in the same workflow; that block is
canonical across the fleet and compared byte for byte, so do not edit it here.

## Invariants that are easy to break

- **Retry only transient errors.** `Error.IsTransient` is true for HTTP 429, 502, 503, 504; a 4xx
  must never be retried. The multipart body is built once into a byte snapshot and replayed on
  every attempt.
- **The circuit breaker has an explicit half-open state** with a single probe request; its clock is
  injectable (`setClock`) so tests do not sleep. Keep both.
- **Configuration errors panic at construction.** `WithRetry` panics on `maxAttempts` outside
  1..100 or a non-positive delay, by design: the `Option` and `New` signatures return no error.
- **`StreamParams.VAD` and `Punctuate` default to true when nil.** That is why they are `*bool`.
- **Streaming is bounded.** Inbound messages are capped at 10 MiB, `Close` waits at most 5 s for a
  graceful close, `ForceClose` and `Close` are idempotent, and a second `Connect` is refused. The
  events channel (64) drops on overflow and counts drops in `DroppedEvents`.
- **WebSocket URLs are derived with `net/url`**, not string replacement; a base URL with no
  `http`, `https`, `ws` or `wss` scheme is an error surfaced by `Connect`.
- **The API key goes in the `Authorization: Bearer` header** on every HTTP request and on the
  WebSocket upgrade. Never put a key in a URL, a log line or an error message.
- **Defaults are API.** Language `ru`, model `moonshine-v2`, format `json`, timeout 60 s, stream
  language `en`, 16 kHz `pcm_s16le`. Changing one changes every consumer.
- **Versions come from release-please**, which reads Conventional Commits. Never tag or edit
  `CHANGELOG.md` or `.release-please-manifest.json` by hand.
- **This repository is public.** No hostnames, IP addresses, operator paths, account names, tokens
  or personal data in any tracked file, tests and fixtures included; use `example.com` for
  hosts in examples and tests.
