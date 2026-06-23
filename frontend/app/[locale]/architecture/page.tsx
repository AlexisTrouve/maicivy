'use client';

import { useState, useEffect } from 'react';
import { Link } from '@/i18n/navigation';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useTranslations, useLocale } from 'next-intl';
import {
  Server,
  Globe,
  Cpu,
  Cloud,
  GitBranch,
  Shield,
  Zap,
  Code,
  Layers,
  ArrowRight,
  CheckCircle2,
  Terminal,
  FileCode2,
  Boxes,
  FlaskConical,
  Network,
  BookOpen,
  FileText,
  ExternalLink,
} from 'lucide-react';
import testStats from '@/lib/test-stats.json';
import { TestStats } from '@/lib/types';

// Stats de tests RÉELLES, auto-régénérées à chaque commit (cf. lib/test-stats.json) → jamais périmé
// (avant : "882 tests" figé alors qu'on en a 1000+).
const stats = testStats as TestStats;

// Style "verre" intégré à l'aurora du site (cohérent avec la home) : semi-transparent + flou + hover.
const glass = 'backdrop-blur-sm bg-card/80 border-border/60 hover:border-primary/40 transition-colors';

// Tech Stack Data
const backendStack = [
  { name: 'Go 1.24+', description: 'High-performance backend language', icon: '🐹' },
  { name: 'Fiber', description: 'Express-like web framework', icon: '⚡' },
  { name: 'GORM', description: 'ORM for PostgreSQL', icon: '🗄️' },
  { name: 'Redis', description: 'Cache, sessions, rate limiting', icon: '🔴' },
  { name: 'zerolog', description: 'Structured JSON logging', icon: '📝' },
  { name: 'chromedp', description: 'PDF generation', icon: '📄' },
  { name: 'Colly', description: 'Web scraping', icon: '🕷️' },
  { name: 'testify', description: 'Testing framework', icon: '✅' },
];

const frontendStack = [
  { name: 'Next.js 14', description: 'React framework (App Router)', icon: '▲' },
  { name: 'TypeScript 5.3', description: 'Type-safe JavaScript', icon: '📘' },
  { name: 'Tailwind CSS', description: 'Utility-first styling', icon: '🎨' },
  { name: 'shadcn/ui', description: 'Radix UI components', icon: '🧩' },
  { name: 'Framer Motion', description: 'Animations & transitions', icon: '🎬' },
  { name: 'React Hook Form', description: 'Form management', icon: '📋' },
  { name: 'Zod', description: 'Schema validation', icon: '🛡️' },
  { name: 'Jest + Playwright', description: 'Unit & E2E testing', icon: '🧪' },
];

const aiStack = [
  { name: 'Claude (Anthropic)', description: 'Primary AI provider', icon: '🤖' },
  { name: 'GPT-4o (OpenAI)', description: 'Fallback provider', icon: '🧠' },
  { name: 'Multi-source Scraper', description: 'Wikipedia, GitHub, News', icon: '🔍' },
];

const infraStack = [
  { name: 'Docker', description: 'Containerization', icon: '🐳' },
  { name: 'PostgreSQL 16', description: 'Primary database', icon: '🐘' },
  { name: 'Redis 7', description: 'In-memory cache', icon: '⚡' },
  { name: 'Nginx', description: 'Reverse proxy & SSL', icon: '🌐' },
  { name: 'Prometheus', description: 'Metrics collection', icon: '📊' },
  { name: 'Grafana', description: 'Public dashboards', icon: '📈' },
  { name: 'GitHub Actions', description: 'CI/CD pipeline', icon: '🔄' },
];

const getFeatures = (t: any) => {
  const features = t.raw('technicalFeatures');
  return [
    {
      title: features.dynamicCV.title,
      description: features.dynamicCV.description,
      tech: features.dynamicCV.tech,
    },
    {
      title: features.aiLetters.title,
      description: features.aiLetters.description,
      tech: features.aiLetters.tech,
    },
    {
      title: features.scraper.title,
      description: features.scraper.description,
      tech: features.scraper.tech,
    },
    {
      title: features.analytics.title,
      description: features.analytics.description,
      tech: features.analytics.tech,
    },
    {
      title: features.accessGate.title,
      description: features.accessGate.description,
      tech: features.accessGate.tech,
    },
    {
      title: features.rateLimiting.title,
      description: features.rateLimiting.description,
      tech: features.rateLimiting.tech,
    },
  ];
};

const getMetrics = (t: (key: string) => string, locale: string) => [
  { label: t('metrics.goFiles'), value: '126', icon: FileCode2, color: 'text-green-500' },
  { label: t('metrics.reactComponents'), value: '65', icon: Boxes, color: 'text-blue-500' },
  // Tests : LIVE depuis test-stats.json (jamais périmé).
  { label: t('metrics.backendTests'), value: stats.backend.tests.toLocaleString(locale), icon: FlaskConical, color: 'text-emerald-500' },
  { label: t('metrics.frontendTests'), value: stats.frontend.tests.toLocaleString(locale), icon: FlaskConical, color: 'text-cyan-500' },
  { label: t('metrics.passingTests'), value: stats.total.toLocaleString(locale), icon: CheckCircle2, color: 'text-emerald-500' },
  { label: t('metrics.apiEndpoints'), value: '100+', icon: Network, color: 'text-orange-500' },
  { label: t('metrics.documentation'), value: '50k+', icon: BookOpen, color: 'text-purple-500' },
  { label: t('metrics.implementationGuides'), value: '20', icon: FileText, color: 'text-pink-500' },
];

function TechBadge({ name, description, icon }: { name: string; description: string; icon: string }) {
  return (
    <div className="flex items-center gap-3 rounded-lg border bg-card p-3 transition-colors hover:bg-accent">
      <span className="text-2xl">{icon}</span>
      <div>
        <div className="font-medium">{name}</div>
        <div className="text-xs text-muted-foreground">{description}</div>
      </div>
    </div>
  );
}

function ArchitectureDiagram({ t }: { t: (key: string) => string }) {
  return (
    <div className="rounded-xl border bg-gradient-to-br from-slate-900 to-slate-800 p-6 text-white">
      <h3 className="mb-6 text-center text-xl font-bold">{t('systemDiagram')}</h3>

      <div className="space-y-4">
        {/* Client Layer */}
        <div className="rounded-lg border border-blue-500/50 bg-blue-500/10 p-4">
          <div className="mb-2 text-center text-sm font-semibold text-blue-400">{t('layers.client')}</div>
          <div className="flex justify-center gap-4">
            <div className="rounded bg-blue-600 px-4 py-2 text-sm">Next.js 14</div>
            <div className="rounded bg-blue-600 px-4 py-2 text-sm">TypeScript</div>
            <div className="rounded bg-blue-600 px-4 py-2 text-sm">Tailwind</div>
          </div>
        </div>

        {/* Arrow */}
        <div className="flex justify-center">
          <ArrowRight className="h-6 w-6 rotate-90 text-gray-500" />
        </div>

        {/* Edge & Sécurité — Cloudflare → Nginx → frontdoor (checkpoint sus / anti-abus). C'est la
            couche construite cette session, jusque-là absente du diagramme. */}
        <div className="rounded-lg border border-rose-500/50 bg-rose-500/10 p-4">
          <div className="mb-2 text-center text-sm font-semibold text-rose-400">{t('layers.edge')}</div>
          <div className="grid grid-cols-3 gap-2 text-center text-xs">
            <div className="rounded bg-rose-700 p-2">Cloudflare<div className="text-[10px] text-rose-200/70">CDN · WAF</div></div>
            <div className="rounded bg-rose-700 p-2">Nginx<div className="text-[10px] text-rose-200/70">reverse proxy · SSL</div></div>
            <div className="rounded bg-rose-700 p-2">Frontdoor<div className="text-[10px] text-rose-200/70">rate-limit · sus</div></div>
          </div>
        </div>

        {/* Arrow */}
        <div className="flex justify-center">
          <ArrowRight className="h-6 w-6 rotate-90 text-gray-500" />
        </div>

        {/* API Gateway */}
        <div className="rounded-lg border border-green-500/50 bg-green-500/10 p-4">
          <div className="mb-2 text-center text-sm font-semibold text-green-400">{t('layers.backend')}</div>
          <div className="grid grid-cols-3 gap-2 text-center text-xs">
            <div className="rounded bg-green-700 p-2">Go + Fiber</div>
            <div className="rounded bg-green-700 p-2">Middlewares</div>
            <div className="rounded bg-green-700 p-2">WebSocket</div>
          </div>
          <div className="mt-3 grid grid-cols-3 gap-2 text-center text-xs md:grid-cols-6">
            <div className="rounded bg-green-800 p-2">CV</div>
            <div className="rounded bg-green-800 p-2">Lettres</div>
            <div className="rounded bg-green-800 p-2">Analytics</div>
            <div className="rounded bg-green-800 p-2">Chat</div>
            <div className="rounded bg-green-800 p-2">Blog</div>
            <div className="rounded bg-green-800 p-2">GitStats</div>
          </div>
        </div>

        {/* Arrow */}
        <div className="flex justify-center">
          <ArrowRight className="h-6 w-6 rotate-90 text-gray-500" />
        </div>

        {/* Services Layer */}
        <div className="rounded-lg border border-purple-500/50 bg-purple-500/10 p-4">
          <div className="mb-2 text-center text-sm font-semibold text-purple-400">{t('layers.services')}</div>
          <div className="grid grid-cols-2 gap-2 text-center text-xs md:grid-cols-5">
            <div className="rounded bg-purple-700 p-2">AI (Claude/GPT)</div>
            <div className="rounded bg-purple-700 p-2">Scraper</div>
            <div className="rounded bg-purple-700 p-2">DemoMetrics</div>
            <div className="rounded bg-purple-700 p-2">CV Scoring</div>
            <div className="rounded bg-purple-700 p-2">Profile Builder</div>
          </div>
        </div>

        {/* Arrow */}
        <div className="flex justify-center">
          <ArrowRight className="h-6 w-6 rotate-90 text-gray-500" />
        </div>

        {/* Data Layer */}
        <div className="grid gap-4 md:grid-cols-3">
          <div className="rounded-lg border border-orange-500/50 bg-orange-500/10 p-4">
            <div className="mb-2 text-center text-sm font-semibold text-orange-400">{t('layers.database')}</div>
            <div className="text-center">
              <div className="rounded bg-orange-700 px-3 py-2 text-sm">PostgreSQL 16</div>
              <div className="mt-2 text-xs text-orange-300">GORM ORM</div>
            </div>
          </div>

          <div className="rounded-lg border border-red-500/50 bg-red-500/10 p-4">
            <div className="mb-2 text-center text-sm font-semibold text-red-400">{t('layers.cache')}</div>
            <div className="text-center">
              <div className="rounded bg-red-700 px-3 py-2 text-sm">Redis 7</div>
              <div className="mt-2 text-xs text-red-300">Sessions · Rate-limit · Queues</div>
            </div>
          </div>

          <div className="rounded-lg border border-cyan-500/50 bg-cyan-500/10 p-4">
            <div className="mb-2 text-center text-sm font-semibold text-cyan-400">{t('layers.aiProviders')}</div>
            <div className="space-y-1 text-center">
              <div className="rounded bg-cyan-700 px-3 py-1 text-sm">Claude API</div>
              <div className="rounded bg-cyan-800 px-3 py-1 text-sm">GPT-4o</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function ScraperDiagram({ t }: { t: (key: string) => string }) {
  return (
    <div className="rounded-xl border bg-gradient-to-br from-slate-900 to-slate-800 p-6 text-white">
      <h3 className="mb-6 text-center text-xl font-bold">{t('scraper')}</h3>

      <div className="space-y-4">
        {/* Input */}
        <div className="flex justify-center">
          <div className="rounded-lg border-2 border-yellow-500 bg-yellow-500/20 px-6 py-3">
            <span className="font-mono text-yellow-300">&quot;Vercel&quot;</span>
          </div>
        </div>

        {/* Arrow */}
        <div className="flex justify-center">
          <ArrowRight className="h-6 w-6 rotate-90 text-gray-500" />
        </div>

        {/* Parallel Sources */}
        <div className="rounded-lg border border-blue-500/50 bg-blue-500/10 p-4">
          <div className="mb-3 text-center text-sm font-semibold text-blue-400">
            {t('scraperParallel')}
          </div>
          <div className="grid grid-cols-2 gap-2 text-xs md:grid-cols-5">
            <div className="rounded bg-blue-700 p-2 text-center">
              <div className="text-lg">📚</div>
              <div>Wikipedia</div>
            </div>
            <div className="rounded bg-blue-700 p-2 text-center">
              <div className="text-lg">🦆</div>
              <div>DuckDuckGo</div>
            </div>
            <div className="rounded bg-blue-700 p-2 text-center">
              <div className="text-lg">🌐</div>
              <div>Website</div>
            </div>
            <div className="rounded bg-blue-700 p-2 text-center">
              <div className="text-lg">🐙</div>
              <div>GitHub</div>
            </div>
            <div className="rounded bg-blue-700 p-2 text-center">
              <div className="text-lg">📰</div>
              <div>Blog/News</div>
            </div>
          </div>
        </div>

        {/* Arrow */}
        <div className="flex justify-center">
          <ArrowRight className="h-6 w-6 rotate-90 text-gray-500" />
        </div>

        {/* Output */}
        <div className="rounded-lg border border-green-500/50 bg-green-500/10 p-4">
          <div className="mb-3 text-center text-sm font-semibold text-green-400">
            {t('scraperResult')}
          </div>
          <div className="space-y-2 font-mono text-xs">
            <div className="rounded bg-green-900/50 p-2">
              <span className="text-green-400">description:</span> &quot;Vercel Inc. is an American cloud application company...&quot;
            </div>
            <div className="rounded bg-green-900/50 p-2">
              <span className="text-green-400">github_projects:</span> [&quot;next.js&quot;, &quot;turborepo&quot;, &quot;swr&quot;...]
            </div>
            <div className="rounded bg-green-900/50 p-2">
              <span className="text-green-400">recent_news:</span> &quot;Zero-config backends on Vercel AI Cloud&quot;
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function ArchitecturePage() {
  const [mounted, setMounted] = useState(false);
  const t = useTranslations('architecture');
  const locale = useLocale();
  const metrics = getMetrics(t, locale);
  const features = getFeatures(t);

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) {
    return <div className="container py-12 animate-pulse"><div className="h-96"></div></div>;
  }

  return (
    <div className="container py-12">
      {/* Header */}
      <div className="mx-auto max-w-4xl text-center">
        <h1 className="font-heading text-4xl font-bold tracking-tight sm:text-5xl">
          <span className="bg-gradient-to-r from-primary via-blue-400 to-cyan-400 bg-clip-text text-transparent">
            {t('title')}
          </span>
        </h1>
        <p className="mt-4 text-lg text-muted-foreground">
          {t('subtitle')}
        </p>
      </div>

      {/* Metrics Overview */}
      <div className="mx-auto mt-12 max-w-6xl">
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
          {metrics.map((metric) => {
            const Icon = metric.icon;
            return (
              <Card key={metric.label} className={`text-center ${glass}`}>
                <CardContent className="pt-6">
                  <Icon className={`mx-auto mb-2 h-6 w-6 ${metric.color}`} />
                  <div className="text-3xl font-bold text-primary">{metric.value}</div>
                  <div className="mt-1 text-sm text-muted-foreground">{metric.label}</div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>

      {/* Architecture Diagram */}
      <div className="mx-auto mt-16 max-w-4xl">
        <h2 className="mb-6 text-2xl font-bold">
          <Layers className="mb-1 mr-2 inline h-6 w-6" />
          {t('overview')}
        </h2>
        <ArchitectureDiagram t={t} />
      </div>

      {/* Tech Stack Grid */}
      <div className="mx-auto mt-16 max-w-6xl">
        <h2 className="mb-8 text-2xl font-bold">
          <Code className="mb-1 mr-2 inline h-6 w-6" />
          {t('fullStack')}
        </h2>

        <div className="grid gap-8 md:grid-cols-2">
          {/* Backend */}
          <Card className={glass}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5 text-green-500" />
                {t('stacks.backend.title')}
              </CardTitle>
              <CardDescription>{t('stacks.backend.description')}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-2">
              {backendStack.map((tech) => (
                <TechBadge key={tech.name} {...tech} />
              ))}
            </CardContent>
          </Card>

          {/* Frontend */}
          <Card className={glass}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Globe className="h-5 w-5 text-blue-500" />
                {t('stacks.frontend.title')}
              </CardTitle>
              <CardDescription>{t('stacks.frontend.description')}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-2">
              {frontendStack.map((tech) => (
                <TechBadge key={tech.name} {...tech} />
              ))}
            </CardContent>
          </Card>

          {/* AI */}
          <Card className={glass}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Cpu className="h-5 w-5 text-purple-500" />
                {t('stacks.ai.title')}
              </CardTitle>
              <CardDescription>{t('stacks.ai.description')}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-2">
              {aiStack.map((tech) => (
                <TechBadge key={tech.name} {...tech} />
              ))}
            </CardContent>
          </Card>

          {/* Infrastructure */}
          <Card className={glass}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Cloud className="h-5 w-5 text-orange-500" />
                {t('stacks.infra.title')}
              </CardTitle>
              <CardDescription>{t('stacks.infra.description')}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-2">
              {infraStack.map((tech) => (
                <TechBadge key={tech.name} {...tech} />
              ))}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Scraper Diagram */}
      <div className="mx-auto mt-16 max-w-4xl">
        <h2 className="mb-6 text-2xl font-bold">
          <Zap className="mb-1 mr-2 inline h-6 w-6" />
          {t('scraper')}
        </h2>
        <ScraperDiagram t={t} />
      </div>

      {/* Features */}
      <div className="mx-auto mt-16 max-w-6xl">
        <h2 className="mb-8 text-2xl font-bold">
          <CheckCircle2 className="mb-1 mr-2 inline h-6 w-6" />
          {t('features')}
        </h2>

        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => (
            <Card key={feature.title} className={glass}>
              <CardHeader>
                <CardTitle className="text-lg">{feature.title}</CardTitle>
                <CardDescription>{feature.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  {feature.tech.map((techItem: string) => (
                    <span
                      key={techItem}
                      className="rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary"
                    >
                      {techItem}
                    </span>
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      {/* Security */}
      <div className="mx-auto mt-16 max-w-4xl">
        <h2 className="mb-6 text-2xl font-bold">
          <Shield className="mb-1 mr-2 inline h-6 w-6" />
          {t('security')}
        </h2>

        <Card className={glass}>
          <CardContent className="pt-6">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="flex items-start gap-3">
                <CheckCircle2 className="mt-0.5 h-5 w-5 text-green-500" />
                <div>
                  <div className="font-medium">{t('securityItems.injection.title')}</div>
                  <div className="text-sm text-muted-foreground">{t('securityItems.injection.description')}</div>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <CheckCircle2 className="mt-0.5 h-5 w-5 text-green-500" />
                <div>
                  <div className="font-medium">{t('securityItems.xss.title')}</div>
                  <div className="text-sm text-muted-foreground">{t('securityItems.xss.description')}</div>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <CheckCircle2 className="mt-0.5 h-5 w-5 text-green-500" />
                <div>
                  <div className="font-medium">{t('securityItems.rateLimit.title')}</div>
                  <div className="text-sm text-muted-foreground">{t('securityItems.rateLimit.description')}</div>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <CheckCircle2 className="mt-0.5 h-5 w-5 text-green-500" />
                <div>
                  <div className="font-medium">{t('securityItems.headers.title')}</div>
                  <div className="text-sm text-muted-foreground">{t('securityItems.headers.description')}</div>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <CheckCircle2 className="mt-0.5 h-5 w-5 text-green-500" />
                <div>
                  <div className="font-medium">{t('securityItems.gdpr.title')}</div>
                  <div className="text-sm text-muted-foreground">{t('securityItems.gdpr.description')}</div>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <CheckCircle2 className="mt-0.5 h-5 w-5 text-green-500" />
                <div>
                  <div className="font-medium">{t('securityItems.cookies.title')}</div>
                  <div className="text-sm text-muted-foreground">{t('securityItems.cookies.description')}</div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Code Example */}
      <div className="mx-auto mt-16 max-w-4xl">
        <h2 className="mb-6 text-2xl font-bold">
          <Terminal className="mb-1 mr-2 inline h-6 w-6" />
          {t('codeExample')}
        </h2>

        <div className="rounded-xl bg-slate-900 p-6 text-sm">
          <pre className="overflow-x-auto text-slate-300">
            <code>{`// GetCompanyInfo : point d'entrée principal - multi-sources pour résilience
func (s *CompanyScraper) GetCompanyInfo(ctx context.Context, companyName string) (*models.CompanyInfo, error) {
    var wg sync.WaitGroup
    var mu sync.Mutex

    info := &models.CompanyInfo{Name: companyName}

    // Source 1: Wikipedia
    wg.Add(1)
    go func() {
        defer wg.Done()
        wikiInfo, _ := s.fetchFromWikipedia(ctx, companyName)
        mu.Lock()
        if info.Description == "" && wikiInfo.Description != "" {
            info.Description = wikiInfo.Description
        }
        mu.Unlock()
    }()

    // Source 2: GitHub - projets open-source
    wg.Add(1)
    go func() {
        defer wg.Done()
        repos, _ := s.fetchFromGitHub(ctx, companyName)
        mu.Lock()
        info.RecentNews = repos // Projets actifs
        mu.Unlock()
    }()

    // Source 3: Blog/Newsroom
    wg.Add(1)
    go func() {
        defer wg.Done()
        news, _ := s.fetchRecentNews(ctx, companyName)
        mu.Lock()
        info.RecentNews += "\\n" + news
        mu.Unlock()
    }()

    wg.Wait() // Attendre toutes les sources
    return info, nil
}`}</code>
          </pre>
        </div>
      </div>

      {/* GitHub Link */}
      <div className="mx-auto mt-16 max-w-4xl text-center">
        <Card className={`border-dashed ${glass}`}>
          <CardContent className="pt-6">
            <GitBranch className="mx-auto h-12 w-12 text-muted-foreground" />
            <h3 className="mt-4 text-xl font-semibold">{t('sourceCode')}</h3>
            <p className="mt-2 text-muted-foreground">
              {t('sourceCodeDesc')}
            </p>
            <div className="mt-6 flex flex-wrap justify-center gap-4">
              <Button asChild>
                <a href="https://github.com/AlexisTrouve/maicivy" target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="mr-2 h-4 w-4" />
                  {t('viewOnGithub')}
                </a>
              </Button>
              <Button variant="outline" asChild>
                <Link href="/cv">
                  {t('backToCV')}
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
