# Security Incident Response Plan

**Project:** maicivy
**Version:** 1.0
**Last Updated:** 2025-12-08
**Classification:** Internal - Confidential

---

## 📋 Overview

This document outlines the security incident response procedures for the maicivy application. It defines roles, responsibilities, and step-by-step processes for detecting, responding to, and recovering from security incidents.

---

## 🎯 Objectives

1. **Detect** security incidents quickly
2. **Contain** the incident to prevent further damage
3. **Eradicate** the threat from the environment
4. **Recover** normal operations safely
5. **Learn** from the incident to prevent recurrence

---

## 👥 Incident Response Team

### Roles & Responsibilities

| Role | Responsibility | Contact |
|------|----------------|---------|
| **Incident Commander** | Overall coordination, decision-making | Alexi (alexi@maicivy.example.com) |
| **Technical Lead** | Technical investigation, remediation | DevOps Team |
| **Communications Lead** | Internal/external communications | PR/Marketing |
| **Legal Advisor** | Legal implications, compliance | Legal Team (if applicable) |
| **Security Analyst** | Log analysis, threat intelligence | Security Team |

### Escalation Path

```
Severity 1-2 (Critical/High)
└── Incident Commander (immediately)
    ├── Technical Lead
    ├── CTO/CEO
    └── Legal (if data breach)

Severity 3-4 (Medium/Low)
└── Technical Lead
    └── Incident Commander (within 24h)
```

---

## 🚨 Incident Severity Levels

| Level | Description | Response Time | Examples |
|-------|-------------|---------------|----------|
| **S1 - Critical** | System compromise, data breach | **< 15 minutes** | RCE, database breach, widespread service down |
| **S2 - High** | Significant security threat | **< 1 hour** | Successful SQL injection, authentication bypass |
| **S3 - Medium** | Security vulnerability exploited | **< 4 hours** | XSS exploitation, rate limit bypass |
| **S4 - Low** | Suspicious activity, no active exploit | **< 24 hours** | Failed login attempts, port scanning |

---

## 🔍 Detection & Identification

### Detection Methods

#### 1. Automated Monitoring
- **Prometheus Alerts**
  - High error rate (> 1% 5xx errors)
  - Unusual traffic patterns
  - Resource exhaustion (CPU, memory, disk)

- **Log Analysis**
  - Security event logs (zerolog)
  - Rate limit violations
  - Validation failures
  - Authentication failures

- **External Monitoring**
  - Uptime monitoring (UptimeRobot, Pingdom)
  - SSL certificate expiry
  - DNS changes

#### 2. Manual Detection
- User reports
- Security scan findings
- Threat intelligence alerts
- Responsible disclosure reports

### Incident Classification

**Ask these questions:**
1. Is there evidence of unauthorized access?
2. Has data been accessed, modified, or exfiltrated?
3. Are systems currently under attack?
4. What is the scope of the incident?
5. What is the potential impact?

---

## 📞 Incident Response Process

### Phase 1: Preparation

**Before an incident occurs:**

- ✅ Maintain incident response contacts list
- ✅ Ensure backups are current and tested
- ✅ Document system architecture and dependencies
- ✅ Keep incident response tools ready
- ✅ Conduct tabletop exercises (quarterly)

**Tools & Access:**
- Admin access to all systems
- Backup access (separate credentials)
- Forensics tools (Docker containers ready)
- Communication channels (Slack, email, phone)

---

### Phase 2: Detection & Analysis

**Initial Response (First 15 minutes)**

1. **Confirm the Incident**
   ```bash
   # Check system status
   docker-compose ps

   # Check logs for anomalies
   docker-compose logs --tail=1000 backend | grep -i "error\|security"

   # Check Prometheus metrics
   curl http://localhost:9090/api/v1/query?query=up
   ```

2. **Assess Severity**
   - Determine incident level (S1-S4)
   - Identify affected systems
   - Estimate impact scope

3. **Initial Containment (if S1/S2)**
   - Isolate affected systems
   - Block malicious IPs
   - Disable compromised accounts

4. **Notify Incident Response Team**
   ```markdown
   Subject: [SECURITY INCIDENT - S1] Brief Description

   Incident ID: INC-2025-001
   Severity: S1 - Critical
   Detected: 2025-12-08 14:30 UTC
   Affected Systems: Backend API, Database

   Summary: [Brief description]
   Current Status: [Investigating / Contained / Resolved]

   Next Steps:
   - [Action items]

   Incident Commander: Alexi
   ```

5. **Begin Investigation**
   ```bash
   # Collect logs
   mkdir -p /tmp/incident-$(date +%Y%m%d)
   docker-compose logs backend > /tmp/incident-$(date +%Y%m%d)/backend.log
   docker-compose logs postgres > /tmp/incident-$(date +%Y%m%d)/postgres.log

   # Check network connections
   docker-compose exec backend netstat -antp

   # Check for suspicious files
   docker-compose exec backend find / -type f -mtime -1

   # Database audit
   docker-compose exec postgres psql -U maicivy -d maicivy_db \
     -c "SELECT * FROM pg_stat_activity;"
   ```

---

### Phase 3: Containment

**Short-term Containment (Immediate)**

#### For SQL Injection / Database Compromise
```bash
# 1. Block attacker IP
# Add to nginx.conf or firewall
sudo iptables -A INPUT -s <ATTACKER_IP> -j DROP

# 2. Rotate database credentials
./scripts/rotate-secrets.sh db

# 3. Enable query logging
docker-compose exec postgres psql -U postgres -c \
  "ALTER SYSTEM SET log_statement = 'all';"
docker-compose restart postgres

# 4. Backup current database state (forensics)
docker-compose exec postgres pg_dump -U maicivy maicivy_db > \
  /tmp/incident-$(date +%Y%m%d)/db_backup.sql
```

#### For Authentication Bypass / Session Hijacking
```bash
# 1. Invalidate all sessions
docker-compose exec redis redis-cli FLUSHALL

# 2. Rotate session secret
./scripts/rotate-secrets.sh session

# 3. Restart backend
docker-compose restart backend

# 4. Force re-authentication
# (Users will need to log in again)
```

#### For SSRF / Scraper Compromise
```bash
# 1. Disable scraper temporarily
# Edit .env
echo "SCRAPER_ENABLED=false" >> .env
docker-compose restart backend

# 2. Review scraper logs
docker-compose logs backend | grep -i "scraper\|ssrf"

# 3. Update domain whitelist
# backend/internal/services/scraper.go
```

#### For Rate Limit Bypass
```bash
# 1. Enable stricter rate limits
docker-compose exec redis redis-cli SET ratelimit:emergency:enabled true

# 2. Ban abusive IPs
docker-compose exec redis redis-cli SET "ratelimit:ban:<IP>" "banned" EX 3600

# 3. Review rate limit logs
docker-compose logs backend | grep "rate_limit_exceeded"
```

**Long-term Containment**

- Patch vulnerabilities
- Update dependencies
- Harden configurations
- Implement additional monitoring

---

### Phase 4: Eradication

**Remove the Threat**

1. **Identify Root Cause**
   - Review logs thoroughly
   - Analyze attack vectors
   - Identify compromised components

2. **Remove Malicious Elements**
   ```bash
   # Remove malicious files (if any)
   docker-compose exec backend rm -f /path/to/malicious/file

   # Remove backdoor accounts (if any)
   docker-compose exec postgres psql -U maicivy -d maicivy_db \
     -c "DELETE FROM users WHERE created_at > 'INCIDENT_TIME';"

   # Remove malicious code
   git log --since="INCIDENT_TIME" --oneline
   git revert <COMMIT_HASH>
   ```

3. **Patch Vulnerabilities**
   ```bash
   # Update dependencies
   cd backend && go get -u all && go mod tidy
   cd frontend && npm audit fix

   # Rebuild Docker images
   docker-compose build --no-cache

   # Redeploy
   docker-compose up -d
   ```

4. **Verify Eradication**
   ```bash
   # Run security scans
   gosec ./backend/...
   npm audit --audit-level=high

   # Check for persistence
   docker-compose exec backend ps aux
   docker-compose exec backend crontab -l
   ```

---

### Phase 5: Recovery

**Restore Normal Operations**

1. **Restore from Backup (if necessary)**
   ```bash
   # Stop services
   docker-compose down

   # Restore database from clean backup
   cat /backups/db_backup_clean.sql | \
     docker-compose exec -T postgres psql -U maicivy maicivy_db

   # Restore Redis data (if needed)
   docker-compose exec redis redis-cli --pipe < /backups/redis_dump.rdb

   # Restart services
   docker-compose up -d
   ```

2. **Verify System Integrity**
   ```bash
   # Check all services are running
   docker-compose ps

   # Health checks
   curl http://localhost:8080/health
   curl http://localhost:3000/

   # Run smoke tests
   cd backend && go test ./...
   cd frontend && npm test
   ```

3. **Gradual Service Restoration**
   - Start with read-only mode
   - Monitor closely for 24-48 hours
   - Enable write operations
   - Full service restoration

4. **Enhanced Monitoring**
   ```bash
   # Enable debug logging temporarily
   docker-compose exec backend \
     sh -c 'echo "LOG_LEVEL=debug" >> .env'
   docker-compose restart backend

   # Monitor real-time logs
   docker-compose logs -f backend
   ```

---

### Phase 6: Post-Incident Activity

**Within 7 days of incident resolution:**

#### 1. Incident Report

**Template:**
```markdown
# Post-Incident Report: INC-2025-001

## Executive Summary
Brief overview of the incident, impact, and resolution.

## Incident Timeline
| Time (UTC) | Event |
|------------|-------|
| 14:30 | Incident detected |
| 14:35 | Incident Commander notified |
| 14:45 | Containment actions started |
| 16:00 | Threat eradicated |
| 18:00 | Services fully restored |

## Root Cause Analysis
Detailed analysis of how the incident occurred.

## Impact Assessment
- Systems affected: Backend API, Database
- Data compromised: None / Limited / Significant
- Downtime: 3 hours 30 minutes
- Users affected: ~500 active sessions

## Response Actions
- [Action 1]
- [Action 2]
- [Action 3]

## Lessons Learned
### What Went Well
- Quick detection (< 5 minutes)
- Effective containment
- Good communication

### What Needs Improvement
- Incident playbooks need update
- Backup restoration took too long
- Missing monitoring for X

## Recommendations
1. [Recommendation 1]
2. [Recommendation 2]
3. [Recommendation 3]

## Follow-up Actions
| Action | Owner | Due Date | Status |
|--------|-------|----------|--------|
| Update WAF rules | DevOps | 2025-12-15 | Pending |
| Improve monitoring | Security | 2025-12-20 | Pending |
| Security training | All | 2025-12-31 | Pending |
```

#### 2. Update Runbooks
- Document new attack patterns
- Update incident playbooks
- Improve detection rules

#### 3. Training & Awareness
- Team debriefing
- Security awareness training
- Tabletop exercises

#### 4. Process Improvements
- Update monitoring thresholds
- Improve backup procedures
- Enhance access controls

---

## 🔧 Incident Playbooks

### Playbook: Data Breach

**Trigger:** Evidence of unauthorized data access

**Actions:**
1. Confirm breach scope
2. Secure affected systems
3. Notify legal team immediately
4. Preserve evidence (logs, files)
5. Assess legal obligations (GDPR, etc.)
6. Prepare breach notification (if required)
7. Notify affected users (if PII compromised)

**Legal Requirements:**
- GDPR: 72 hours to notify authorities
- Document all actions for compliance

### Playbook: DDoS Attack

**Trigger:** Unusual traffic spike, service degradation

**Actions:**
1. Verify it's DDoS (not legitimate traffic)
2. Enable DDoS protection (Cloudflare, AWS Shield)
3. Block attacking IPs (rate limiting)
4. Scale infrastructure if needed
5. Contact ISP/hosting provider
6. Monitor attack duration

### Playbook: Ransomware

**Trigger:** Files encrypted, ransom note found

**Actions:**
1. **DO NOT** pay ransom
2. Isolate infected systems immediately
3. Power off (don't reboot) infected hosts
4. Restore from clean backups
5. Scan all systems for malware
6. Report to law enforcement
7. Investigate infection vector

---

## 📞 Emergency Contacts

### Internal Team
| Name | Role | Email | Phone | Availability |
|------|------|-------|-------|--------------|
| Alexi | Incident Commander | alexi@maicivy.example.com | +XX-XXX-XXX-XXXX | 24/7 |
| DevOps | Technical Lead | devops@maicivy.example.com | +XX-XXX-XXX-XXXX | 24/7 |

### External Services
| Service | Purpose | Contact | Account ID |
|---------|---------|---------|------------|
| OVH Support | Hosting | support.ovh.com | XXXXXXX |
| Cloudflare | CDN/DDoS | https://dash.cloudflare.com | XXXXXXX |
| Anthropic | Claude API | support@anthropic.com | XXXXXXX |

### Regulatory/Legal
| Entity | Purpose | Contact |
|--------|---------|---------|
| CNIL (France) | Data Protection Authority | https://www.cnil.fr/ |
| Law Enforcement | Cybercrime reporting | cybercrime@police.fr |

---

## 🛠️ Incident Response Tools

### Pre-installed Tools
```bash
# Docker forensics
docker run --rm -it alpine sh

# Network analysis
tcpdump, wireshark, netstat

# Log analysis
grep, awk, sed, jq

# File integrity
sha256sum, diff
```

### External Tools
- **Forensics:** Volatility, Autopsy
- **Malware Analysis:** VirusTotal, Hybrid Analysis
- **Threat Intelligence:** AlienVault OTX, Shodan
- **Communication:** Slack, Email, Phone

---

## 📚 References

- [NIST Computer Security Incident Handling Guide](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-61r2.pdf)
- [SANS Incident Response Process](https://www.sans.org/white-papers/33901/)
- [OWASP Incident Response Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Incident_Response_Cheat_Sheet.html)

---

**Approval:**
- Incident Commander: _________________ Date: _________
- CTO: _________________ Date: _________

**Next Review Date:** 2026-06-08
