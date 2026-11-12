import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { MontagneExplorer } from '@/components/montagne/MontagneExplorer';
import { MontagneList } from '@/components/montagne/MontagneList';
import { MontagnePeak } from '@/components/montagne/MontagnePeak';
import { MontagneVoice } from '@/components/montagne/MontagneVoice';
import styles from '@/components/montagne/montagne.module.css';
import { getMontagne, type Montagne } from '@/lib/montagne-api';
import { PERSON_SCHEMA } from '@/lib/site/identity';
import { socialMetadata } from '@/lib/seo/social';

/**
 * /[locale] — l'accueil EST la montagne : les projets d'Alexis en un relief, la voix en tête, les
 * œuvres au sommet, puis tous les projets en rangs qui descendent jusqu'aux fondations, reliés par
 * les liens réels « sert à construire ». (Jusqu'à la refonte de 2026-11, elle vivait sur
 * /[locale]/montagne, qui redirige désormais ici en 308 — next.config.js.)
 *
 * Tout ce qui se voit à l'arrivée est dans le HTML SERVEUR : voix, sommet, montagne (HTML + SVG), et
 * la liste complète en dessous — instantané, indexable, utilisable sans JavaScript. Le JavaScript
 * n'ajoute que la fleur au clic et la vue réseau 3D, chargée au clic (MontagneExplorer). Conçue
 * d'abord pour le téléphone (393 px), puis élargie.
 * L'en-tête et le pied de page sont ceux de tout le site (app/[locale]/layout.tsx) ; l'en-tête a
 * repris le langage de l'ancien bandeau de la montagne.
 */

// Rendu à la demande : une recuration côté maiProFiles apparaît sans redéploiement du front (le
// backend porte le cache Redis devant maiProFiles).
export const dynamic = 'force-dynamic';

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'montagne.meta' });
  return {
    title: t('title'),
    description: t('description'),
    ...socialMetadata({ locale, title: t('title'), description: t('description') }),
  };
}

async function loadMontagne(locale: string): Promise<Montagne | null> {
  try {
    return await getMontagne(locale);
  } catch {
    // Backend ou maiProFiles indisponible sans rien en cache : la voix reste, la montagne dit
    // pourquoi elle est absente — jamais une page d'erreur brute.
    return null;
  }
}

export default async function HomePage({ params }: PageProps) {
  const { locale } = await params;
  const [montagne, t] = await Promise.all([loadMontagne(locale), getTranslations({ locale, namespace: 'montagne.stage' })]);

  return (
    <div className={`${styles.sky} isolate min-h-screen text-slate-200`}>
      {/* JSON-LD Person — rendu côté serveur dans le HTML, lu par Google pour construire l'entité
          « Alexis Trouvé » (Knowledge Panel). Identité stable → constante (lib/site/identity.ts),
          aucun fetch qui retarderait le premier octet de l'accueil. */}
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(PERSON_SCHEMA) }} />
      <MontagneVoice locale={locale} />
      {montagne ? (
        <>
          <MontagneExplorer montagne={montagne}>
            <MontagnePeak locale={locale} montagne={montagne} />
          </MontagneExplorer>
          <MontagneList locale={locale} montagne={montagne} />
        </>
      ) : (
        <p data-testid="montagne-unavailable" className="px-6 py-16 text-center text-sm text-slate-500">
          {t('unavailable')}
        </p>
      )}
    </div>
  );
}
