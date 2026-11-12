import { Metadata } from 'next';
import Image from 'next/image';
import Link from 'next/link';
import { notFound } from 'next/navigation';
import { getTranslations } from 'next-intl/server';

import { portfolioApi, portfolioImageUrl, type PortfolioPage } from '@/lib/portfolio-api';
import { TitreSection } from '@/components/site-page/kit';
import { socialMetadata } from '@/lib/seo/social';

// Espacement entre les sections de la fiche (titres serif via TitreSection), comme celui des
// sections de la feuille du dossier vivant.
const SECTION = 'mt-16';

export const dynamic = 'force-dynamic';

type PageProps = {
  params: Promise<{ locale: string; id: string }>;
};

// Rend la page, `null` si le projet est ABSENT de la curation, et relance toute autre erreur.
//
// POURQUOI la distinction : auparavant, toute erreur devenait `null` puis `notFound()`. Le 2026-09-27,
// maiProFiles saturé → backend 502 → le front affichait « 404 » sur des projets qui existent, pour les
// visiteurs comme pour Google (un 404 fait sortir une page de l'index, une erreur serveur non).
// COMMENT : seul un 404 du backend (« absent de la curation », sentinelle ErrPortfolioAbsent côté Go)
// mène à notFound(). Une panne (502, timeout, réseau) est relancée : elle remonte à la frontière
// d'erreur `[locale]/error.tsx`, rendue avec un statut 5xx — temporaire aux yeux d'un crawler.
async function getPage(id: string, locale: string): Promise<PortfolioPage | null> {
  try {
    return await portfolioApi.getPage(id, locale);
  } catch (error) {
    if ((error as { statusCode?: number })?.statusCode === 404) return null;
    throw error;
  }
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { locale, id } = await params;
  const projet = await getPage(id, locale);
  if (!projet) {
    const t = await getTranslations({ locale, namespace: 'portfolio' });
    return { title: t('notFound') };
  }

  // La description vient du premier paragraphe du récit, débarrassé de son balisage : c'est la
  // phrase écrite pour accrocher un lecteur, donc exactement ce qu'on veut dans un aperçu.
  const resume = (projet.paragraphs?.[0] || '').replace(/<[^>]+>/g, '').slice(0, 200);
  const image = projet.image?.url ? portfolioImageUrl(projet.image.url) : undefined;

  return {
    title: `${projet.name} — Alexis Trouvé`,
    description: resume,
    // Le visuel du projet s'il en a un, sinon l'aperçu du site — jamais un aperçu sans image.
    ...socialMetadata({
      locale,
      title: projet.name,
      description: resume,
      type: 'article',
      image: image ? { url: image, alt: projet.image.alt || projet.name } : undefined,
    }),
  };
}

export default async function PortfolioProjectPage({ params }: PageProps) {
  const { locale, id } = await params;
  const t = await getTranslations({ locale, namespace: 'portfolio' });
  const projet = await getPage(id, locale);

  if (!projet) notFound();

  const statut = projet.status
    ? t.has(`status.${projet.status}`)
      ? t(`status.${projet.status}`)
      : projet.status
    : '';

  return (
    <article className="container max-w-4xl px-5 py-14 md:px-8 md:py-20" data-testid="portfolio-project">
      <Link
        href={`/${locale}/portfolio`}
        className="inline-flex min-h-[44px] items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
        data-testid="portfolio-back"
      >
        <span aria-hidden="true">←</span> {t('back')}
      </Link>

      <header className="mt-4">
        <h1 className="text-4xl font-medium leading-[1.08] tracking-tight text-foreground md:text-6xl" data-testid="portfolio-title">
          {projet.name}
        </h1>
        {/* Catégorie + statut. Le statut « en production » est une preuve : il porte le point d'or
            (signal), mais le texte reste ivoire. */}
        <div className="mt-4 flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
          {projet.category && <span>{projet.category}</span>}
          {statut && (
            <span className="inline-flex items-center gap-2 rounded-full border border-border px-3 py-0.5 text-foreground">
              {projet.status === 'production' && <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-gold" />}
              {statut}
            </span>
          )}
        </div>
      </header>

      {projet.image?.url && (
        // QUOI : image de tête, via next/image (variante AVIF/WebP redimensionnée par /_next/image).
        // POURQUOI `priority` : au-dessus de la ligne de flottaison, c'est l'élément LCP de la page.
        // COMMENT : les dimensions d'origine ne sont pas connues — width/height ne servent qu'au
        // ratio réservé avant chargement ; `h-auto w-full` rend l'image à son ratio réel.
        // Largeur affichée : conteneur de l'article (≤ 896 px).
        <Image
          src={portfolioImageUrl(projet.image.url)}
          alt={projet.image.alt || projet.name}
          width={1600}
          height={1000}
          priority
          sizes="(min-width: 960px) 896px, 100vw"
          data-testid="portfolio-hero-image"
          className="mt-10 h-auto w-full rounded-[10px] border border-border object-cover"
        />
      )}

      {projet.paragraphs?.length > 0 && (
        <section className={SECTION} data-testid="portfolio-pitch">
          <TitreSection>{t('pitch')}</TitreSection>
          <div className="max-w-[68ch] space-y-5 text-base leading-relaxed text-foreground/85 md:text-lg [&_strong]:font-semibold [&_strong]:text-foreground">
            {projet.paragraphs.map((paragraphe, i) => (
              // Le récit est curé dans book.yaml et porte son propre balisage (<strong> sur les
              // chiffres qui comptent). Contenu écrit par nous, pas saisi par un visiteur.
              <p key={i} dangerouslySetInnerHTML={{ __html: paragraphe }} />
            ))}
          </div>
        </section>
      )}

      {projet.video?.url && (
        <section className={SECTION} data-testid="portfolio-video">
          <TitreSection>{t('demo')}</TitreSection>
          {/* QUOI : la démo vidéo du projet, juste après le récit — c'est la preuve de ce qu'il raconte.
              POURQUOI preload="metadata" et JAMAIS "none" : mesuré le 20/10 (Chrome headless ET fenêtré,
              page vierge), avec "none" un clic sur la vidéo ne lance RIEN (readyState 0, aucun événement
              play) ; avec "metadata" elle démarre. Le fichier étant en faststart, "metadata" ne tire que
              l'en-tête (quelques Ko), pas les ~10 Mo. Poster pour l'attente, pas d'autoplay : c'est au
              visiteur de décider.
              COMMENT : fichiers servis par maiProFiles (`/images/<id>`, répond 206 aux requêtes Range,
              exigé par Safari pour lire et pour se déplacer dans la vidéo). Pas de piste son. */}
          <figure className="overflow-hidden rounded-[10px] border border-border bg-card">
            <video
              src={portfolioImageUrl(projet.video.url)}
              poster={projet.video.poster_url ? portfolioImageUrl(projet.video.poster_url) : undefined}
              controls
              playsInline
              muted
              preload="metadata"
              data-testid="portfolio-video-player"
              className="h-auto w-full bg-background"
            />
            {projet.video.caption && (
              <figcaption className="border-t border-border px-4 py-3 text-sm text-muted-foreground" data-testid="portfolio-video-caption">
                {projet.video.caption}
              </figcaption>
            )}
          </figure>
        </section>
      )}

      {projet.stats?.length > 0 && (
        <section className={SECTION} data-testid="portfolio-stats">
          <TitreSection>{t('figures')}</TitreSection>
          {/* Lignes de preuve : libellé en <dt>, chiffre en <dd> (ordre sémantique), chiffre remonté
              à l'œil par flex-col-reverse. Filets d'1 px entre les cases (gap-px sur fond border). */}
          <dl className="grid grid-cols-2 gap-px overflow-hidden rounded-[10px] border border-border bg-border sm:grid-cols-4">
            {projet.stats.map((stat) => (
              <div key={stat.label} className="flex flex-col-reverse bg-card p-4 md:p-5">
                <dt className="mt-1 text-xs leading-snug text-muted-foreground">{stat.label.trim()}</dt>
                <dd className="font-heading text-2xl font-medium tabular-nums text-foreground md:text-3xl">{stat.value}</dd>
              </div>
            ))}
          </dl>
        </section>
      )}

      {projet.badges?.length > 0 && (
        <section className={SECTION}>
          <TitreSection>{t('stack')}</TitreSection>
          <div className="flex flex-wrap gap-2" data-testid="portfolio-badges">
            {projet.badges.map((badge) => (
              <span
                key={badge.label}
                className="rounded-md border border-border px-2.5 py-1 text-sm text-foreground/90"
              >
                {badge.label}
              </span>
            ))}
          </div>
        </section>
      )}

      {projet.images?.length > 1 && (
        <section className={SECTION} data-testid="portfolio-gallery">
          <TitreSection>{t('gallery')}</TitreSection>
          {/* La première image est déjà en tête de page : la galerie montre les SUIVANTES.
              Les visuels sont localisés côté maiProFiles — une capture d'interface est commune
              à toutes les langues, un diagramme légendé peut avoir sa variante. */}
          <div className="grid gap-5 sm:grid-cols-2">
            {projet.images.slice(1).map((visuel) => (
              <figure key={visuel.url} className="overflow-hidden rounded-[10px] border border-border bg-card">
                {/* QUOI : visuel de galerie via next/image, en lazy (sous la ligne de flottaison).
                    POURQUOI : variante redimensionnée à la demi-largeur de l'article au lieu de la
                    pleine définition. COMMENT : width/height ne fixent que le ratio provisoire,
                    `h-auto` rend le ratio réel une fois l'image chargée. */}
                <Image
                  src={portfolioImageUrl(visuel.url)}
                  alt={visuel.alt || projet.name}
                  width={1200}
                  height={750}
                  sizes="(min-width: 640px) 448px, 100vw"
                  className="h-auto w-full object-cover"
                />
                {visuel.caption && (
                  <figcaption className="border-t border-border px-4 py-3 text-xs leading-relaxed text-muted-foreground">
                    {visuel.caption}
                  </figcaption>
                )}
              </figure>
            ))}
          </div>
        </section>
      )}

      {projet.description_portfolio_html && (
        <section className={SECTION} data-testid="portfolio-technical">
          <TitreSection>{t('technical')}</TitreSection>
          {/* Fiche technique rendue en HTML côté Go (goldmark, comme le blog) : le front n'a pas
              de moteur markdown et en ajouter un pour ça seul serait une dépendance de plus. */}
          {/* prose-invert + couleurs du thème : titres serif ivoire, texte ivoire atténué, filets
              du thème — la fiche se lit comme le reste de la page, pas comme un document importé. */}
          <div
            className="prose prose-invert max-w-none prose-headings:font-heading prose-headings:font-medium prose-headings:text-foreground prose-p:text-foreground/85 prose-li:text-foreground/85 prose-strong:text-foreground prose-a:text-foreground prose-code:text-foreground prose-hr:border-border prose-th:text-foreground prose-td:text-foreground/85 prose-pre:border prose-pre:border-border prose-pre:bg-background"
            dangerouslySetInnerHTML={{ __html: projet.description_portfolio_html }}
          />
        </section>
      )}

      {Object.keys(projet.links || {}).length > 0 && (
        <section className={SECTION}>
          <TitreSection>{t('links')}</TitreSection>
          <ul className="flex flex-wrap gap-3" data-testid="portfolio-links">
            {Object.entries(projet.links).map(([type, url]) => {
              const href = url.startsWith('http') ? url : `https://${url}`;
              return (
                <li key={type}>
                  <a
                    href={href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex min-h-[44px] items-center gap-2 rounded-md border border-border px-4 text-sm text-foreground transition-colors hover:border-muted-foreground"
                  >
                    {t.has(`linkLabels.${type}`) ? t(`linkLabels.${type}`) : type}
                    <span aria-hidden="true">↗</span>
                  </a>
                </li>
              );
            })}
          </ul>
        </section>
      )}
    </article>
  );
}
