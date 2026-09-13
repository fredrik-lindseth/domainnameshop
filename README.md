# Domainname.shop DNS module for Caddy

This module lets [Caddy](https://caddyserver.com/) perform ACME DNS-01 challenges
using [Domainname.shop / Domeneshop](https://domene.shop).
Its Caddy module ID is `dns.providers.domainnameshop`.

This repository is a fork of `github.com/caddy-dns/domainnameshop` and retains
that Go module path. The DNS client is supplied by
[`github.com/libdns/domainnameshop`](https://github.com/libdns/domainnameshop).
See [the dependency review](docs/review.md) for known DNS correctness issues in
the pinned client, including cleanup in zones with MX/SRV records. Passing
wrapper tests does not mean those upstream issues are fixed.

## Build

Use Go 1.25 or newer and [xcaddy](https://github.com/caddyserver/xcaddy).
From this checkout:

```sh
xcaddy build v2.10.2 --with github.com/caddy-dns/domainnameshop=.
./caddy list-modules
```

To build this fork directly:

```sh
xcaddy build v2.10.2 --with github.com/caddy-dns/domainnameshop=github.com/fredrik-lindseth/domainnameshop@main
```

Replace `main` with a reviewed commit for reproducible deployment.
The output of `list-modules` should include `dns.providers.domainnameshop`.

Alternatively, build the local checkout into the official Caddy image:

```sh
docker build --tag caddy-domainnameshop:local .
docker run --rm caddy-domainnameshop:local caddy list-modules
```

The Dockerfile pins Caddy 2.10.2; override it with
`--build-arg CADDY_VERSION=<version>` when updating and testing Caddy.

## Credentials

Generate an API token and secret in the
[Domeneshop control panel](https://www.domeneshop.no/admin?view=api).
The [API documentation](https://api.domeneshop.no/docs/#section/Authentication)
describes token/secret HTTP Basic authentication. These are API credentials,
not the account login password.

Provide `DOMAINNAMESHOP_API_TOKEN` and `DOMAINNAMESHOP_API_SECRET` to the Caddy
process through your service manager or container environment. Both are
required. Empty or unknown placeholders are rejected during provisioning.
Keep credentials out of source control and command-line arguments.

## Caddyfile

```caddyfile
example.com {
    respond "Hello World"
    tls {
        dns domainnameshop {
            api_token {env.DOMAINNAMESHOP_API_TOKEN}
            api_secret {env.DOMAINNAMESHOP_API_SECRET}
        }
        propagation_delay 60s
    }
}
```

The positional form is also supported:

```caddyfile
dns domainnameshop {env.DOMAINNAMESHOP_API_TOKEN} {env.DOMAINNAMESHOP_API_SECRET}
```

For a global default:

```caddyfile
{
    acme_dns domainnameshop {env.DOMAINNAMESHOP_API_TOKEN} {env.DOMAINNAMESHOP_API_SECRET}
}
```

The global example does not set a propagation delay. DNS updates can take time
to appear; the site example waits 60 seconds before Caddy checks propagation.
Adjust this for your deployment; it is not a guarantee of propagation time.

## JSON

Use this fragment in an ACME issuer's `challenges.dns` configuration:

```json
{
  "provider": {
    "name": "domainnameshop",
    "api_token": "{env.DOMAINNAMESHOP_API_TOKEN}",
    "api_secret": "{env.DOMAINNAMESHOP_API_SECRET}"
  },
  "propagation_delay": "60s"
}
```

See Caddy's [ACME DNS configuration](https://caddyserver.com/docs/json/apps/tls/automation/policies/issuers/acme/challenges/dns/)
for its place in a complete config. Plain strings are literal credentials;
environment variable names alone are not expanded.

## Development

```sh
go mod tidy -diff
go vet ./...
go test -race ./...
```

CI runs these checks, builds the local module into Caddy, and verifies module
registration. Tests cover registration, JSON decoding, Caddyfile parsing,
placeholder resolution, and credential validation without real DNS changes.
