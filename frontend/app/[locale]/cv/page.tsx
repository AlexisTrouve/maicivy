import { Suspense } from 'react';
import { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { socialMetadata } from '@/lib/seo/social';

// Force dynamic rendering to avoid build-time API calls
export const dynamic = 'force-dynamic';
// Sélecteur de thème en chargement paresseux : il tire @radix-ui/react-select (~510 kB de JS,
// chargés par AUCUNE autre page du site) pour un menu de 5 thèmes. Le dynamic() vit dans un
// composant client — `ssr: false` est ignoré EN SILENCE depuis un Server Component (mesuré : le
// chunk était renommé mais toujours chargé). Détail dans CVThemeSelectorLazy.tsx.
import CVThemeSelector from '@/components/cv/CVThemeSelectorLazy';
import ExperienceTimeline from '@/components/cv/ExperienceTimeline';
import SkillsCloud from '@/components/cv/SkillsCloud';
import ProjectsGrid from '@/components/cv/ProjectsGrid';
import ExportPDFButton from '@/components/cv/ExportPDFButton';
import { CVSkeleton } from '@/components/cv/CVSkeleton';
// Sections async isolées dans leur module : elles absorbent l'attente des stats LOC sous <Suspense>
// (cf. l'en-tête de sections.tsx pour le pourquoi du découpage).
import { DevPortraitSection, SkillsSection } from './sections';
import { PageHeading, SectionHeading } from '@/components/cv/PageHeading';
import { CVData, LangStatsResponse } from '@/lib/types';
import { DEFAULT_CV_THEME, isKnownCvTheme } from '@/lib/cvThemes';
// CV adapté au besoin déjà exprimé par le visiteur (chat public, dossier vivant) : lu CÔTÉ CLIENT
// (cookie de session) au-dessus d'un rendu serveur qui reste le CV standard — cf. CVContextProvider.
import { CVContextProvider } from '@/components/cv/context/CVContextProvider';
import { CVContextBanner } from '@/components/cv/context/CVContextBanner';
import { CVContextContent } from '@/components/cv/context/CVContextContent';
import { AdaptToOfferForm } from '@/components/cv/context/AdaptToOfferForm';

// Next 15 : params et searchParams arrivent en Promise (API de requête asynchrone).
interface CVPageProps {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{
    theme?: string;
  }>;
}

// Get API URL - internal for server, public for client
const getApiUrl = () => {
  if (typeof window === 'undefined') {
    return process.env.API_URL || 'http://maicivy-backend:8080';
  }
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
};

// Fetch CV data
async function getCVData(theme: string, lang: string): Promise<CVData> {
  const apiUrl = getApiUrl();
  const res = await fetch(
    `${apiUrl}/api/v1/cv?theme=${theme}&lang=${lang}`,
    {
      // Cache Next 5 min (au lieu de no-store). POURQUOI : en no-store, CHAQUE visiteur payait
      // l'aller-retour backend complet — sur le rendu serveur, donc avant le premier octet de HTML.
      // Le backend a déjà son cache Redis, mais ça ne dispense pas du round-trip réseau à chaque
      // requête. 5 min est un compromis assumé : un CV ne change pas d'une minute à l'autre, et le
      // visiteur suivant est servi sans attente.
      next: { revalidate: 300 },
    }
  );

  if (!res.ok) {
    throw new Error('Failed to fetch CV data');
  }

  return res.json();
}

// Récupère les LOC par langage (pour la fiche détail des skills). Enrichissement NON critique :
// si l'endpoint est indispo (Gitea down, token absent), on rend null et la fiche masque juste le
// bloc LOC — le CV reste affiché. Ce n'est pas un fallback qui masque un bug : c'est une dégradation
// gracieuse d'une donnée d'agrément, le cœur de page (CV) a son propre fetch qui, lui, throw.
async function getLocStats(): Promise<LangStatsResponse | null> {
  try {
    const res = await fetch(`${getApiUrl()}/api/v1/cv/loc`, { next: { revalidate: 300 } });
    if (!res.ok) return null;
    // `await` OBLIGATOIRE ici : sans lui on retourne la promesse de json(), qui rejette APRÈS la
    // sortie du try — donc hors de portée du catch. Cette fonction est désormais appelée sans await
    // (promesse partagée entre les deux <Suspense>), et une promesse flottante qui rejette devient
    // un unhandled rejection capable de tuer le process Node. Avec l'await, un corps illisible
    // retombe bien sur le null attendu.
    return await res.json();
  } catch {
    return null;
  }
}

// Nom affiché d'un profil : traduit s'il est connu de l'interface, sinon l'identifiant brut (un
// profil ajouté au backend apparaît, simplement non traduit — cf. lib/cvThemes.ts).
function profileName(t: (key: string) => string, theme: string): string {
  return isKnownCvTheme(theme) ? t(`themes.${theme}.name`) : theme;
}

// Generate dynamic metadata
export async function generateMetadata({
  params,
  searchParams,
}: CVPageProps): Promise<Metadata> {
  const { locale } = await params;
  const theme = (await searchParams).theme || DEFAULT_CV_THEME;
  const t = await getTranslations({ locale, namespace: 'cv' });
  const profile = profileName(t, theme);

  const title = t('metaTitle', { profile });
  const description = t('metaDescription', { profile });
  return {
    title,
    description,
    ...socialMetadata({ locale, title, description, type: 'profile' }),
  };
}

// Main CV Page Component
export default async function CVPage({ params, searchParams }: CVPageProps) {
  const resolvedParams = await params;
  // Profil par défaut = agents IA (positionnement du site) ; cf. lib/cvThemes.ts.
  const theme = (await searchParams).theme || DEFAULT_CV_THEME;
  const lang = resolvedParams.locale;

  // On n'attend QUE le CV — le cœur de la page. Les stats git (LOC + gitstats), qui sont
  // l'enrichissement le plus lent, sont déléguées aux <Suspense> ci-dessous sous forme de PROMESSE
  // NON attendue ici.
  //
  // POURQUOI : `await Promise.all([cv, loc])` bloquait le rendu entier sur le plus lent des deux
  // AVANT d'atteindre le moindre <Suspense>. Les frontières de Suspense existaient donc sans
  // jamais servir : leur skeleton ne pouvait pas s'afficher, puisqu'on n'arrivait au JSX qu'une
  // fois tous les fetchs terminés. Résultat, la page entière attendait les stats git.
  // En laissant la promesse flotter, le HTML du CV part immédiatement et les blocs qui en
  // dépendent sont streamés à l'arrivée de la donnée.
  //
  // COMMENT : `getLocStats()` est appelé SANS await — l'appel réseau démarre donc tout de suite,
  // en parallèle du fetch CV (pas de cascade). La promesse est passée aux deux consommateurs, qui
  // l'attendent chacun derrière sa propre frontière de Suspense.
  //
  // SÛRETÉ : une promesse flottante qui rejette = unhandled rejection (fatal sous Node). getLocStats
  // est écrit pour ne JAMAIS rejeter — try/catch englobant, ET `await res.json()` à l'intérieur du
  // try (sans cet await, un corps illisible rejetterait hors de portée du catch). Ne pas retirer
  // cet await sans remettre un .catch() ici.
  const locStatsPromise = getLocStats();
  const cvData = await getCVData(theme, lang);
  const t = await getTranslations({ locale: resolvedParams.locale, namespace: 'cv' });
  // Signature du site (nom + positionnement), lue dans les namespaces qui la portent déjà : le
  // surtitre ne recopie pas une chaîne que l'en-tête et le pied de page affichent ailleurs.
  const tNav = await getTranslations({ locale: resolvedParams.locale, namespace: 'nav' });
  const tVoice = await getTranslations({ locale: resolvedParams.locale, namespace: 'montagne.voice' });

  return (
    <div className="container mx-auto max-w-6xl px-4 py-12 md:py-16">
      <CVContextProvider>
        <PageHeading eyebrow={`${tNav('brand')} · ${tVoice('tagline')}`} title={t('title')} lead={t('lead')}>
          {/* Profil + export. L'export est L'action principale de la page : c'est le seul bouton
              plein (or) de l'écran. Le menu reste un champ, en creux. */}
          <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
            <div className="w-full sm:w-80">
              <label htmlFor="cv-profile" className="mb-2 block text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                {t('profileLabel')}
              </label>
              <CVThemeSelector currentTheme={theme} triggerId="cv-profile" />
            </div>
            <ExportPDFButton theme={theme} />
          </div>
          {/* Description du profil courant, rendue côté serveur : elle est là dès le premier octet
              et ne dépend pas du chargement paresseux du menu. */}
          {isKnownCvTheme(theme) && (
            <p className="mt-4 max-w-2xl text-sm text-muted-foreground" data-testid="cv-profile-description">
              {t(`themes.${theme}.description`)}
            </p>
          )}
          {/* Bandeau du CV adapté (rien sans contexte), puis le lien discret vers une offre. */}
          <CVContextBanner />
          <AdaptToOfferForm />
        </PageHeading>

        {/* Portrait Dev — en-tête de crédibilité : stats pro réelles (LOC, commits, genre récent,
            momentum, repos chauds), branché sur /cv/loc + /cv/gitstats. null si endpoints indispo. */}
        <div className="my-14">
          <Suspense fallback={<DevPortraitSkeleton />}>
            {/* La promesse LOC est passée telle quelle : DevPortrait l'attend DERRIÈRE cette
                frontière, donc son attente ne retient plus le reste de la page. */}
            <DevPortraitSection locale={resolvedParams.locale} locStatsPromise={locStatsPromise} />
          </Suspense>
        </div>

        {/* Main Content — le standard, remplacé par l'adapté quand il est prêt. */}
        <CVContextContent>
          <Suspense fallback={<CVSkeleton />}>
            <main className="space-y-20">
              <section id="experiences">
                <SectionHeading>{t('sections.experiences')}</SectionHeading>
                <ExperienceTimeline experiences={cvData.experiences} />
              </section>

              <section id="skills">
                <SectionHeading>{t('sections.skills')}</SectionHeading>
                {/* Suspense dédié : les skills s'affichent sans attendre les LOC, qui ne servent qu'à
                    la fiche détail d'un skill (ouverte au clic). Le fallback rend la même nuée sans
                    LOC — donc pas de saut de mise en page, juste un enrichissement qui arrive après. */}
                <Suspense
                  fallback={
                    <SkillsCloud
                      skills={cvData.skills}
                      projects={cvData.projects}
                      experiences={cvData.experiences}
                      langStats={null}
                    />
                  }
                >
                  <SkillsSection cvData={cvData} locStatsPromise={locStatsPromise} />
                </Suspense>
              </section>

              <section id="projects">
                <SectionHeading>{t('sections.projects')}</SectionHeading>
                <ProjectsGrid projects={cvData.projects} />
              </section>
            </main>
          </Suspense>
        </CVContextContent>
      </CVContextProvider>
    </div>
  );
}

// Skeleton de la bande Portrait Dev pendant le fetch (loc + gitstats).
// Mêmes rythmes que DevPortrait (padding, rangée de 3 cartes) pour limiter le saut à l'arrivée.
function DevPortraitSkeleton() {
  return (
    <div className="rounded-xl border border-border bg-card p-6 animate-pulse md:p-8">
      <div className="h-7 w-48 bg-muted rounded mb-8" />
      <div className="grid grid-cols-1 gap-4 mb-8 sm:grid-cols-3">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="h-20 bg-muted rounded-lg" />
        ))}
      </div>
      <div className="h-40 bg-muted rounded" />
    </div>
  );
}
