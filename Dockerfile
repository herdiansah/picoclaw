# ============================================================
# Stage 1: Build the picoclaw binary (with CGO for sqlite3)
# ============================================================
FROM golang:1.26.0-alpine AS builder

# Install build tools and sqlite dev headers required by github.com/mattn/go-sqlite3
RUN apk add --no-cache git make build-base sqlite-dev

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build (enable CGO)
COPY . .
ENV CGO_ENABLED=1
RUN make build

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata sqlite curl

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:18790/health || exit 1

# Copy binary
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw

# Run onboard to create default config/workspace
RUN /usr/local/bin/picoclaw onboard

# Ensure working dir and data directory exist
WORKDIR /app
RUN mkdir -p /app/data && touch /app/data/history.db

# Copy config into image (tokens should be overridden via env vars)
COPY config/config.json /root/.picoclaw/config.json

ENTRYPOINT ["picoclaw"]
CMD ["gateway"]