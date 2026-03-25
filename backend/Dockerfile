# ============================================
# Stage 0: Headless Chrome 120 (stable en containers VPS)
# Alpine et Debian ont tous les deux Chromium 146 qui crashe sur kernels restreints.
# chromedp/headless-shell:120 est le binaire officiel testé pour headless en container.
# ============================================
FROM chromedp/headless-shell:123.0.6312.58 AS headless-chrome

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

# Install runtime dependencies (pas de chromium system — on utilise headless-shell 120)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    git \
    fonts-noto \
    fonts-noto-cjk \
    fonts-dejavu \
    fontconfig \
    libglib2.0-0 \
    libnss3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    && rm -rf /var/lib/apt/lists/*

# Copier headless-shell 120 depuis l'image officielle chromedp
COPY --from=headless-chrome /headless-shell /usr/bin/headless-shell

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
