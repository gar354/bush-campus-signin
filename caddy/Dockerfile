FROM golang:latest as builder

WORKDIR /build

RUN go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest && \
  xcaddy build --with github.com/caddy-dns/azure

FROM caddy:latest

COPY --from=builder /build/caddy /usr/bin/caddy
