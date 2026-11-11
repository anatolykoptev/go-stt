# Contributing to go-stt

go-stt is a small client library; other services depend on its defaults and error behaviour, so
open an issue before changing either. Report vulnerabilities privately, see [SECURITY.md](SECURITY.md).

## Development

You need Go (the `go` line in `go.mod` is the minimum; CI uses the current stable release).

```sh
make preflight   # the CI gate: gofmt -s check, go vet, go build, go test -race -count=1
make lint        # errcheck -blank and staticcheck at pinned versions; not part of preflight
```

All targets build and test the root package (`.`), never `./...`. CI runs `make preflight` on
`ubuntu-24.04-arm` for every pull request (workflow `preflight`), and a nightly workflow repeats the
suite with `-race -count=3`. If preflight passes locally it passes there. A pull request that changes
only documents skips that job.

## Tests

Tests run against `httptest` and local WebSocket servers, with no network. A fix comes with a test
that fails without it; keep tests free of sleeps (the circuit breaker takes an injected clock) and
of leaked goroutines (`goleak_test.go` enforces it).

## Commits

Conventional Commits: `feat:`, `fix:`, `perf:`, `refactor:`, `docs:`, `deps:`, `revert:`, and
`chore:`, `ci:`, `test:`, `build:`, `style:` (hidden from the changelog). release-please reads them:
`feat:` makes a minor release, `fix:` a patch, and `feat!:` or a `BREAKING CHANGE:` footer a
breaking one (before 1.0 it raises the minor version). A commit that does not parse is left out of
the release notes, so keep the subject clean. Never tag or edit `CHANGELOG.md` by hand.

## Pull requests

Describe what changed, why, and how it was tested. Keep the repository free of hostnames, IP
addresses, operator paths and tokens, fixtures included. Update [docs/INDEX.md](docs/INDEX.md) when
you add or rename a document. By contributing you agree your work is licensed under Apache-2.0, see
[LICENSE](LICENSE).
