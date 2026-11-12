/*
 * /[locale]/architecture — comment ce site est construit : couches, stack, garde-fous.
 *
 * QUOI : une page de contenu dans la matière de la feuille du dossier vivant (serif, filets, tableaux
 *   à lignes fines), sans aucune couleur en dur : le schéma est monochrome, ses couches se
 *   distinguent par leur étiquette et leur position, pas par une teinte.
 * POURQUOI Server Component : la page était un Client Component qui rendait un bloc vide de 384 px
 *   tant que `mounted` était faux, puis la page entière — le pied de page remontait puis
 *   redescendait (CLS mesuré 0,34). Rien ici n'a besoin du navigateur : rendue côté serveur, la page
 *   arrive avec sa hauteur définitive.
 * COMMENT : libellés via next-intl (namespace `architecture`) ; les noms propres de technologies,
 *   identiques dans toutes les langues, vivent dans la constante TECH ci-dessous.
 */
import type { Metadata } from 'next';
import type { ReactNode } from 'react';
import { getTranslations } from 'next-intl/server';
import { socialMetadata } from '@/lib/seo/social';

import { Link } from '@/i18n/navigation';
import { Feuille, PageIntro, TitreSection, boutonOr, boutonSobre, etiquette } from '@/components/site-page/kit';
import testStats from '@/lib/test-stats.json';
import type { TestStats } from '@/lib/types';

// Stats de tests RÉELLES, régénérées à chaque commit (lib/test-stats.json) → jamais périmées.
const stats = testStats as TestStats;

const DEPOT_PUBLIC = 'https://github.com/AlexisTrouve/maicivy';

/*
 * Noms propres de technologies et de services. QUOI : tout ce qui s'affiche tel quel dans les cinq
 * langues (une marque ne se traduit pas). POURQUOI une constante : la règle i18n interdit le texte en
 * dur dans le JSX ; un nom propre n'est pas une phrase, mais il doit être identifiable comme tel et
 * modifiable en un seul endroit. Tout ce qui DÉCRIT (rôle, couche, légende) passe par next-intl.
 */
const TECH = {
  nextjs: 'Next.js 15',
  typescript: 'TypeScript',
  tailwind: 'Tailwind',
  cloudflare: 'Cloudflare',
  nginx: 'Nginx',
  frontdoor: 'Frontdoor',
  goFiber: 'Go + Fiber',
  websocket: 'WebSocket',
  postgres: 'PostgreSQL 16',
  redis: 'Redis 7',
  claude: 'Claude',
  gpt: 'GPT-5.6',
  wikipedia: 'Wikipedia',
  duckduckgo: 'DuckDuckGo',
  github: 'GitHub',
  // Noms de modules du backend (identifiants de code, pas des mots de la langue).
  gitstats: 'GitStats',
  demoMetrics: 'DemoMetrics',
} as const;

/*
 * Tableaux de la stack : la clé i18n du rôle + le nom propre. L'ordre est celui de la lecture
 * (du langage aux outils de test).
 */
const STACKS = [
  {
    id: 'backend',
    items: [
      ['go', 'Go 1.24'],
      ['fiber', 'Fiber'],
      ['gorm', 'GORM'],
      ['redis', 'Redis'],
      ['zerolog', 'zerolog'],
      ['chromedp', 'chromedp'],
      ['colly', 'Colly'],
      ['testify', 'testify'],
    ],
  },
  {
    id: 'frontend',
    items: [
      ['nextjs', 'Next.js 15'],
      ['typescript', 'TypeScript 5'],
      ['tailwind', 'Tailwind CSS'],
      ['shadcn', 'shadcn/ui'],
      ['framer', 'Framer Motion'],
      ['rhf', 'React Hook Form'],
      ['zod', 'Zod'],
      ['tests', 'Jest + Playwright'],
    ],
  },
  {
    id: 'ai',
    items: [
      ['claude', 'Claude (Anthropic)'],
      ['fallback', 'GPT-5.6'],
      ['scraper', 'Colly + API'],
    ],
  },
  {
    id: 'infra',
    items: [
      ['docker', 'Docker Compose'],
      ['postgres', 'PostgreSQL 16'],
      ['redis', 'Redis 7'],
      ['nginx', 'Nginx'],
      ['prometheus', 'Prometheus'],
      ['grafana', 'Grafana'],
      ['ci', 'Gitea Actions'],
    ],
  },
] as const;

const FEATURES = ['dynamicCV', 'aiLetters', 'scraper', 'analytics', 'accessGate', 'rateLimiting'] as const;
const SECURITY = ['injection', 'xss', 'rateLimit', 'headers', 'gdpr', 'cookies'] as const;

/*
 * Extrait de services/scraper.go (GetCompanyInfo), simplifié pour la lecture. C'est une CITATION de
 * code — identique dans toutes les langues, comme le code source lui-même.
 */
const EXTRAIT_SCRAPER = `func (s *CompanyScraper) GetCompanyInfo(ctx context.Context, companyName string) (*models.CompanyInfo, error) {
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

    // Source 2: GitHub
    wg.Add(1)
    go func() {
        defer wg.Done()
        repos, _ := s.fetchFromGitHub(ctx, companyName)
        mu.Lock()
        info.RecentNews = repos
        mu.Unlock()
    }()

    // Source 3: blog / newsroom
    wg.Add(1)
    go func() {
        defer wg.Done()
        news, _ := s.fetchRecentNews(ctx, companyName)
        mu.Lock()
        info.RecentNews += "\\n" + news
        mu.Unlock()
    }()

    wg.Wait()
    return info, nil
}`;

/*
 * Exemple de sortie réelle du scraper pour « Vercel » : des DONNÉES récupérées (anglais de
 * Wikipedia et de GitHub), pas de l'interface — elles s'affichent telles quelles dans toutes les langues.
 */
const EXEMPLE_ENTREE = 'Vercel';
const EXEMPLE_SORTIE: ReadonlyArray<readonly [string, string]> = [
  ['description', '"Vercel Inc. is an American cloud application company…"'],
  ['github_projects', '["next.js", "turborepo", "swr", …]'],
  ['recent_news', '"Zero-config backends on Vercel AI Cloud"'],
];

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'architecture' });
  return {
    title: t('meta.title'),
    description: t('meta.description'),
    ...socialMetadata({ locale, title: t('meta.title'), description: t('meta.description') }),
  };
}

/** Une case du schéma : nom en clair, légende facultative en dessous. */
function Noeud({ nom, legende }: { nom: string; legende?: string }) {
  return (
    <div className="rounded-md border border-border bg-background/70 px-3 py-2 text-center">
      <div className="text-sm font-medium text-foreground">{nom}</div>
      {legende && <div className="mt-0.5 text-xs leading-snug text-muted-foreground">{legende}</div>}
    </div>
  );
}

/** Une couche du schéma : étiquette à gauche (au-dessus en mobile), ses cases à droite. */
function Couche({ nom, children }: { nom: string; children: ReactNode }) {
  return (
    <div className="grid gap-3 md:grid-cols-[10rem_minmax(0,1fr)] md:items-center md:gap-6">
      <div className={etiquette}>{nom}</div>
      <div>{children}</div>
    </div>
  );
}

/** Liaison verticale entre deux couches : un filet, alignée sur la colonne des cases. */
function Liaison() {
  return (
    <div aria-hidden="true" className="grid md:grid-cols-[10rem_minmax(0,1fr)] md:gap-6">
      <div className="hidden md:block" />
      <div className="mx-auto h-6 w-px bg-border" />
    </div>
  );
}

export default async function ArchitecturePage({ params }: PageProps) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'architecture' });

  // Les chiffres : valeurs de tests LIVE (test-stats.json), le reste est l'inventaire du dépôt.
  const metriques: ReadonlyArray<readonly [string, string]> = [
    [t('metrics.passingTests'), stats.total.toLocaleString(locale)],
    [t('metrics.backendTests'), stats.backend.tests.toLocaleString(locale)],
    [t('metrics.frontendTests'), stats.frontend.tests.toLocaleString(locale)],
    [t('metrics.apiEndpoints'), '100+'],
    [t('metrics.goFiles'), '126'],
    [t('metrics.reactComponents'), '65'],
    [t('metrics.documentation'), '50k+'],
    [t('metrics.implementationGuides'), '20'],
  ];

  return (
    <div className="container max-w-6xl px-5 py-14 md:px-8 md:py-20">
      <PageIntro surtitre={t('eyebrow')} titre={t('title')} chapeau={t('subtitle')} />

      {/* Chiffres — des lignes de preuve, pas des tuiles colorées. */}
      <section aria-labelledby="arch-chiffres" className="mt-14">
        <TitreSection id="arch-chiffres" note={t('metricsNote')}>
          {t('metricsTitle')}
        </TitreSection>
        <dl className="grid grid-cols-2 gap-px overflow-hidden rounded-[10px] border border-border bg-border md:grid-cols-4">
          {metriques.map(([libelle, valeur]) => (
            // dt avant dd (ordre exigé par <dl>), la valeur remontée à l'œil par flex-col-reverse.
            <div key={libelle} className="flex flex-col-reverse bg-card px-4 py-5 md:px-5">
              <dt className="mt-1 text-sm text-muted-foreground">{libelle}</dt>
              <dd className="font-heading text-3xl font-medium tabular-nums text-foreground md:text-4xl">{valeur}</dd>
            </div>
          ))}
        </dl>
      </section>

      {/* Schéma système — la pièce maîtresse de la page : la seule feuille au filet d'or. */}
      <section aria-labelledby="arch-schema" className="mt-16">
        <TitreSection id="arch-schema" note={t('overviewNote')}>
          {t('overview')}
        </TitreSection>
        <Feuille signal data-testid="architecture-diagram">
          <div className="space-y-1">
            <Couche nom={t('layers.client')}>
              <div className="grid grid-cols-3 gap-2">
                <Noeud nom={TECH.nextjs} />
                <Noeud nom={TECH.typescript} />
                <Noeud nom={TECH.tailwind} />
              </div>
            </Couche>
            <Liaison />
            <Couche nom={t('layers.edge')}>
              <div className="grid gap-2 sm:grid-cols-3">
                <Noeud nom={TECH.cloudflare} legende={t('nodes.cdnWaf')} />
                <Noeud nom={TECH.nginx} legende={t('nodes.proxyTls')} />
                <Noeud nom={TECH.frontdoor} legende={t('nodes.frontdoor')} />
              </div>
            </Couche>
            <Liaison />
            <Couche nom={t('layers.backend')}>
              <div className="grid grid-cols-3 gap-2">
                <Noeud nom={TECH.goFiber} />
                <Noeud nom={t('nodes.middlewares')} />
                <Noeud nom={TECH.websocket} />
              </div>
              <div className="mt-2 grid grid-cols-3 gap-2 md:grid-cols-6">
                <Noeud nom={t('nodes.modules.cv')} />
                <Noeud nom={t('nodes.modules.letters')} />
                <Noeud nom={t('nodes.modules.analytics')} />
                <Noeud nom={t('nodes.modules.chat')} />
                <Noeud nom={t('nodes.modules.blog')} />
                <Noeud nom={TECH.gitstats} />
              </div>
            </Couche>
            <Liaison />
            <Couche nom={t('layers.services')}>
              <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
                <Noeud nom={t('nodes.services.ai')} />
                <Noeud nom={t('nodes.services.scraper')} />
                <Noeud nom={TECH.demoMetrics} />
                <Noeud nom={t('nodes.services.cvScoring')} />
                <Noeud nom={t('nodes.services.profileBuilder')} />
              </div>
            </Couche>
            <Liaison />
            <Couche nom={t('layers.data')}>
              <div className="grid gap-2 sm:grid-cols-3">
                <Noeud nom={TECH.postgres} legende={t('nodes.orm')} />
                <Noeud nom={TECH.redis} legende={t('nodes.cacheUses')} />
                <Noeud
                  nom={`${TECH.claude} · ${TECH.gpt}`}
                  legende={t('nodes.models')}
                />
              </div>
            </Couche>
          </div>
        </Feuille>
      </section>

      {/* La stack — quatre tableaux outil / rôle (matière de .proofs). */}
      <section aria-labelledby="arch-stack" className="mt-16">
        <TitreSection id="arch-stack">{t('stackTitle')}</TitreSection>
        <div className="grid gap-6 md:grid-cols-2">
          {STACKS.map((pile) => (
            <Feuille key={pile.id}>
              <h3 className="text-xl font-medium text-foreground">{t(`stacks.${pile.id}.title`)}</h3>
              <p className="mt-1 text-sm text-muted-foreground">{t(`stacks.${pile.id}.description`)}</p>
              <table className="mt-5 w-full border-collapse text-sm">
                <thead>
                  <tr className="border-b border-border text-left text-xs font-medium text-muted-foreground">
                    <th scope="col" className="pb-2 pr-4 font-medium">{t('stackColumns.tool')}</th>
                    <th scope="col" className="pb-2 font-medium">{t('stackColumns.role')}</th>
                  </tr>
                </thead>
                <tbody>
                  {pile.items.map(([cle, nom]) => (
                    <tr key={cle} className="border-b border-border/70 last:border-b-0">
                      <td className="whitespace-nowrap py-2.5 pr-4 align-top font-medium text-foreground">{nom}</td>
                      <td className="py-2.5 align-top text-muted-foreground">{t(`stackItems.${pile.id}.${cle}`)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Feuille>
          ))}
        </div>
      </section>

      {/* Le scraper multi-sources : entrée → sources parallèles → résultat agrégé. */}
      <section aria-labelledby="arch-scraper" className="mt-16">
        <TitreSection id="arch-scraper" note={t('scraperNote')}>
          {t('scraper')}
        </TitreSection>
        <Feuille>
          <div className="space-y-1">
            <Couche nom={t('scraperInput')}>
              <div className="flex">
                <span className="rounded-md border border-border bg-background/70 px-4 py-2 font-mono text-sm text-foreground">
                  &quot;{EXEMPLE_ENTREE}&quot;
                </span>
              </div>
            </Couche>
            <Liaison />
            <Couche nom={t('scraperParallel')}>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
                <Noeud nom={TECH.wikipedia} />
                <Noeud nom={TECH.duckduckgo} />
                <Noeud nom={t('scraperSources.website')} />
                <Noeud nom={TECH.github} />
                <Noeud nom={t('scraperSources.blogNews')} />
              </div>
            </Couche>
            <Liaison />
            <Couche nom={t('scraperResult')}>
              <dl className="space-y-2 font-mono text-xs leading-relaxed">
                {EXEMPLE_SORTIE.map(([champ, valeur]) => (
                  <div key={champ} className="rounded-md border border-border bg-background/70 px-3 py-2">
                    <dt className="inline text-muted-foreground">{champ}: </dt>
                    <dd className="inline break-words text-foreground">{valeur}</dd>
                  </div>
                ))}
              </dl>
            </Couche>
          </div>
        </Feuille>
      </section>

      {/* Fonctionnalités : six fiches, chacune avec ses techniques. */}
      <section aria-labelledby="arch-features" className="mt-16">
        <TitreSection id="arch-features">{t('features')}</TitreSection>
        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((cle) => (
            <Feuille key={cle} className="flex flex-col">
              <h3 className="text-lg font-medium leading-snug text-foreground">{t(`technicalFeatures.${cle}.title`)}</h3>
              <p className="mt-2 flex-1 text-sm leading-relaxed text-muted-foreground">
                {t(`technicalFeatures.${cle}.description`)}
              </p>
              <ul className="mt-4 flex flex-wrap gap-1.5">
                {(t.raw(`technicalFeatures.${cle}.tech`) as string[]).map((tech) => (
                  <li key={tech} className="rounded border border-border px-2 py-0.5 text-xs text-muted-foreground">
                    {tech}
                  </li>
                ))}
              </ul>
            </Feuille>
          ))}
        </div>
      </section>

      {/* Sécurité : une liste à glyphes (le ✓ en or, comme les puces de la feuille du dossier). */}
      <section aria-labelledby="arch-securite" className="mt-16">
        <TitreSection id="arch-securite">{t('security')}</TitreSection>
        <Feuille>
          <ul className="grid gap-x-10 gap-y-5 md:grid-cols-2">
            {SECURITY.map((cle) => (
              <li key={cle} className="flex gap-3">
                <span aria-hidden="true" className="mt-0.5 text-gold">✓</span>
                <div>
                  <div className="font-medium text-foreground">{t(`securityItems.${cle}.title`)}</div>
                  <div className="mt-0.5 text-sm text-muted-foreground">{t(`securityItems.${cle}.description`)}</div>
                </div>
              </li>
            ))}
          </ul>
        </Feuille>
      </section>

      {/* Extrait de code — le creux de la page (bg-background sous la carte), défilement horizontal. */}
      <section aria-labelledby="arch-code" className="mt-16">
        <TitreSection id="arch-code" note={t('codeExampleNote')}>
          {t('codeExample')}
        </TitreSection>
        <pre className="overflow-x-auto rounded-[10px] border border-border bg-background p-5 text-[13px] leading-relaxed text-foreground/85">
          <code>{EXTRAIT_SCRAPER}</code>
        </pre>
      </section>

      {/* Code source — l'action principale de la page (or) : lire le dépôt. */}
      <section aria-labelledby="arch-source" className="mt-16">
        <Feuille className="text-center">
          <h2 id="arch-source" className="text-2xl font-medium text-foreground">
            {t('sourceCode')}
          </h2>
          <p className="mx-auto mt-3 max-w-xl text-muted-foreground">{t('sourceCodeDesc')}</p>
          <div className="mt-7 flex flex-wrap justify-center gap-3">
            <a href={DEPOT_PUBLIC} target="_blank" rel="noopener noreferrer" className={boutonOr}>
              {t('viewOnGithub')}
              <span aria-hidden="true">↗</span>
            </a>
            <Link href="/cv" className={boutonSobre}>
              {t('backToCV')}
            </Link>
          </div>
        </Feuille>
      </section>
    </div>
  );
}
