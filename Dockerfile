ARG CADDY_VERSION=2.10.2
FROM caddy:${CADDY_VERSION}-builder AS builder
ARG CADDY_VERSION
WORKDIR /src/domainnameshop
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN xcaddy build v${CADDY_VERSION} \
    --output /usr/bin/caddy \
    --with github.com/caddy-dns/domainnameshop=/src/domainnameshop

FROM caddy:${CADDY_VERSION}-alpine
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
