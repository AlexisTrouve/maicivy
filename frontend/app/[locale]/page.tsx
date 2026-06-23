// Server Component — pas de 'use client', pas de useState/useEffect
// Animations via CSS custom (classes fade-in-up avec delays) au lieu de framer-motion
// Résultat : TBT réduit drastiquement, CLS éliminé (pas de skeleton mount)
import { Link } from '@/i18n/navigation';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { FileText, Sparkles, BarChart3, Layers, ArrowRight } from 'lucide-react';

// Person JSON-LD — données structurées schema.org pour que Google construise l'ENTITÉ "Alexis Trouvé"
// (Knowledge Panel / rich results) → sert le goal "qu'on me trouve". Identité STABLE → hardcodée
// (pas de fetch maiProFiles qui ajouterait de la latence au TTFB de la home). sameAs = profils
// officiels (Google les utilise pour relier l'entité). Sync avec maiProFiles /profile.
const PERSON_SCHEMA = {
  '@context': 'https://schema.org',
  '@type': 'Person',
  name: 'Alexis Trouvé',
  url: 'https://maicivy.etheryale.com',
  jobTitle: 'Full-Stack Engineer & AI Specialist',
  address: { '@type': 'PostalAddress', addressCountry: 'France' },
  sameAs: [
    'https://www.linkedin.com/in/alexis-trouve-432397a9/',
    'https://github.com/AlexisTrouve',
  ],
};

export default function HomePage() {
  const t = useTranslations('home');

  return (
    <div className="container py-12 md:py-24">
      {/* JSON-LD Person — rendu côté serveur dans le HTML, lu par Google pour l'entité d'Alexi. */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(PERSON_SCHEMA) }}
      />
      {/* ─── Hero section ─────────────────────────────────────────────── */}
      <div className="mx-auto max-w-4xl text-center">

        {/* Badge disponibilité freelance */}
        <div className="animate-fade-in-up mb-6 inline-flex items-center gap-2 rounded-full border border-primary/30 bg-primary/10 px-4 py-1.5 text-sm font-medium text-primary">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-green-400" />
          </span>
          {/* i18n : ces deux lignes étaient hardcodées en FR → fuite FR en mode /en. Passées par t(). */}
          {t('availability')}
        </div>

        {/* Identité */}
        <p className="animate-fade-in-up [animation-delay:100ms] mb-3 text-base font-medium text-muted-foreground tracking-widest uppercase">
          {t('identity')}
        </p>

        {/* Titre principal avec gradient */}
        <h1 className="animate-fade-in-up [animation-delay:200ms] font-heading text-4xl font-bold tracking-tight sm:text-5xl md:text-6xl">
          {t('title').split(' ').slice(0, 2).join(' ')}{' '}
          <span className="bg-gradient-to-r from-primary via-blue-400 to-cyan-400 bg-clip-text text-transparent">
            {t('title').split(' ').slice(2).join(' ')}
          </span>
        </h1>

        {/* Sous-titre */}
        <p className="animate-fade-in-up [animation-delay:350ms] mt-6 text-lg text-muted-foreground max-w-2xl mx-auto">
          {t('subtitle')}
        </p>

        {/* Stack technique — badges visuels avec backdrop-blur pour s'intégrer à l'aurora */}
        <div className="animate-fade-in-up [animation-delay:450ms] mt-6 flex flex-wrap justify-center gap-2">
          {['Go', 'Next.js', 'TypeScript', 'PostgreSQL', 'Redis', 'Docker'].map((tech) => (
            <span
              key={tech}
              className="inline-flex items-center rounded-md border border-border/60 bg-muted/40 px-2.5 py-1 text-xs font-medium text-muted-foreground backdrop-blur-sm"
            >
              {tech}
            </span>
          ))}
        </div>

        {/* CTAs */}
        <div className="animate-fade-in-up [animation-delay:550ms] mt-10 flex flex-wrap justify-center gap-4">
          <Button asChild size="lg" className="gap-2 shadow-lg shadow-primary/25">
            <Link href="/cv">
              <FileText className="h-4 w-4" />
              {t('cta.viewCV')}
            </Link>
          </Button>
          <Button asChild size="lg" variant="outline" className="gap-2 backdrop-blur-sm">
            <Link href="/letters">
              <Sparkles className="h-4 w-4" />
              {t('cta.generateLetter')}
            </Link>
          </Button>
        </div>
      </div>

      {/* ─── Feature cards ────────────────────────────────────────────── */}
      {/* Délai 0.7s pour apparaître après le hero — donne un effet de cascade */}
      <div className="animate-fade-in-up [animation-delay:700ms] mx-auto mt-24 grid max-w-5xl gap-6 md:grid-cols-2 lg:grid-cols-4">
        {/* Card CV — backdrop-blur-sm s'intègre à l'aurora derrière */}
        <Card className="backdrop-blur-sm bg-card/80 border-border/60 hover:border-primary/40 transition-colors duration-300">
          <CardHeader>
            <FileText className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.cv.title')}</CardTitle>
            <CardDescription>{t('features.cv.description')}</CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full gap-1">
              <Link href="/cv">
                {t('features.cv.action')}
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardContent>
        </Card>

        {/* Card Lettres de motivation */}
        <Card className="backdrop-blur-sm bg-card/80 border-border/60 hover:border-primary/40 transition-colors duration-300">
          <CardHeader>
            <Sparkles className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.letters.title')}</CardTitle>
            <CardDescription>{t('features.letters.description')}</CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full gap-1">
              <Link href="/letters">
                {t('features.letters.action')}
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardContent>
        </Card>

        {/* Card Analytics */}
        <Card className="backdrop-blur-sm bg-card/80 border-border/60 hover:border-primary/40 transition-colors duration-300">
          <CardHeader>
            <BarChart3 className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.analytics.title')}</CardTitle>
            <CardDescription>{t('features.analytics.description')}</CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full gap-1">
              <Link href="/analytics">
                {t('features.analytics.action')}
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardContent>
        </Card>

        {/* Card Architecture — bordure en pointillés pour signaler que c'est différent/expérimental */}
        <Card className="backdrop-blur-sm bg-card/80 border-border/60 hover:border-primary/40 transition-colors duration-300 border-dashed border-primary/50">
          <CardHeader>
            <Layers className="h-10 w-10 text-primary" />
            <CardTitle>{t('features.architecture.title')}</CardTitle>
            <CardDescription>{t('features.architecture.description')}</CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full gap-1">
              <Link href="/architecture">
                {t('features.architecture.action')}
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
