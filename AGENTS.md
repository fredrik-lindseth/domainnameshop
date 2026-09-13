# domainnameshop

This repository is a Caddy DNS module wrapper around
`github.com/libdns/domainnameshop`. DNS HTTP operations, record conversion,
caching, and concurrency belong to that dependency; this repository owns
module registration, configuration parsing, and credential provisioning.

## Files and contracts

- `domainnameshop.go`: registers `dns.providers.domainnameshop`, parses
  Caddyfile credentials, and resolves Caddy placeholders before validation.
- `domainnameshop_test.go`: configuration and module regression tests.
- `Dockerfile`: builds Caddy from this checkout via a local xcaddy replacement.
- `docs/review.md`: identified limitations in the pinned libdns dependency.

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
real DNS or require credentials. Wrapper tests do not establish correctness of
the upstream provider. Report remaining dependency limitations explicitly.
If dependencies change, update `go.mod` and `go.sum` together and verify the
Caddy integration build, not just this package.

## Work visibility

When delegating, state the agent role/model and owned scope. Report concrete
progress at least once per minute during ongoing work. On completion, report
changed files or commit, checks run, and remaining blockers. Distinguish tested
local changes from committed or pushed changes.
