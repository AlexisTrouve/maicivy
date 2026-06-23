# Security Policy

## Supported Versions

| Component | Version | Security Updates |
|-----------|---------|------------------|
| Backend (Go) | 1.21+ | ✅ Active |
| Frontend (Next.js) | 14.2.35+ | ✅ Active |
| Node.js | 20+ | ✅ Active |

## Security Updates

### 2026-01-20: Next.js DoS Vulnerabilities Fixed

**Fixed Vulnerabilities:**
- **GHSA-mwv6-3258-q52c** (CVE-2025-55184) - DoS with Server Components
  - Severity: HIGH (CVSS 7.5)
  - Status: ✅ Fixed in Next.js 14.2.35

- **GHSA-5j59-xgg2-r9c4** (CVE-2025-67779) - DoS with Server Components (incomplete fix follow-up)
  - Severity: HIGH (CVSS 7.5)
  - Status: ✅ Fixed in Next.js 14.2.35

**Impact:** These vulnerabilities allowed malicious HTTP requests to cause denial of service by hanging the server process and consuming CPU in React Server Components.

**Action Taken:** Updated Next.js from 14.0.4 → 14.2.35

### Development Dependencies

The following vulnerabilities are suppressed as they only affect development dependencies and do not impact production:

- **cookie** (GHSA-pxg6-pf52-xh8x) - In msw (test mocking library)
- **diff** (GHSA-73rr-hh4g-fpgx) - In ts-node/jest (build/test tools)
- **undici 5.28.0** - Pinned for Next.js compatibility, expires 2026-03-01

See `dependency-check-suppressions.xml` for details.

## Reporting a Vulnerability

If you discover a security vulnerability, please email:
- **Email:** [your-email@example.com]
- **Expected Response Time:** 48 hours

Please include:
1. Description of the vulnerability
2. Steps to reproduce
3. Potential impact
4. Suggested fix (if any)

## Security Scanning

Our CI/CD pipeline includes:

| Scanner | Purpose | Exclusions | Configuration |
|---------|---------|------------|---------------|
| **gosec** | Go security issues | `scripts/`, `test/` | `.github/workflows/security-scan.yml` |
| **govulncheck** | Go vulnerability DB | `scripts/` | Uses `go list` filter |
| **npm audit** | npm vulnerabilities | - | `package.json` |
| **Trivy** | Container scanning | - | Docker images |
| **OWASP Dependency Check** | Dependency CVEs | Dev dependencies | `dependency-check-suppressions.xml` |
| **Gitleaks** | Secrets detection | Documentation examples | `.gitleaksignore` |
| **CodeQL** | Static analysis | Auto-configured | GitHub native |

Scans run on every push and pull request.

### Scanner Configurations

#### gosec & govulncheck
The `backend/scripts/` directory is excluded because it contains development utilities with build tags, not production code. See `backend/scripts/README.md` for details.

#### OWASP Dependency Check
Development dependencies (msw, jest, ts-node) are suppressed as they don't affect production. See `dependency-check-suppressions.xml` for details.

### Gitleaks Configuration

Gitleaks scans for accidentally committed secrets. False positives are managed via `.gitleaksignore`:

**Ignored Files:** see `.gitleaksignore` for the current list of docs/test files with placeholder tokens.

**Adding New Ignores:**
If you have legitimate documentation with example secrets, add them to `.gitleaksignore`:
```
# Format: filepath:pattern
path/to/file.md:SECRET_KEY=example_value
```

## Security Best Practices

### Backend (Go)
- ✅ Input validation with custom validators
- ✅ SQL injection prevention with GORM
- ✅ Rate limiting (Redis-based)
- ✅ JWT authentication
- ✅ CORS configuration
- ✅ Security headers middleware
- ✅ Input sanitization

### Frontend (Next.js)
- ✅ Content Security Policy (CSP)
- ✅ XSS prevention
- ✅ CSRF protection
- ✅ Secure cookie configuration
- ✅ Environment variable protection

### Infrastructure
- ✅ Docker image scanning
- ✅ Minimal base images (Alpine)
- ✅ Non-root containers
- ✅ Secrets management via environment variables

## Update Schedule

- **Critical Vulnerabilities:** Immediate patch
- **High Severity:** Within 7 days
- **Medium Severity:** Within 30 days
- **Low Severity:** Next regular update cycle

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Next.js Security](https://nextjs.org/docs/app/building-your-application/configuring/security)
- [Go Security Policy](https://go.dev/security/)
