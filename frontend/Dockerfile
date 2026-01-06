# ============================================================================
# MAICIVY Frontend - Next.js Application
# ============================================================================
# Multistage build for optimized image size
# ============================================================================

# Stage 1: Dependencies
FROM node:20-alpine AS deps

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Install dependencies (legacy-peer-deps for @react-three compatibility)
RUN npm ci --legacy-peer-deps

# Stage 2: Build
FROM node:20-alpine AS builder

WORKDIR /app

# Build arguments for Next.js
ARG NEXT_PUBLIC_API_URL=http://localhost:8080
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules

# Copy source code
COPY . .

# Build Next.js app
RUN npm run build

# Stage 3: Runtime
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
# API_URL for server-side (internal docker network)
ENV API_URL=http://maicivy-backend:8080

# Copy public files
COPY --from=builder /app/public ./public

# Copy .next build
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

# Use existing node user (already in node:alpine image)
USER node

# Expose port
EXPOSE 3000

ENV PORT=3000

# Health check
HEALTHCHECK --interval=15s --timeout=5s --retries=5 --start-period=45s \
    CMD wget --quiet --tries=1 --spider http://localhost:3000 || exit 1

# Run
CMD ["node", "server.js"]
