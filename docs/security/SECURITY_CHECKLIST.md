# Pre-Production Security Checklist

**Project:** maicivy
**Version:** 1.0
**Last Updated:** 2025-12-08

---

## 📋 Overview

This checklist must be completed before deploying to production. Each item should be verified and signed off.

**Legend:**
- ✅ Completed
- ⚠️ Partial / Needs attention
- ❌ Not done
- N/A - Not applicable

---

## 🔐 Authentication & Authorization

- [ ] JWT secret is strong (minimum 64 characters)
- [ ] Session secret is unique and strong
- [ ] Password policy enforced (minimum 8 characters, complexity)
- [ ] Passwords hashed with bcrypt (cost >= 12)
- [ ] Rate limiting on authentication endpoints
- [ ] Account lockout after failed attempts
- [ ] Session timeout configured (< 24 hours)
- [ ] Secure cookie flags (HttpOnly, Secure, SameSite)
- [ ] CSRF protection implemented
- [ ] Multi-factor authentication available (future)

---

## 🗄️ Database Security

- [ ] Database password is strong and unique
- [ ] Database user has least privilege (not superuser)
- [ ] SSL/TLS enabled for database connections
- [ ] Prepared statements used (no string concatenation)
- [ ] Input validation before database queries
- [ ] Database backups encrypted
- [ ] Database backup tested (restore verified)
- [ ] Sensitive data encrypted at rest (future)
- [ ] Database logs do not contain sensitive data

---

## 🌐 Network & Infrastructure

- [ ] HTTPS enforced (HTTP→HTTPS redirect)
- [ ] TLS 1.2+ only (TLS 1.0/1.1 disabled)
- [ ] Strong cipher suites configured
- [ ] HSTS header enabled (max-age >= 1 year)
- [ ] Firewall rules configured (allow only necessary ports)
- [ ] SSH key-based authentication only (no passwords)
- [ ] Services not exposed unnecessarily (e.g., Redis, PostgreSQL)
- [ ] Internal services use separate network (Docker network)
- [ ] DDoS protection enabled (Cloudflare, AWS Shield)

---

## 🛡️ Input Validation & Sanitization

- [ ] All inputs validated on backend (never trust client)
- [ ] Input validation on frontend (UX, not security)
- [ ] Whitelist validation used (not blacklist)
- [ ] Maximum input lengths enforced
- [ ] HTML sanitization applied (bluemonday)
- [ ] SQL injection protection (parameterized queries)
- [ ] XSS protection (input escaping)
- [ ] SSRF protection (URL validation, domain whitelist)
- [ ] File upload validation (if applicable)
- [ ] Request size limits enforced (< 10MB)

---

## 🔒 Security Headers

- [ ] `X-Frame-Options: DENY`
- [ ] `X-Content-Type-Options: nosniff`
- [ ] `X-XSS-Protection: 1; mode=block`
- [ ] `Strict-Transport-Security` (HSTS)
- [ ] `Content-Security-Policy` configured
- [ ] `Referrer-Policy: strict-origin-when-cross-origin`
- [ ] `Permissions-Policy` configured
- [ ] No sensitive data in response headers

---

## 🚦 Rate Limiting & Access Control

- [ ] Global rate limiting (100 req/min per IP)
- [ ] AI endpoint rate limiting (5/day per session)
- [ ] Rate limit headers sent (X-RateLimit-*)
- [ ] Ban mechanism for abusive IPs
- [ ] Access gate for AI features (3 visits minimum)
- [ ] Rate limit bypass attempts logged
- [ ] Whitelist for trusted IPs (if needed)

---

## 🔑 Secrets Management

- [ ] No secrets in code (environment variables only)
- [ ] `.env` in `.gitignore`
- [ ] `.env.example` provided (no real secrets)
- [ ] Production secrets stored securely (GitHub Secrets, Vault)
- [ ] API keys rotated regularly (quarterly)
- [ ] Database passwords rotated
- [ ] Secrets backup encrypted
- [ ] Access to secrets limited (least privilege)

---

## 📝 Logging & Monitoring

- [ ] Structured logging enabled (JSON in production)
- [ ] Security events logged (rate limit, validation failures)
- [ ] No sensitive data in logs (passwords, tokens, PII redacted)
- [ ] Log retention policy defined (6-12 months)
- [ ] Centralized logging configured (optional: Loki, ELK)
- [ ] Monitoring alerts configured (Prometheus + Alertmanager)
- [ ] Alert on 5xx errors (> 1% threshold)
- [ ] Alert on rate limit abuse (> 10 violations/hour)
- [ ] Alert on security events
- [ ] Health check endpoint available (`/health`)

---

## 🐳 Docker & Containers

- [ ] Docker images scanned for vulnerabilities (Trivy)
- [ ] Base images from official sources only
- [ ] Images run as non-root user
- [ ] No secrets in Docker images
- [ ] Multi-stage builds used (minimal final image)
- [ ] Docker Content Trust enabled (signed images - future)
- [ ] Resource limits configured (CPU, memory)
- [ ] Container logs centralized

---

## 🔄 CI/CD Security

- [ ] GitHub Actions workflows secure (no hardcoded secrets)
- [ ] Security scans in CI/CD (gosec, npm audit, Trivy)
- [ ] Build fails on high/critical vulnerabilities
- [ ] Code review required before merge
- [ ] Branch protection enabled (main branch)
- [ ] Automated dependency updates (Dependabot)
- [ ] Deploy keys have minimal permissions
- [ ] Manual approval for production deploys

---

## 📦 Dependencies

- [ ] All dependencies up to date
- [ ] No known vulnerabilities (gosec, npm audit)
- [ ] Dependencies from trusted sources only
- [ ] Dependency versions pinned
- [ ] Automated dependency scanning (weekly)
- [ ] Update strategy documented

---

## 🧪 Testing

- [ ] Unit tests for security functions (80% coverage)
- [ ] Integration tests for authentication/authorization
- [ ] E2E tests for critical paths
- [ ] Security tests (XSS, SQL injection attempts)
- [ ] Rate limiting tests
- [ ] Penetration testing completed
- [ ] No critical/high vulnerabilities found

---

## 📄 Documentation

- [ ] Security policy documented (SECURITY.md)
- [ ] Incident response plan documented
- [ ] Runbooks for common security scenarios
- [ ] API documentation (endpoints, authentication)
- [ ] Deployment documentation
- [ ] Backup & restore procedures documented
- [ ] Secrets rotation procedures documented

---

## 🌍 CORS & API Security

- [ ] CORS whitelist configured (no wildcard `*`)
- [ ] CORS credentials flag set appropriately
- [ ] API versioning implemented (future)
- [ ] API rate limiting per endpoint
- [ ] API authentication required (future admin endpoints)
- [ ] API error messages generic (no sensitive info)

---

## 🔍 Vulnerability Management

- [ ] Vulnerability scanning automated (weekly)
- [ ] Vulnerability remediation SLA defined
  - Critical: 24 hours
  - High: 7 days
  - Medium: 30 days
  - Low: 90 days
- [ ] Responsible disclosure policy published (SECURITY.md)
- [ ] Bug bounty program considered (future)

---

## 📊 Compliance & Privacy

- [ ] Privacy policy published
- [ ] Data retention policy defined
- [ ] User data anonymization (where required)
- [ ] GDPR compliance reviewed (if applicable)
- [ ] Cookie consent implemented (if applicable)
- [ ] Data breach notification procedures documented

---

## 🔐 Backup & Recovery

- [ ] Automated daily backups
- [ ] Backup encryption enabled
- [ ] Backup restoration tested
- [ ] Off-site backup storage
- [ ] Recovery Time Objective (RTO) defined
- [ ] Recovery Point Objective (RPO) defined
- [ ] Disaster recovery plan documented

---

## 🚀 Production Deployment

- [ ] Staging environment mirrors production
- [ ] Smoke tests pass in staging
- [ ] Load testing completed
- [ ] Performance benchmarks met
- [ ] Rollback procedure tested
- [ ] Zero-downtime deployment strategy
- [ ] DNS configured correctly
- [ ] SSL certificates valid and auto-renewing
- [ ] Monitoring enabled before launch

---

## 📞 Incident Response

- [ ] Incident response team identified
- [ ] Incident response plan documented
- [ ] Emergency contacts list maintained
- [ ] Incident communication plan defined
- [ ] Forensics tools ready
- [ ] Tabletop exercise conducted

---

## ✅ Final Checks

- [ ] All above items completed
- [ ] Security review conducted
- [ ] Stakeholder sign-off obtained
- [ ] Go-live date scheduled
- [ ] Post-launch monitoring plan ready

---

## 📝 Sign-off

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Developer | Alexi | _____________ | _______ |
| Security | _____________ | _____________ | _______ |
| DevOps | _____________ | _____________ | _______ |
| CTO | _____________ | _____________ | _______ |

---

**Deployment Approval:** ☐ APPROVED ☐ DENIED ☐ CONDITIONAL

**Conditions (if conditional):**
_____________________________________________________________________
_____________________________________________________________________
_____________________________________________________________________

**Next Security Review:** 2026-03-08
