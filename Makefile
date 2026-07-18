.PHONY: build test lint preflight fmt vet golangci-lint clean

# Single-package library — all build/test/lint targets scope to "."
# (never "./..." for build/test, per repo convention).

build:
	GOWORK=off go build .

test:
	GOWORK=off go test -race -count=1 .

# lint runs errcheck (-blank) and staticcheck (SA4006, SA4008) via `go run`
# so no separate tool installation is required — CI is self-contained.
# Uses single-package "." — not "./...".
lint:
	GOWORK=off go run github.com/kisielk/errcheck@latest -blank .
	GOWORK=off go run honnef.co/go/tools/cmd/staticcheck@latest -checks SA4006,SA4008 .

# golangci-lint is kept for local use only — it is NOT part of preflight
# because it would need a separate install step in CI.
golangci-lint:
	GOWORK=off golangci-lint run .

# fmt fails the build if any file needs gofmt formatting.
fmt:
	@out=$$(GOWORK=off gofmt -l -s .); if [ -n "$$out" ]; then echo "$$out" >&2; exit 1; fi

vet:
	GOWORK=off go vet .

# preflight = gofmt + vet + lint + build + test — the CI gate.
# Run locally before pushing: `make preflight`.
preflight: fmt vet lint build test

clean:
	go clean -cache -testcache

# Follow-up (not blocking): WebSocket protocol conformance via the Autobahn
# Test Suite (https://github.com/crossbario/autobahn-testsuite). Run locally
# with `wstest -m fuzzingclient` against a test-server harness — not wired
# into preflight because it requires a docker/autobahn environment.
