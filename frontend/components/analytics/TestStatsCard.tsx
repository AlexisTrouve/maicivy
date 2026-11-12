'use client';

import { useTranslations, useLocale } from 'next-intl';
import testStats from '@/lib/test-stats.json';
import { TestStats } from '@/lib/types';

// ============================================================================
// TestStatsCard — carte "Tests & Qualité" de la page /analytics.
// ----------------------------------------------------------------------------
// QUOI : nombre RÉEL de tests automatisés (backend Go + frontend jest) + statut tout-vert.
// POURQUOI ici (et pas sur le CV) : c'est un signal de SANTÉ projet → sa place est sur le dashboard
//        analytics. Le portrait dev (LOC/commits/tendances) vit, lui, sur le CV (cf. DevPortrait).
// COMMENT : chiffres figés dans lib/test-stats.json, régénérés par scripts/gen-test-stats.mjs depuis
//        les vraies suites → jamais inventés. Matière de la feuille du dossier vivant : filet d'or en
//        tête (c'est la preuve la plus solide de la page), chiffres en serif, aucune couleur par
//        chiffre — le statut « tout vert » est porté par le glyphe ✓ et son libellé, pas par du vert.
// ============================================================================
const stats = testStats as TestStats;

export default function TestStatsCard() {
  const t = useTranslations('analytics.testStats');
  const locale = useLocale();

  const chiffres = [
    { valeur: stats.total, libelle: t('total') },
    { valeur: stats.backend.tests, libelle: t('backend') },
    { valeur: stats.frontend.tests, libelle: t('frontend', { suites: stats.frontend.suites }) },
  ];

  return (
    <div
      data-testid="test-stats-card"
      className="rounded-[10px] border border-border border-t-gold/60 bg-card p-5 md:p-6"
    >
      {/* En-tête : titre + statut (glyphe + libellé, lisible sans la couleur) */}
      <div className="mb-5 flex flex-wrap items-center gap-x-3 gap-y-2">
        <h2 className="text-xl font-medium text-foreground">{t('title')}</h2>
        {stats.allGreen && (
          <span className="ml-auto inline-flex items-center gap-1.5 rounded-full border border-border px-2.5 py-0.5 text-xs font-medium text-foreground">
            <span aria-hidden="true" className="text-gold">✓</span>
            {t('allGreen')}
          </span>
        )}
      </div>

      {/* 3 chiffres : total, backend, frontend */}
      <dl className="grid grid-cols-3 gap-3">
        {chiffres.map(({ valeur, libelle }) => (
          // dt avant dd (ordre exigé par <dl>) ; la valeur est remontée à l'œil par flex-col-reverse.
          <div key={libelle} className="flex flex-col-reverse">
            <dt className="mt-1 text-xs leading-snug text-muted-foreground">{libelle}</dt>
            <dd className="font-heading text-2xl font-medium tabular-nums text-foreground md:text-3xl">
              {valeur.toLocaleString(locale)}
            </dd>
          </div>
        ))}
      </dl>

      <p className="mt-4 text-right text-xs text-muted-foreground">{t('updated', { date: stats.generatedAt })}</p>
    </div>
  );
}
