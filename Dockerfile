# ── Stage 1: Builder ──────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy dependency files first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary (statically linked, no CGO)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /app/server ./cmd/api/main.go

# ── Stage 2: Runtime (minimal image) ──────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Copy binary and timezone data
COPY --from=builder /app/server .
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy migration files (dijalankan manual via flyctl ssh atau migrate tool)
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["/app/server"]
