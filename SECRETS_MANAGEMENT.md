# Secrets Management Guide

## Overview

This project uses environment variables for secrets management. **Never commit actual secrets to the repository.**

## ✅ Safe Practices

### 1. Use `.env` Files (Local Development)

```bash
# Copy the example file
cp .env.example .env
cp backend/.env.example backend/.env

# Edit with your actual secrets
# These files are in .gitignore and will NEVER be committed
```

### 2. Use `.env.example` Files (Templates)

**Always commit:**
- ✅ `.env.example` files with placeholder values
- ✅ Documentation with `<PLACEHOLDER>` format

**Never commit:**
- ❌ `.env` files with real values
- ❌ Hardcoded secrets in code
- ❌ Secrets in commit messages

### 3. Example Format for Documentation

When writing documentation with secrets:

```bash
# ❌ BAD - Looks like a real secret (DO NOT DO THIS)
GITHUB_CLIENT_SECRET=abc123_fake_example_do_not_use

# ✅ GOOD - Clearly a placeholder
GITHUB_CLIENT_SECRET=<your_github_client_secret_here>
GITHUB_CLIENT_SECRET=your_github_client_secret_here
```

## 🔍 Checking for Secrets

### Local Scan (Before Commit)

```bash
# Install gitleaks
brew install gitleaks  # macOS
# or download from: https://github.com/gitleaks/gitleaks/releases

# Scan all files
gitleaks detect --redact -v

# Scan uncommitted changes only
gitleaks protect --redact -v
```

### CI/CD Scan

Every push and PR is automatically scanned by Gitleaks in GitHub Actions.

## 🚨 If You Accidentally Commit a Secret

### Immediate Actions

1. **Revoke the secret immediately** (GitHub token, API key, etc.)
2. **Remove from history** (requires force push):

```bash
# Use BFG Repo-Cleaner or git-filter-repo
# WARNING: This rewrites history

# Option 1: BFG (easier)
brew install bfg
bfg --replace-text secrets.txt  # Create secrets.txt with the secret
git reflog expire --expire=now --all
git gc --prune=now --aggressive

# Option 2: git-filter-repo
pip install git-filter-repo
git filter-repo --invert-paths --path-match 'file-with-secret.txt'
```

3. **Force push** (⚠️ Coordinate with team first):
```bash
git push --force-with-lease
```

4. **Notify the team** about the incident

### Prevention

- Enable **pre-commit hooks** (see below)
- Use **GitHub Secret Scanning** (enable in repo settings)
- Regular security audits

## 🔐 Pre-commit Hook (Recommended)

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash

# Run gitleaks on staged files
if command -v gitleaks &> /dev/null; then
    echo "🔍 Running gitleaks scan..."
    gitleaks protect --staged --redact
    if [ $? -ne 0 ]; then
        echo "❌ Gitleaks found secrets! Commit aborted."
        echo "Fix the issues or add to .gitleaksignore if false positive."
        exit 1
    fi
    echo "✅ No secrets detected"
fi
```

Make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

## 📝 Environment Variables by Service

### Backend (Go)

| Variable | Description | Example |
|----------|-------------|---------|
| `DB_PASSWORD` | PostgreSQL password | `<secure_password>` |
| `REDIS_PASSWORD` | Redis password | `<secure_password>` |
| `JWT_SECRET` | JWT signing key | `<random_64_char_string>` |
| `CLAUDE_API_KEY` | Anthropic API key | `sk-ant-...` |
| `OPENAI_API_KEY` | OpenAI API key | `sk-...` |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth secret | `<from_github_oauth_app>` |

### Frontend (Next.js)

| Variable | Description | Public? |
|----------|-------------|---------|
| `NEXT_PUBLIC_API_URL` | Backend API URL | ✅ Yes |
| `NEXT_PUBLIC_SITE_URL` | Site URL | ✅ Yes |

**Note:** Only variables prefixed with `NEXT_PUBLIC_` are exposed to the browser. Never put secrets in these!

## 🛡️ Production Secrets

Use your hosting provider's secrets management:

- **Vercel:** Environment Variables in project settings
- **Railway:** Environment Variables in service settings
- **AWS:** AWS Secrets Manager or Parameter Store
- **Docker:** Docker secrets or encrypted environment files

Never store production secrets in:
- Code repositories
- Unencrypted files
- CI/CD logs
- Public documentation

## 🔄 Rotating Secrets

Regular rotation schedule:

- **Critical secrets** (DB, API keys): Every 90 days
- **OAuth secrets**: Every 180 days
- **JWT secrets**: After any security incident

## ℹ️ False Positives

If Gitleaks detects secrets in documentation or test files, add them to `.gitleaksignore`:

```
# Example entries
docs/example.md:API_KEY=fake_key_for_example
test/fixtures/sample.json:password
```

See `.gitleaksignore` for current exclusions.

## 📚 Additional Resources

- [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)
- [Gitleaks Documentation](https://github.com/gitleaks/gitleaks)
- [GitHub Secret Scanning](https://docs.github.com/en/code-security/secret-scanning)
