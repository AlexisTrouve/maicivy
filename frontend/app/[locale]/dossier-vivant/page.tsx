import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { DossierFooter } from '@/components/dossier-vivant/DossierFooter';
import { DossierVivant } from '@/components/dossier-vivant/DossierVivant';
import styles from '@/components/dossier-vivant/dossier-vivant.module.css';
import { getProjectFacts, getProjectIcons, replayContext, toSceneInput, type ProjectFactsMap, type ProjectIcons } from '@/lib/dossier-vivant/data';
import { getMontagne, type Montagne } from '@/lib/montagne-api';
import { montagneSerif } from '@/lib/montagne/fonts';
import { unpublishedRobots } from '@/lib/publication';
import { socialMetadata } from '@/lib/seo/social';

/**
 * /[locale]/dossier-vivant — un besoin de visiteur devient une tuile de la montagne, une équipe
 * d'agents (orchestrateur + Lecteur, Chercheur, Architecte, Rédacteur) le traite en parallèle et
 * rédige un dossier de conception avec preuves chiffrées.
 *
 * Tout visiteur, sans connexion, écrit son besoin et lance la VRAIE équipe (route publique du run,
 * côté navigateur) ; la démo enregistrée (scénario Tessera, aucun appel IA) reste à un clic. La
 * montagne, les noms, les chiffres des preuves et les démos sont lus ICI, côté serveur, sur les
 * vrais endpoints (montagne, fiches projets). La page est immersive : l'en-tête du site est là mais
 * défile avec elle, et la barre du dossier vivant colle dessous (lib/immersiveRoutes.ts).
 * La page DÉFILE : la scène au premier écran — à l'ouverture, prolongée jusqu'au pied de la
 * montagne, qu'on voit entière —, le dossier remis dessous, puis le pied de page
 * (DossierFooter, rendu ici côté serveur) et, sous lui, le pied commun du site (layout), par
 * lequel la page mène à tout le reste. Les nœuds complets de la montagne (et ses cartes du
 * sommet) voyagent jusqu'au navigateur pour une seule raison : la fleur d'un projet cliqué.
 * `?t=<s>` fige l'écran à un instant du rejeu (captures) ; `?speed=<x>` accélère son horloge (E2E).
 */

export const dynamic = 'force-dynamic';

type PageProps = { params: Promise<{ locale: string }>; searchParams: Promise<{ t?: string; speed?: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'dossierVivant.meta' });
  return {
    title: t('title'),
    description: t('description'),
    ...socialMetadata({ locale, title: t('title'), description: t('description') }),
    // Hors index tant que la page n'est pas publiée : la bascule est dans lib/publication.ts.
    robots: unpublishedRobots('dossier-vivant'),
  };
}

async function load(locale: string): Promise<{ montagne: Montagne; facts: ProjectFactsMap; icons: ProjectIcons } | null> {
  try {
    // Les icônes ne font jamais échouer la page (getProjectIcons rend {} en cas d'échec).
    const [montagne, facts, icons] = await Promise.all([getMontagne(locale), getProjectFacts(locale), getProjectIcons()]);
    return { montagne, facts, icons };
  } catch {
    // Backend indisponible : une phrase qui dit pourquoi, jamais une page d'erreur brute.
    return null;
  }
}

/** Nombre fini borné, ou null — les paramètres d'URL sont une entrée externe. */
function numberParam(v: string | undefined, min: number, max: number): number | null {
  if (v === undefined) return null;
  const n = Number(v);
  return Number.isFinite(n) && n >= min && n <= max ? n : null;
}

export default async function DossierVivantPage({ params, searchParams }: PageProps) {
  const [{ locale }, query] = await Promise.all([params, searchParams]);
  const [data, t] = await Promise.all([load(locale), getTranslations({ locale, namespace: 'dossierVivant' })]);

  if (!data) {
    return (
      <div className={`${styles.root} ${montagneSerif.variable}`}>
        <p className={styles.unavailable} data-testid="dv-unavailable">
          {t('unavailable')}
        </p>
      </div>
    );
  }

  return (
    <div className={montagneSerif.variable}>
      <DossierVivant
        sceneInput={toSceneInput(data.montagne, data.icons)}
        facts={data.facts}
        replay={replayContext(data.montagne, locale)}
        seek={numberParam(query.t, 0, 600)}
        speed={numberParam(query.speed, 0.25, 8) ?? 1}
        projects={{ nodes: data.montagne.nodes, summit: data.montagne.summit }}
        footer={<DossierFooter locale={locale} />}
      />
    </div>
  );
}
