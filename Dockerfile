# ==========================================
# Stage 1: Build the Go Binaries Statically
# ==========================================
FROM golang:alpine AS builder

WORKDIR /app

# Install build essentials and SSL certs
RUN apk add --no-cache ca-certificates git

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build statically linked binaries for API server and Seeder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/seed ./cmd/seed

# ==========================================
# Stage 2: Minimal Production Runtime
# ==========================================
FROM alpine:3.20

WORKDIR /app

# Install ca-certificates for outbound HTTPS calls (LLMs & news sites) and curl for healthchecks
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root application user for container security
RUN adduser -D -g '' -u 10001 appuser

# Copy binaries from builder
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/seed /app/seed

# Run as non-root
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/app/api"]

