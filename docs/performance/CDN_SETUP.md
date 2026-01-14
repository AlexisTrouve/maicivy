# CDN Setup for maicivy

This document explains how to configure a Content Delivery Network (CDN) for optimal performance.

## Why Use a CDN?

**Benefits:**
- **Faster loading times** - Assets served from geographically closer servers
- **Reduced bandwidth costs** - Offload traffic from origin server
- **Better availability** - CDN handles traffic spikes
- **Automatic optimization** - Image compression, minification, Brotli

**Recommended CDN:** Cloudflare (free tier available)

---

## Cloudflare Setup

### Step 1: Create Cloudflare Account

1. Sign up at https://www.cloudflare.com/
2. Add your domain (e.g., `maicivy.com`)
3. Update nameservers at your domain registrar

### Step 2: DNS Configuration

Point your domain to your server:

```
# A Records
maicivy.com         A       YOUR_SERVER_IP
www.maicivy.com     A       YOUR_SERVER_IP

# CNAME (optional subdomain)
api.maicivy.com     CNAME   maicivy.com
```

### Step 3: SSL/TLS Configuration

**Settings → SSL/TLS:**
- **Encryption mode:** Full (strict)
- **Always Use HTTPS:** On
- **Automatic HTTPS Rewrites:** On
- **Min TLS Version:** 1.2

### Step 4: Caching Configuration

**Settings → Caching:**

#### Browser Cache TTL
```
Respect Existing Headers
```

#### Caching Level
```
Standard (default)
```

#### Page Rules

Create page rules for optimal caching:

**Rule 1: Static Assets (Aggressive Caching)**
```
URL: *maicivy.com/_next/static/*
Settings:
  - Cache Level: Cache Everything
  - Edge Cache TTL: 1 year
  - Browser Cache TTL: 1 year
```

**Rule 2: Images (Moderate Caching)**
```
URL: *maicivy.com/images/*
Settings:
  - Cache Level: Cache Everything
  - Edge Cache TTL: 1 month
  - Browser Cache TTL: 1 week
```

**Rule 3: API Endpoints (No Caching)**
```
URL: *maicivy.com/api/*
Settings:
  - Cache Level: Bypass
```

**Rule 4: HTML Pages (Short Caching)**
```
URL: *maicivy.com/*
Settings:
  - Cache Level: Cache Everything
  - Edge Cache TTL: 2 hours
  - Browser Cache TTL: 30 minutes
```

### Step 5: Performance Optimizations

**Settings → Speed → Optimization:**

Enable the following:

- **Auto Minify:**
  - [x] JavaScript
  - [x] CSS
  - [x] HTML

- **Brotli:** On
- **Early Hints:** On
- **Rocket Loader:** Off (conflicts with React hydration)
- **Mirage:** Off (Next.js handles lazy loading)

### Step 6: Image Optimization

**Settings → Speed → Image Optimization:**

- **Polish:** Lossless
  - Automatically optimizes images (WebP/AVIF conversion)
  - Removes metadata
  - Reduces file size by 20-40%

- **Mirage:** Off (Next.js Image component handles this)

### Step 7: Bandwidth Alliance

If using compatible hosting (e.g., DigitalOcean, Vultr):
- Zero egress fees between your server and Cloudflare
- Significantly reduces bandwidth costs

---

## Cache Purging

### Purge Everything
```bash
curl -X POST "https://api.cloudflare.com/client/v4/zones/{zone_id}/purge_cache" \
  -H "Authorization: Bearer {api_token}" \
  -H "Content-Type: application/json" \
  --data '{"purge_everything":true}'
```

### Purge Specific URLs
```bash
curl -X POST "https://api.cloudflare.com/client/v4/zones/{zone_id}/purge_cache" \
  -H "Authorization: Bearer {api_token}" \
  -H "Content-Type: application/json" \
  --data '{"files":["https://maicivy.com/api/cv","https://maicivy.com/_next/static/css/app.css"]}'
```

### Purge by Tag (requires Enterprise)
```bash
curl -X POST "https://api.cloudflare.com/client/v4/zones/{zone_id}/purge_cache" \
  -H "Authorization: Bearer {api_token}" \
  -H "Content-Type: application/json" \
  --data '{"tags":["cv","letters"]}'
```

---

## Alternative CDNs

### Bunny CDN

**Pros:**
- Very affordable ($0.01/GB)
- Great performance
- Simple pricing

**Setup:**
1. Create Pull Zone at https://bunny.net/
2. Set origin URL: `https://maicivy.com`
3. Enable Optimizer (image optimization)
4. Update domain DNS to point to Bunny CDN URL

**Configuration:**
```
Pull Zone URL: maicivy.b-cdn.net
Origin URL: https://maicivy.com
Optimizer: Enabled
Perma-Cache: Enabled for static assets
```

### AWS CloudFront

**Pros:**
- Integrates with AWS services
- Advanced features (Lambda@Edge)
- Global coverage

**Setup:**
1. Create CloudFront distribution
2. Set origin: Your server IP or S3 bucket
3. Configure cache behaviors
4. Set up custom domain (Route 53)

**Cache Behaviors:**
```
# Static assets
Path Pattern: /_next/static/*
Min TTL: 31536000 (1 year)
Max TTL: 31536000
Compress: Yes

# API endpoints
Path Pattern: /api/*
Min TTL: 0
Max TTL: 0
Compress: Yes
```

### Vercel CDN (if deploying frontend separately)

If deploying Next.js frontend to Vercel:
- Automatic CDN configuration
- Edge functions
- Instant cache invalidation
- Zero configuration

---

## Cache Headers in Nginx

Even with CDN, set proper cache headers at origin:

```nginx
# /etc/nginx/sites-available/maicivy

server {
    listen 80;
    server_name maicivy.com;

    # Static assets
    location /_next/static/ {
        add_header Cache-Control "public, max-age=31536000, immutable";
    }

    # Images
    location /images/ {
        add_header Cache-Control "public, max-age=2592000";
    }

    # API (no cache)
    location /api/ {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        proxy_pass http://backend:3000;
    }

    # HTML pages
    location / {
        add_header Cache-Control "public, max-age=1800";
        proxy_pass http://frontend:3001;
    }
}
```

---

## Monitoring CDN Performance

### Cloudflare Analytics

**Dashboard → Analytics:**
- Total requests
- Bandwidth saved
- Cache hit ratio (target: > 80%)
- Threat analytics

### Custom Monitoring

Track CDN performance with custom headers:

```javascript
// frontend/lib/api.ts
fetch('https://maicivy.com/api/cv', {
  headers: {
    'CF-Cache-Status': 'HIT' // Cloudflare adds this
  }
}).then(res => {
  // Check if cached
  const cacheStatus = res.headers.get('CF-Cache-Status')
  if (cacheStatus === 'HIT') {
    console.log('Cache hit!')
  }
})
```

---

## Testing CDN

### Check Cache Status

```bash
# Check if asset is cached
curl -I https://maicivy.com/_next/static/chunks/main.js

# Look for these headers:
# CF-Cache-Status: HIT (Cloudflare)
# X-Cache: Hit from cloudfront (CloudFront)
# X-Bunny-Cache: HIT (Bunny CDN)
```

### Measure Performance Improvement

**Before CDN:**
```bash
curl -w "@curl-format.txt" -o /dev/null -s https://your-server-ip/
```

**After CDN:**
```bash
curl -w "@curl-format.txt" -o /dev/null -s https://maicivy.com/
```

**curl-format.txt:**
```
    time_namelookup:  %{time_namelookup}\n
       time_connect:  %{time_connect}\n
    time_appconnect:  %{time_appconnect}\n
   time_pretransfer:  %{time_pretransfer}\n
      time_redirect:  %{time_redirect}\n
 time_starttransfer:  %{time_starttransfer}\n
                    ----------\n
         time_total:  %{time_total}\n
```

---

## Cost Estimation

### Cloudflare Free Tier
- **Cost:** $0/month
- **Bandwidth:** Unlimited
- **Cache:** Unlimited
- **SSL:** Included
- **DDoS protection:** Included

### Bunny CDN
- **Cost:** ~$1-5/month (typical small site)
- **Bandwidth:** $0.01/GB
- **Storage:** $0.02/GB/month

### AWS CloudFront
- **Cost:** $0.085/GB (first 10TB)
- **Requests:** $0.01 per 10,000 HTTPS requests
- **Free tier:** 1TB data transfer out, 10M HTTP/HTTPS requests (12 months)

---

## Best Practices

1. **Set long TTLs for static assets** - 1 year for hashed files
2. **Use Cache-Control headers** - Let CDN know what to cache
3. **Purge cache on deploy** - Automate cache invalidation
4. **Monitor cache hit rate** - Target > 80%
5. **Enable compression** - Brotli > Gzip
6. **Use image optimization** - WebP/AVIF automatic conversion
7. **Avoid caching personalized content** - Use cookies to bypass cache

---

## Troubleshooting

### Cache Not Working

**Problem:** CF-Cache-Status: MISS

**Solutions:**
1. Check Cache-Control headers from origin
2. Verify page rules configuration
3. Check if cookies are being set (bypasses cache)
4. Ensure URLs are consistent (www vs non-www)

### Images Not Optimized

**Problem:** Images still serving as JPEG/PNG

**Solutions:**
1. Enable Polish in Cloudflare
2. Check if client supports WebP/AVIF (Accept header)
3. Verify image formats in Next.js config
4. Clear Cloudflare cache

### SSL Errors

**Problem:** ERR_SSL_VERSION_OR_CIPHER_MISMATCH

**Solutions:**
1. Set SSL mode to "Full (strict)"
2. Ensure origin has valid SSL certificate
3. Check TLS version (min 1.2)

---

## References

- [Cloudflare Cache Documentation](https://developers.cloudflare.com/cache/)
- [Bunny CDN Guide](https://docs.bunny.net/)
- [AWS CloudFront Documentation](https://docs.aws.amazon.com/cloudfront/)
- [HTTP Caching Best Practices](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching)

---

**Last Updated:** 2025-12-08
**Author:** Alexi
**Version:** 1.0
