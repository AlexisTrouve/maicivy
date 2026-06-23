'use client';

import { useTranslations, useLocale } from 'next-intl';
import { CheckCircle2 } from 'lucide-react';
import testStats from '@/lib/test-stats.json';
import { TestStats } from '@/lib/types';

// ============================================================================
// TestStatsCard — carte "Tests & Qualité" de la page /analytics.
// ----------------------------------------------------------------------------
// QUOI : nombre RÉEL de tests automatisés (backend Go + frontend jest) + statut tout-vert.
// POURQUOI ici (et pas sur le CV) : c'est un signal de SANTÉ projet → sa place est sur le dashboard
//        analytics. Le portrait dev (LOC/commits/tendances) vit, lui, sur le CV (cf. DevPortrait).
// COMMENT : chiffres figés dans lib/test-stats.json, régénérés par scripts/gen-test-stats.mjs depuis
//        les vraies suites → jamais inventés.
// ============================================================================
const stats = testStats as TestStats;

export default function TestStatsCard() {
  const t = useTranslations('analytics.testStats');
  const locale = useLocale();

  return (
    <div
      data-testid="test-stats-card"
      className="rounded-xl border bg-card p-5 shadow-sm"
    >
      {/* En-tête : titre + badge vert si tout passe */}
      <div className="mb-4 flex items-center gap-2">
        <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
        <h3 className="font-semibold">{t('title')}</h3>
        {stats.allGreen && (
          <span className="ml-auto rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-300">
            {t('allGreen')}
          </span>
        )}
      </div>

      {/* 3 chiffres : total, backend, frontend */}
      <div className="grid grid-cols-3 gap-3 text-center">
        <div>
          <div className="text-2xl font-bold">{stats.total.toLocaleString(locale)}</div>
          <div className="mt-1 text-xs text-muted-foreground">{t('total')}</div>
        </div>
        <div>
          <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">{stats.backend.tests.toLocaleString(locale)}</div>
          <div className="mt-1 text-xs text-muted-foreground">{t('backend')}</div>
        </div>
        <div>
          <div className="text-2xl font-bold text-purple-600 dark:text-purple-400">{stats.frontend.tests.toLocaleString(locale)}</div>
          <div className="mt-1 text-xs text-muted-foreground">{t('frontend', { suites: stats.frontend.suites })}</div>
        </div>
      </div>

      <p className="mt-3 text-right text-[11px] text-muted-foreground">{t('updated', { date: stats.generatedAt })}</p>
    </div>
  );
}
