# syntax=docker/dockerfile:1

# build stage — context is frontend/public_site
FROM node:22-alpine AS builder

ARG VERSION

WORKDIR /build

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# final image: Caddy serving the static build
FROM caddy:2-alpine
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=builder /build/dist /srv

EXPOSE 8080
