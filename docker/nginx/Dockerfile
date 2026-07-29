FROM nginx:1.25-alpine

# Remove default config
RUN rm /etc/nginx/conf.d/default.conf

# Copy custom config
COPY nginx.conf /etc/nginx/nginx.conf
COPY conf.d/ /etc/nginx/conf.d/

# Create SSL directory (certificates will be mounted)
RUN mkdir -p /etc/nginx/ssl

# Health check
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:80/health || exit 1

EXPOSE 80 443

CMD ["nginx", "-g", "daemon off;"]
