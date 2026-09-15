# syntax=docker/dockerfile:1

# build stage — context is frontend/public_site
FROM oven/bun:1-alpine AS builder

ARG VERSION

WORKDIR /build

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .
RUN bun run build

# final image: Caddy serving the static build
FROM caddy:2-alpine
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=builder /build/dist /srv

EXPOSE 8080
