import { Metadata } from 'next';
import Image from 'next/image';
import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { socialMetadata } from '@/lib/seo/social';

import { portfolioApi, portfolioImageUrl, type PortfolioBook } from '@/lib/portfolio-api';
import { PageIntro } from '@/components/site-page/kit';

// Le portfolio est servi par maiProFiles à travers le backend Go : rendu à la demande, pour
// qu'une recuration apparaisse sans redéploiement du front.
export const dynamic = 'force-dynamic';

// QUOI : taille d'affichage réelle d'une capture de carte (grille 1/2/3 colonnes, conteneur
// max-w-6xl ≈ 1152 px → ~370 px par carte en 3 colonnes).
// POURQUOI : next/image en déduit la variante à demander à /_next/image — sans `sizes`, il
// servirait la pleine largeur de l'écran pour chaque carte.
const CARD_SIZES = '(min-width: 1024px) 370px, (min-width: 640px) 50vw, 100vw';
// Cartes au-dessus de la ligne de flottaison (première rangée en 3 colonnes) : chargées en
// priorité (preload, pas de lazy) car l'une d'elles est l'élément LCP. Les autres restent en lazy.
const PRIORITY_CARDS = 3;

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'portfolio' });
  return {
    title: t('meta.title'),
    description: t('meta.description'),
    ...socialMetadata({ locale, title: t('meta.title'), description: t('meta.description') }),
  };
}

async function getBook(locale: string): Promise<PortfolioBook | null> {
  try {
    return await portfolioApi.getIndex(locale);
  } catch {
    // Une panne de maiProFiles ne doit pas rendre une page d'erreur brute : on affiche un
    // message et le reste du site continue de vivre.
    return null;
  }
}

export default async function PortfolioPage({ params }: PageProps) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'portfolio' });
  const book = await getBook(locale);

  if (!book || book.projects.length === 0) {
    return (
      <div className="container max-w-3xl px-5 py-24 md:px-8">
        <PageIntro surtitre={t('eyebrow')} titre={t('title')} />
        <p className="mt-6 text-muted-foreground" data-testid="portfolio-unavailable">
          {t('unavailable')}
        </p>
      </div>
    );
  }

  return (
    <div className="container max-w-6xl px-5 py-14 md:px-8 md:py-20">
      <PageIntro surtitre={t('eyebrow')} titre={t('title')} chapeau={t('subtitle')} className="mb-12">
        <p className="mt-3 text-sm text-muted-foreground" data-testid="portfolio-count">
          {t('projectCount', { count: book.projects.length })}
        </p>
      </PageIntro>

      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {book.projects.map((projet, index) => (
          // La carte : un panneau de la nuit ; au survol, le trait se relève (or atténué) — la
          // capture reste l'argument, la carte ne fait que l'encadrer.
          <Link
            key={projet.id}
            href={`/${locale}/portfolio/${projet.id}`}
            data-testid="portfolio-card"
            data-project-id={projet.id}
            className="group flex flex-col overflow-hidden rounded-[10px] border border-border bg-card/90 transition-colors hover:border-gold/50 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {projet.image?.url && (
              // Capture du produit réel — c'est l'argument le plus fort de la page, donc elle
              // occupe le haut de la carte.
              // POURQUOI next/image : les captures servies par maiProFiles sont en pleine
              // définition (LCP ~12 s mesuré le 2026-10-12). L'optimiseur de Next les sert
              // redimensionnées à la taille de la carte, en AVIF/WebP, avec cache (.next/cache).
              // COMMENT : `fill` dans un conteneur au ratio fixe 16/10 (les dimensions d'origine ne
              // sont pas connues) ; le domaine est autorisé dans next.config.js (remotePatterns).
              <div className="relative aspect-[16/10] w-full overflow-hidden border-b border-border">
                <Image
                  src={portfolioImageUrl(projet.image.url)}
                  alt={projet.image.alt || projet.name}
                  fill
                  sizes={CARD_SIZES}
                  priority={index < PRIORITY_CARDS}
                  className="object-cover object-top transition-transform duration-300 group-hover:scale-[1.02]"
                />
              </div>
            )}
            <div className="flex flex-1 flex-col p-5 md:p-6">
              <h2 className="text-xl font-medium leading-tight text-foreground">{projet.index_name || projet.name}</h2>
              {projet.index_domain && (
                <p className="mt-1 text-sm text-muted-foreground">{projet.index_domain}</p>
              )}
              {projet.badges?.length > 0 && (
                <div className="mt-4 flex flex-wrap gap-1.5">
                  {projet.badges.slice(0, 5).map((badge) => (
                    <span
                      key={badge.label}
                      className="rounded border border-border px-2 py-0.5 text-xs text-muted-foreground"
                    >
                      {badge.label}
                    </span>
                  ))}
                </div>
              )}
              {/* mt-auto : le lien s'aligne en bas de toutes les cartes d'une rangée. */}
              <span className="mt-auto pt-5 text-sm font-medium text-foreground underline-offset-4 group-hover:underline">
                {t('viewProject')} <span aria-hidden="true">→</span>
              </span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
