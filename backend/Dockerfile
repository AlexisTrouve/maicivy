# ============================================
# Stage 1: Builder
# ============================================
FROM golang:1.24-alpine AS builder

# Build arguments
ARG BUILD_DATE
ARG VCS_REF

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Update go.sum with any new dependencies
RUN go mod tidy

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VCS_REF} -X main.buildDate=${BUILD_DATE}" \
    -a -installsuffix cgo \
    -o maicivy-backend \
    ./cmd/main.go

# ============================================
# Stage 2: Runtime
# ============================================
FROM alpine:latest

# Install runtime dependencies including Chromium for PDF generation
RUN apk add --no-cache ca-certificates tzdata curl chromium

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/maicivy-backend .

# Copy migrations (if needed)
COPY --from=builder /app/migrations ./migrations

# Copy templates (for PDF generation)
COPY --from=builder /app/templates ./templates

# Change ownership
RUN chown -R app:app /app

USER app

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./maicivy-backend"]
