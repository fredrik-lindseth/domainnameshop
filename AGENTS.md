# domainnameshop

This repository contains a Caddy DNS module and its maintained local
`libdns` provider. It owns module registration, configuration parsing,
credential provisioning, DNS HTTP operations, record conversion, and
concurrency.

## Files and contracts

- `domainnameshop.go`: registers `dns.providers.domainnameshop`, parses
  Caddyfile credentials, and resolves Caddy placeholders before validation.
- `domainnameshop_test.go`: configuration and module regression tests.
- `provider/`: Domeneshop API client and libdns implementation; its tests
  cover DNS record conversion and identity contracts.
- `Dockerfile`: builds Caddy from this checkout via a local xcaddy replacement.
- `docs/review.md`: upstream limitations that motivated the local provider.

Keep the module path `github.com/caddy-dns/domainnameshop` for compatibility.
This GitHub fork is `fredrik-lindseth/domainnameshop`; use an explicit module
replacement when building the fork, as documented in README.md.

Credentials are `api_token` and `api_secret`. Support JSON, positional
Caddyfile arguments, and Caddyfile blocks. The zero-value wrapper must be safe
to configure. Reject missing credentials after placeholder expansion and never
include credential values in errors, tests, or logs.

## Verification

Use the Go version declared in `go.mod` or newer:

```sh
go mod tidy -diff
go vet ./...
go test -race ./...
docker build --tag caddy-domainnameshop:test .
docker run --rm caddy-domainnameshop:test caddy list-modules
```

Use mock HTTP responses for API regression tests; normal tests must not mutate
real DNS or require credentials. Report remaining provider limitations explicitly.
If dependencies change, update `go.mod` and `go.sum` together and verify the
Caddy integration build, not just this package.

## Work visibility

When delegating, state the agent role/model and owned scope. Report concrete
progress at least once per minute during ongoing work. On completion, report
changed files or commit, checks run, and remaining blockers. Distinguish tested
local changes from committed or pushed changes.
