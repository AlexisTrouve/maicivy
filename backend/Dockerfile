# ============================================
# Stage 1: Builder (Debian Bookworm / glibc)
# ============================================
FROM golang:1.24-bookworm AS builder

# Build arguments
ARG BUILD_DATE
ARG VCS_REF

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends git gcc && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Update go.sum with any new dependencies
RUN go mod tidy

# Build binary (CGO pour sqlite, glibc runtime)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VCS_REF} -X main.buildDate=${BUILD_DATE}" \
    -a -installsuffix cgo \
    -o maicivy-backend \
    ./cmd/main.go

# ============================================
# Stage 2: Runtime (Debian Bookworm)
# Alpine auto-update avait tiré Chromium 146 qui crashe sur kernels VPS.
# Debian Bookworm = Chromium ~120, stable en headless container.
# ============================================
FROM debian:bookworm-slim

# Install runtime dependencies including Chromium and fonts for PDF generation
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    git \
    chromium \
    fonts-noto \
    fonts-noto-cjk \
    fonts-dejavu \
    fontconfig \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1000 app && useradd -u 1000 -g app -m app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/maicivy-backend .

# Copy migrations (if needed)
COPY --from=builder /app/migrations ./migrations

# Copy templates (for PDF generation)
COPY --from=builder /app/templates ./templates

# Content markdown files are mounted via docker-compose volume

# Change ownership
RUN chown -R app:app /app

USER app

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./maicivy-backend"]
