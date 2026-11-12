/*
 * /[locale]/analytics — le tableau de bord PUBLIC du site.
 *
 * QUOI : santé du projet (tests), filtre de période, 5 cartes KPI, visiteurs en direct, thèmes du CV,
 *   lettres générées, carte des clics (heatmap + légende).
 * POURQUOI cette forme : matière de la feuille du dossier vivant — titres serif, panneaux opaques
 *   sur la nuit, filets fins ; la seule couleur de la page est celle des DONNÉES (rampe de la
 *   heatmap, pastille d'état), accordée à la palette de la montagne (components/site-page/palette.ts).
 * Contraintes (CLAUDE.md › Analytics) : 5 cartes KPI SANS sous-titre ni delta fabriqué ; la heatmap
 *   garde sa légende. Les chiffres sont blendés côté backend (DemoMetrics) — le front n'invente rien.
 */
import { Metadata } from 'next';
import { socialMetadata } from '@/lib/seo/social';
import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';

import TestStatsCard from '@/components/analytics/TestStatsCard';
import RealtimeVisitors from '@/components/analytics/RealtimeVisitors';
import ThemeStats from '@/components/analytics/ThemeStats';
import LettersGenerated from '@/components/analytics/LettersGenerated';
import Heatmap from '@/components/analytics/Heatmap';
import DateFilter from '@/components/analytics/DateFilter';
import StatsOverview from '@/components/analytics/StatsOverview';
import { PageIntro } from '@/components/site-page/kit';

// Données en direct : jamais de rendu statique.
export const dynamic = 'force-dynamic';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'analytics' });
  // Le gabarit du layout ajoute déjà « | maicivy » : le titre seul suffit.
  return {
    title: t('meta.title'),
    description: t('meta.description'),
    ...socialMetadata({ locale, title: t('meta.title'), description: t('meta.description') }),
  };
}

export default async function AnalyticsPage({ params }: PageProps) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'analytics' });

  return (
    <div className="container max-w-6xl px-5 py-14 md:px-8 md:py-20">
      <PageIntro surtitre={t('eyebrow')} titre={t('title')} chapeau={t('subtitle')} className="mb-12" />

      {/* Tests & Qualité — santé projet : nombre réel de tests automatisés (cf. TestStatsCard).
          Le portrait dev (LOC/commits/tendances), lui, est sur le CV. */}
      <div className="mb-10">
        <TestStatsCard />
      </div>

      {/* Filtre de période — pilote les cartes KPI via ?period= (fallback à la hauteur du filtre). */}
      <div className="mb-6">
        <Suspense fallback={<div className="h-10 rounded-md bg-muted animate-pulse" />}>
          <DateFilter />
        </Suspense>
      </div>

      {/* Les 5 cartes KPI */}
      <div className="mb-6">
        <Suspense fallback={<StatsOverviewSkeleton />}>
          <StatsOverview />
        </Suspense>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Visiteurs en direct — pleine largeur */}
        <div className="lg:col-span-3">
          <Suspense fallback={<PanelSkeleton hauteur="h-24" />}>
            <RealtimeVisitors />
          </Suspense>
        </div>

        {/* Thèmes du CV — 2 colonnes sur grand écran */}
        <div className="lg:col-span-2">
          <Suspense fallback={<PanelSkeleton hauteur="h-64" />}>
            <ThemeStats />
          </Suspense>
        </div>

        {/* Lettres générées — 1 colonne sur grand écran */}
        <div className="lg:col-span-1">
          <Suspense fallback={<PanelSkeleton hauteur="h-64" />}>
            <LettersGenerated />
          </Suspense>
        </div>

        {/* Carte des clics — pleine largeur */}
        <div className="lg:col-span-3">
          <Suspense fallback={<PanelSkeleton hauteur="h-96" />}>
            <Heatmap />
          </Suspense>
        </div>
      </div>

      {/* Note de confidentialité — en pied, sobre. */}
      <p
        data-testid="analytics-privacy"
        className="mt-10 border-t border-border pt-6 text-center text-sm leading-relaxed text-muted-foreground"
      >
        {t('privacy')}
      </p>
    </div>
  );
}

// Gabarits de chargement : à la hauteur des panneaux chargés (pas de saut à l'arrivée des données).
function StatsOverviewSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-5">
      {[...Array(5)].map((_, i) => (
        <div key={i} className="animate-pulse rounded-lg border bg-card p-5 md:p-6">
          <div className="mb-3 h-5 w-1/2 rounded bg-muted" />
          <div className="h-9 w-3/4 rounded bg-muted" />
        </div>
      ))}
    </div>
  );
}

function PanelSkeleton({ hauteur }: { hauteur: string }) {
  return (
    <div className="animate-pulse rounded-lg border bg-card p-5 md:p-6">
      <div className={`${hauteur} rounded bg-muted`} />
    </div>
  );
}
