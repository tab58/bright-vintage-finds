# syntax=docker/dockerfile:1

# build stage — context must be the monorepo root
FROM golang:1.25 AS builder

ARG VERSION
ENV VERSION=${VERSION}
ENV GOWORK=off

# fail if VERSION is not set
RUN if [ -z "$VERSION" ]; then \
  echo "Error: VERSION cannot be empty or unset"; \
  exit 1; \
  fi

WORKDIR /build

# Copy shared library and service module files for dependency caching
COPY environment/shared/golang/go.mod environment/shared/golang/go.sum ./environment/shared/golang/
COPY backend/main-api/go.mod backend/main-api/go.sum ./backend/main-api/

RUN --mount=type=cache,target=/go/pkg/mod \
  cd backend/main-api && go mod download

# Copy full source for shared lib and service
COPY environment/shared/golang/ ./environment/shared/golang/
COPY backend/main-api/ ./backend/main-api/

# Build the binary
WORKDIR /build/backend/main-api
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux \
  go build -ldflags "-X 'main.Version=${VERSION}'" -o main-api ./cmd/app

# generate clean, final image
FROM scratch
COPY --from=builder /build/backend/main-api/main-api .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 3000

ENTRYPOINT ["./main-api"]
