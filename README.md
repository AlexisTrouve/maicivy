# maicivy - My AI-Powered Interactive CV

<div align="center">

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Next.js](https://img.shields.io/badge/Next.js-14-black?style=for-the-badge&logo=next.js&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5.3-3178C6?style=for-the-badge&logo=typescript&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=for-the-badge&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis&logoColor=white)
![Claude](https://img.shields.io/badge/Claude_AI-Anthropic-orange?style=for-the-badge)

**An intelligent, adaptive CV platform with AI-powered cover letter generation**

🌐 **Live Demo:** [https://maicivy.etheryale.com](https://maicivy.etheryale.com)

[Features](#-features) | [Quick Start](#-quick-start) | [Tech Stack](#-tech-stack) | [API](#-api-endpoints) | [Documentation](#-documentation)

</div>

---

## What is maicivy?

**maicivy** transforms the traditional CV into an interactive, intelligent experience. It's both a personal portfolio showcasing full-stack expertise and a technical demonstration of modern development practices.

### The Signature Feature: AI Letter Generator

Enter any company name, and maicivy generates **two letters simultaneously**:

| **Motivation Letter** | **Anti-Motivation Letter** |
|----------------------|---------------------------|
| Professional, compelling, tailored | Humorous, satirical, creative |
| Highlights relevant experience | Parodies skills with wit |
| Perfect for applications | Perfect for laughs |

The AI researches the company in real-time using **multiple sources** (Wikipedia, GitHub, company blog) to create contextually relevant, personalized content.

---

## Features

### Dynamic Adaptive CV
- **5 Themes**: Backend, C++, Artistic, Full-Stack, DevOps
- **Intelligent Scoring**: Automatically ranks experiences and skills by relevance to selected theme
- **PDF Export**: Generate professional PDFs for any theme
- **Animated Timeline**: Framer Motion-powered experience visualization

### AI-Powered Letter Generation
- **Dual Output**: Motivation + Anti-motivation letters generated in parallel
- **Multi-Source Company Research**:
  - Wikipedia API (company description, industry)
  - GitHub API (open-source projects, what they're building)
  - Blog/Newsroom scraping (recent news and announcements)
  - DuckDuckGo Instant Answer API
  - Clearbit API (optional enrichment)
- **Providers**: Claude (Anthropic) primary, GPT-4o fallback
- **PDF Download**: Professional formatting for both letters

### Real-Time Analytics Dashboard
- Live visitor count (WebSocket)
- Theme popularity charts
- Letter generation statistics
- Interaction heatmap
- **Public dashboard** - transparency as a feature

### Smart Access Control
- **Access Gate**: AI features unlock after 3 visits
- **Profile Detection**: Immediate access for recruiters, CTOs, tech leads
- **Rate Limiting**: 5 generations/day, 2-minute cooldown

### GitHub Integration
- OAuth authentication
- Auto-import public repositories
- Automatic sync every 6 hours

---

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Claude API key (`ANTHROPIC_API_KEY`)

### One-Command Setup

```bash
# Clone the repository
git clone <REPO_URL>
cd maicivy

# Configure environment
cp .env.example .env
# Edit .env and add your ANTHROPIC_API_KEY

# Start everything
bash START.sh
```

### Access Points
| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| API Docs | http://localhost:8080/api/docs |
| Grafana | http://localhost:3001 |

### Stop
```bash
bash STOP.sh
```

---

## Tech Stack

### Backend
| Technology | Purpose |
|------------|---------|
| **Go 1.24+** | Language |
| **Fiber** | Web framework (Express-like, high performance) |
| **GORM** | ORM for PostgreSQL |
| **Redis** | Caching, sessions, rate limiting |
| **zerolog** | Structured logging |
| **chromedp** | PDF generation (headless Chrome) |
| **Colly** | Web scraping |
| **testify** | Testing framework |

### Frontend
| Technology | Purpose |
|------------|---------|
| **Next.js 14** | React framework (App Router) |
| **TypeScript 5.3** | Type safety |
| **Tailwind CSS** | Styling |
| **shadcn/ui** | UI components |
| **Framer Motion** | Animations |
| **React Hook Form + Zod** | Form validation |
| **Jest + Playwright** | Testing |

### AI Services
| Provider | Model | Use Case |
|----------|-------|----------|
| **Anthropic** | claude-sonnet-4 | Primary letter generation |
| **OpenAI** | gpt-4o | Fallback provider |

### Infrastructure
| Technology | Purpose |
|------------|---------|
| **Docker** | Containerization |
| **PostgreSQL 16** | Primary database |
| **Redis 7** | Cache & sessions |
| **Nginx** | Reverse proxy |
| **Prometheus + Grafana** | Monitoring |
| **GitHub Actions** | CI/CD |

---

## Project Structure

```
maicivy/
├── backend/                 # Go API Server
│   ├── cmd/main.go         # Entry point
│   ├── internal/
│   │   ├── api/            # HTTP handlers
│   │   ├── services/       # Business logic
│   │   │   ├── ai.go               # Claude/GPT integration
│   │   │   ├── scraper.go          # Multi-source company scraper
│   │   │   ├── letter_generator.go # Letter orchestration
│   │   │   ├── profile_builder.go  # User profile builder
│   │   │   └── cv_scoring.go       # Theme scoring algorithm
│   │   ├── middleware/     # CORS, tracking, rate limiting
│   │   ├── models/         # GORM models
│   │   └── workers/        # Background jobs
│   ├── migrations/         # SQL migrations
│   └── tests/              # 28 test files
│
├── frontend/               # Next.js Application
│   ├── app/               # App Router pages
│   │   ├── cv/            # CV page
│   │   ├── letters/       # Letter generator
│   │   └── analytics/     # Dashboard
│   ├── components/        # React components
│   │   ├── cv/            # CVThemeSelector, ExperienceTimeline...
│   │   ├── letters/       # LetterGenerator, LetterPreview...
│   │   └── analytics/     # RealtimeVisitors, ThemeStats...
│   └── __tests__/         # 228 test files
│
├── docs/                  # Comprehensive documentation
│   ├── PROJECT_SPEC.md    # Full specifications
│   └── implementation/    # 19 detailed guides
│
├── docker-compose.yml     # Full stack orchestration
├── START.sh              # Quick start script
└── STOP.sh               # Stop script
```

---

## API Endpoints

### CV
```http
GET  /api/v1/cv?theme=backend     # Get adaptive CV
GET  /api/v1/cv/themes            # List available themes
GET  /api/v1/cv/export?format=pdf # Export to PDF
GET  /api/v1/experiences          # All experiences
GET  /api/v1/skills               # All skills
GET  /api/v1/projects             # All projects
```

### Letters (AI-Powered)
```http
POST /api/v1/letters/generate     # Start async generation
     Body: { "company_name": "Vercel", "letter_type": "motivation" }
     Returns: { "job_id": "uuid" }

GET  /api/v1/letters/job/:id      # Poll job status
GET  /api/v1/letters/:id          # Get letter details
GET  /api/v1/letters/:id/pdf      # Download PDF
GET  /api/v1/letters/access/status    # Check access eligibility
GET  /api/v1/letters/ratelimit/status # Check rate limit
```

### Analytics
```http
GET  /api/v1/analytics/realtime   # Real-time stats
GET  /api/v1/analytics/stats      # Aggregated statistics
GET  /api/v1/analytics/themes     # Top CV themes
GET  /api/v1/analytics/heatmap    # Interaction heatmap
WS   /ws/analytics                # WebSocket real-time updates
```

---

## The Company Scraper

The multi-source scraper fetches real company data to personalize letters:

```
┌─────────────────┐
│  Company Name   │
│   "Vercel"      │
└────────┬────────┘
         │
         ▼
┌────────────────────────────────────────────────────┐
│              Parallel Data Fetching                │
├──────────┬──────────┬──────────┬──────────┬───────┤
│Wikipedia │DuckDuckGo│ Website  │  GitHub  │ Blog  │
│   API    │   API    │ Scraper  │   API    │Scraper│
└────┬─────┴────┬─────┴────┬─────┴────┬─────┴───┬───┘
     │          │          │          │         │
     ▼          ▼          ▼          ▼         ▼
┌─────────────────────────────────────────────────────┐
│                  Aggregated Result                  │
├─────────────────────────────────────────────────────┤
│ Description: "Vercel Inc. is an American cloud..."  │
│ Industry: Technology / Cloud Computing              │
│ GitHub Projects: next.js, turborepo, swr...        │
│ Recent News: "Zero-config backends on Vercel AI"   │
└─────────────────────────────────────────────────────┘
```

**Example output for Vercel:**
- **Wikipedia**: "Vercel Inc. is an American cloud application company. The company created and maintains the Next.js web development framework."
- **GitHub Projects**: academy-subscription-starter, vercel-deploy-claude-code-plugin, v0-starter-template
- **Recent News**: "Zero-config backends on Vercel AI Cloud", "You can just ship agents"

---

## Configuration

### Required Environment Variables

```bash
# Database
DB_USER=maicivy
DB_PASSWORD=your-secure-password
DB_NAME=maicivy_db
DB_HOST=postgres

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# AI (Required for letter generation)
ANTHROPIC_API_KEY=sk-ant-xxxxx  # Required
OPENAI_API_KEY=sk-xxxxx         # Optional fallback

# Server
SERVER_PORT=8080
SERVER_ENV=development

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Optional Features

```bash
# Enhanced company scraping
CLEARBIT_API_KEY=xxxxx

# GitHub integration
GITHUB_CLIENT_ID=xxxxx
GITHUB_CLIENT_SECRET=xxxxx

# Rate limiting
RATE_LIMIT_AI=5                 # Generations per day
AI_GENERATION_COOLDOWN=120      # Seconds between generations
```

---

## Development

### Manual Setup

**Backend:**
```bash
cd backend
go mod download
go run cmd/main.go
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

### Testing

**Backend (Go):**
```bash
cd backend
go test -v ./...           # Run all tests
go test -cover ./...       # With coverage
make test                  # Using Makefile
```

**Frontend (TypeScript):**
```bash
cd frontend
npm test                   # Run Jest tests
npm run test:e2e          # Run Playwright E2E
npm run test:coverage     # Coverage report
```

### Code Quality
- **Go**: gofmt, go vet, golangci-lint
- **TypeScript**: ESLint, Prettier, strict mode
- **Target Coverage**: Backend 80%+, Frontend 70%+

---

## Security

### OWASP Top 10 Compliance
- **Injection Prevention**: GORM ORM, parameterized queries
- **XSS Protection**: Input sanitization, bluemonday
- **Rate Limiting**: Redis-based sliding window
- **Security Headers**: CSP, X-Frame-Options, HSTS
- **GDPR**: IP hashing, no PII storage

### Privacy
- IPs are hashed (SHA256), never stored in plain text
- No personal data collection without consent
- Analytics are aggregated and anonymous

---

## Documentation

| Document | Description |
|----------|-------------|
| [CLAUDE.md](./CLAUDE.md) | Navigation guide for developers |
| [docs/PROJECT_SPEC.md](./docs/PROJECT_SPEC.md) | Complete specifications |
| [QUICKSTART.md](./QUICKSTART.md) | Quick start guide |
| [docs/implementation/](./docs/implementation/) | 19 detailed implementation guides |

### Implementation Guides
- `01_SETUP_INFRASTRUCTURE.md` - Docker, PostgreSQL, Redis
- `02_BACKEND_FOUNDATION.md` - Go, Fiber, GORM setup
- `08_BACKEND_AI_SERVICES.md` - Claude/GPT integration
- `10_FRONTEND_LETTERS.md` - Letter generator UI
- `17_SECURITY.md` - OWASP compliance
- ... and 14 more

---

## Metrics

| Metric | Value |
|--------|-------|
| Backend Go files | 100+ |
| Frontend components | 60+ |
| Backend tests | 28 files |
| Frontend tests | 228 files |
| Total tests | 882 passing |
| Documentation | ~10,000 lines |
| API endpoints | 30+ |

---

## Roadmap

- [x] Backend foundation (Go + Fiber + GORM)
- [x] Frontend foundation (Next.js 14 + TypeScript)
- [x] Database schema + migrations
- [x] Middleware system (tracking, rate limiting, CORS)
- [x] Adaptive CV with 5 themes
- [x] AI letter generator (dual output)
- [x] Multi-source company scraper (Wikipedia, GitHub, News)
- [x] Real-time analytics dashboard (WebSocket)
- [x] GitHub OAuth integration
- [x] Comprehensive testing (882 tests passing)
- [ ] Production deployment
- [ ] Multi-language support (FR/EN)
- [ ] AI chatbot for CV Q&A

---

## License

MIT License - See [LICENSE](./LICENSE) for details.

---

## Author

**Alexis Trouve**
Full-Stack Developer | VBA Migration Specialist | Automation Engineer

*This project itself is the CV.*

---

<div align="center">

**[Back to Top](#maicivy---my-ai-powered-interactive-cv)**

</div>
