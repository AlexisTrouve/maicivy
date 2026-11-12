import DevPortrait from '@/components/cv/DevPortrait';
import SkillsCloud from '@/components/cv/SkillsCloud';
import { CVData, LangStatsResponse } from '@/lib/types';

// ============================================================================
// Sections async de la page /cv — elles ABSORBENT l'attente des stats LOC.
// ----------------------------------------------------------------------------
// QUOI : deux composants serveur qui attendent la promesse LOC puis rendent leur bloc.
//
// POURQUOI : un `await` ne suspend QUE le composant qui l'exécute. La page attendait auparavant
// `Promise.all([cv, loc])` avant même d'atteindre le premier <Suspense> — les frontières existaient
// donc sans jamais servir, et la page entière restait bloquée sur le plus lent des fetchs. En
// déplaçant l'attente ici, elle est confinée SOUS une frontière : le HTML du CV part immédiatement
// et ces blocs sont streamés à l'arrivée de la donnée.
//
// POURQUOI un fichier séparé : la page est testée avec React Testing Library, qui rend hors du
// runtime Next et ne sait pas résoudre un composant serveur async imbriqué dans <Suspense>. Isolés
// ici, ces composants sont stubbables par le test de la page (comme DevPortrait l'est déjà) et
// gardent leur propre couverture. Les laisser dans page.tsx rendait la suite non exécutable.
//
// COMMENT : la promesse est créée UNE fois dans la page et partagée aux deux consommateurs —
// un seul appel réseau, deux attentes indépendantes.
// ============================================================================

// Bande Portrait Dev. Reçoit les LOC déjà en vol : DevPortrait ne refait donc pas cet appel.
export async function DevPortraitSection({
  locale,
  locStatsPromise,
}: {
  locale: string;
  locStatsPromise: Promise<LangStatsResponse | null>;
}) {
  const locStats = await locStatsPromise;
  return <DevPortrait locale={locale} langStats={locStats} />;
}

// Nuée de skills enrichie des LOC par langage (utilisées par la fiche détail, ouverte au clic).
export async function SkillsSection({
  cvData,
  locStatsPromise,
}: {
  cvData: CVData;
  locStatsPromise: Promise<LangStatsResponse | null>;
}) {
  const locStats = await locStatsPromise;
  return (
    <SkillsCloud
      skills={cvData.skills}
      projects={cvData.projects}
      experiences={cvData.experiences}
      langStats={locStats}
    />
  );
}
