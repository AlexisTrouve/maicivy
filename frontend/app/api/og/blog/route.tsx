import { NextRequest } from 'next/server';
import { createTranslator, hasLocale } from 'next-intl';

import { defaultLocale, locales } from '@/i18n/config';
import { loadMessages } from '@/i18n/messages';
import type { BlogPost } from '@/lib/types';

import { Eyebrow, GoldRule, TextColumn } from '../_brand/IdentityBlock';
import { renderPreview, siteDomain } from '../_brand/PreviewFrame';
import { ink } from '../_brand/palette';

export const runtime = 'edge';

/** Article servi dans la langue du visiteur ; `null` si absent ou si le backend est injoignable. */
async function getBlogPost(slug: string, locale: string): Promise<BlogPost | null> {
  try {
    const apiUrl = process.env.API_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    const response = await fetch(`${apiUrl}/api/v1/blog/posts/${encodeURIComponent(slug)}?lang=${locale}`, {
      next: { revalidate: 3600 },
    });
    if (!response.ok) return null;
    return await response.json();
  } catch {
    // Un aperçu n'est pas la page : sans article, il retombe sur l'aperçu du blog plutôt que
    // de rendre une image cassée dans la messagerie qui l'a demandé.
    return null;
  }
}

/**
 * Aperçu de lien du blog — `?slug=…&locale=…`, 1200 × 630, même cadre « montagne peinte » que
 * l'accueil.
 *
 * QUOI : titre de l'article en Cormorant sur le ciel, sous une accroche « Alexis Trouvé · Blog » ;
 *   en dessous, projet, durée de lecture et sujets. Sans slug (ou article introuvable) : l'aperçu
 *   de la liste du blog, titré par sa méta.
 * POURQUOI : un article partagé est souvent la PREMIÈRE image du site qu'on voit ; elle doit être
 *   celle du site, pas un gabarit bleu générique.
 * COMMENT : libellés tirés de messages/*.json (blog, nav) — rien en dur. Le titre est borné à
 *   trois lignes (lineClamp) : au-delà il passerait sous le domaine.
 */
export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const requested = searchParams.get('locale');
  const locale = hasLocale(locales, requested) ? requested : defaultLocale;
  const slug = searchParams.get('slug');

  const messages = loadMessages(locale);
  const t = createTranslator({ locale, messages, namespace: 'blog' });
  const post = slug ? await getBlogPost(slug, locale) : null;

  const title = post?.title || t('meta.title');
  const details = post
    ? [post.project_name, post.reading_time_minutes ? t('post.readTime', { minutes: post.reading_time_minutes }) : '']
        .filter(Boolean)
        .join('  ·  ')
    : t('meta.description');
  const tags = post?.tags?.slice(0, 3) ?? [];

  return renderPreview(
    <TextColumn domain={`${siteDomain()}/blog`}>
      <Eyebrow>{`${messages.nav.brand} · ${t('title')}`}</Eyebrow>
      <div
        style={{
          display: 'block',
          marginTop: 26,
          fontFamily: 'Cormorant',
          fontSize: post ? 58 : 66,
          lineHeight: 1.08,
          maxWidth: 580,
          color: ink.cream,
          lineClamp: 3,
        }}
      >
        {title}
      </div>
      <GoldRule />
      <div style={{ display: 'block', fontSize: 22, lineHeight: 1.4, maxWidth: 560, color: ink.muted, lineClamp: 2 }}>
        {details}
      </div>
      {tags.length > 0 && (
        <div style={{ display: 'flex', marginTop: 14, fontSize: 19, color: ink.muted, opacity: 0.8 }}>
          {tags.map((tag) => `#${tag}`).join('   ')}
        </div>
      )}
    </TextColumn>,
  );
}
