# ==========================================
# Stage 1: Build the React + Vite Frontend
# ==========================================
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Install frontend dependencies with npm ci for clean, deterministic builds
COPY web/package.json web/package-lock.json ./
RUN npm ci

# Copy frontend source code and configurations
COPY web/ ./

# Build optimized production assets into /app/web/dist
RUN npm run build

# ==========================================
# Stage 2: Build the Go Binaries Statically
# ==========================================
FROM golang:alpine AS backend-builder

WORKDIR /app

# Install build essentials and SSL certs
RUN apk add --no-cache ca-certificates git

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy Go application source code
COPY . .

# Copy compiled React frontend assets from frontend-builder into web/dist
# This satisfies Go's `//go:embed all:dist` in web/web.go
COPY --from=frontend-builder /app/web/dist ./web/dist

# Build statically linked binaries for API server and Seeder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/seed ./cmd/seed

# ==========================================
# Stage 3: Minimal Production Runtime
# ==========================================
FROM alpine:3.20

WORKDIR /app

# Install ca-certificates for outbound HTTPS calls (LLMs & news sites), tzdata, and curl for healthchecks
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root application user for container security
RUN adduser -D -g '' -u 10001 appuser

# Copy binaries from backend-builder
COPY --from=backend-builder /app/bin/api /app/api
COPY --from=backend-builder /app/bin/seed /app/seed

# Run as non-root
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/app/api"]


