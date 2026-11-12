import { NextRequest } from 'next/server';
import { createTranslator, hasLocale } from 'next-intl';

import { defaultLocale, locales } from '@/i18n/config';
import { loadMessages } from '@/i18n/messages';
import type { ActivityStats } from '@/lib/types';

import { Eyebrow, GoldRule, TextColumn } from '../_brand/IdentityBlock';
import { renderPreview, siteDomain } from '../_brand/PreviewFrame';
import { ink } from '../_brand/palette';

export const runtime = 'edge';

/** Statistiques d'activité du backend ; `null` si indisponibles. */
async function getActivityStats(): Promise<ActivityStats | null> {
  try {
    const apiUrl = process.env.API_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    const response = await fetch(`${apiUrl}/api/v1/activity/stats`, {
      next: { revalidate: 900 },
    });
    if (!response.ok) return null;
    return await response.json();
  } catch {
    return null;
  }
}

/** Un chiffre en Cormorant, son libellé dessous : la figure domine, le libellé la précise. */
function Figure({ value, label }: { value: string; label: string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', marginRight: 64 }}>
      <div style={{ display: 'flex', fontFamily: 'Cormorant', fontSize: 84, lineHeight: 1, color: ink.cream }}>{value}</div>
      <div style={{ display: 'flex', marginTop: 8, fontSize: 20, color: ink.muted }}>{label}</div>
    </div>
  );
}

/**
 * Aperçu de lien de l'activité — `?locale=…`, 1200 × 630, même cadre « montagne peinte ».
 *
 * QUOI : nom et accroche de l'accueil, puis les commits des 30 derniers jours, les projets actifs
 *   et les langages dominants, lus à `GET /api/v1/activity/stats`.
 * POURQUOI : mêmes palette et typographie que tous les aperçus du site. Les chiffres de repli
 *   codés en dur (69 commits, 8 projets) ont disparu : un aperçu qui invente une activité quand le
 *   backend ne répond pas ment — sans statistiques, il montre l'identité seule.
 * COMMENT : libellés depuis messages/*.json (nav.brand, montagne.voice, metadata.preview), avec
 *   accord en nombre par ICU.
 */
export async function GET(request: NextRequest) {
  const requested = new URL(request.url).searchParams.get('locale');
  const locale = hasLocale(locales, requested) ? requested : defaultLocale;

  const messages = loadMessages(locale);
  const t = createTranslator({ locale, messages, namespace: 'metadata.preview' });
  const stats = await getActivityStats();
  const languages = stats?.top_languages?.slice(0, 4) ?? [];
  // Séparateur de milliers de la locale (« 5 322 » en fr, « 5,322 » en en) : un nombre brut à
  // quatre chiffres se lit mal à la taille où les messageries affichent l'aperçu.
  const number = new Intl.NumberFormat(locale);

  return renderPreview(
    <TextColumn domain={siteDomain()}>
      <div style={{ display: 'flex', fontFamily: 'Cormorant', fontSize: 80, lineHeight: 1, color: ink.cream }}>
        {messages.nav.brand}
      </div>
      <Eyebrow marginTop={20}>{messages.montagne.voice.tagline}</Eyebrow>
      {stats && (
        <div style={{ display: 'flex', flexDirection: 'column' }}>
          <GoldRule />
          <div style={{ display: 'flex' }}>
            <Figure value={number.format(stats.total_commits_30d)} label={t('commits30d', { count: stats.total_commits_30d })} />
            <Figure value={number.format(stats.active_projects)} label={t('activeProjects', { count: stats.active_projects })} />
          </div>
          {languages.length > 0 && (
            <div style={{ display: 'flex', marginTop: 30, fontSize: 21, letterSpacing: 1, color: ink.muted }}>
              {languages.join('  ·  ')}
            </div>
          )}
        </div>
      )}
    </TextColumn>,
  );
}
