import { Suspense } from 'react';
import { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { loadMessages } from '@/i18n/messages';

// Force dynamic rendering to avoid build-time API calls
export const dynamic = 'force-dynamic';
import CVThemeSelector from '@/components/cv/CVThemeSelector';
import ExperienceTimeline from '@/components/cv/ExperienceTimeline';
import SkillsCloud from '@/components/cv/SkillsCloud';
import ProjectsGrid from '@/components/cv/ProjectsGrid';
import ExportPDFButton from '@/components/cv/ExportPDFButton';
import DevPortrait from '@/components/cv/DevPortrait';
import { CVSkeleton } from '@/components/cv/CVSkeleton';
import { CVData, LangStatsResponse } from '@/lib/types';

interface CVPageProps {
  searchParams: {
    theme?: string;
  };
}

// Get API URL - internal for server, public for client
const getApiUrl = () => {
  if (typeof window === 'undefined') {
    return process.env.API_URL || 'http://maicivy-backend:8080';
  }
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
};

// Fetch CV data
async function getCVData(theme: string = 'fullstack', lang: string = 'fr'): Promise<CVData> {
  const apiUrl = getApiUrl();
  const res = await fetch(
    `${apiUrl}/api/v1/cv?theme=${theme}&lang=${lang}`,
    {
      // Pas de cache Next : la page reflète toujours le backend. Le backend a son propre cache
      // Redis (court) → fraîcheur sans recalcul à chaque requête. Évite le contenu figé 1h.
      cache: 'no-store',
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
    const res = await fetch(`${getApiUrl()}/api/v1/cv/loc`, { cache: 'no-store' });
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}

// Generate dynamic metadata
export async function generateMetadata({
  params,
  searchParams,
}: CVPageProps & { params: Promise<{ locale: string }> | { locale: string } }): Promise<Metadata> {
  const resolvedParams = params instanceof Promise ? await params : params;
  const theme = searchParams.theme || 'fullstack';
  const locale = resolvedParams.locale || 'en';
  const messages = loadMessages(locale);

  const themeName = messages.cv.themes[theme as keyof typeof messages.cv.themes] || theme;

  return {
    title: `CV ${themeName} - Alexis`,
    description: `${messages.cv.adaptedTo} ${themeName}`,
    openGraph: {
      title: `CV ${themeName} - Alexis`,
      description: `${messages.cv.adaptedTo} ${themeName}`,
      type: 'profile',
    },
  };
}

// Main CV Page Component
export default async function CVPage({ params, searchParams }: CVPageProps & { params: Promise<{ locale: string }> | { locale: string } }) {
  const resolvedParams = params instanceof Promise ? await params : params;
  const theme = searchParams.theme || 'fullstack';
  const lang = resolvedParams.locale || 'fr';
  // CV (cœur, throw si KO) + LOC (agrément, null si KO) en parallèle.
  const [cvData, locStats] = await Promise.all([getCVData(theme, lang), getLocStats()]);
  const t = await getTranslations({ locale: resolvedParams.locale, namespace: 'cv' });

  return (
    <div className="container mx-auto px-4 py-8 max-w-6xl">
      {/* Header */}
      <header className="mb-12 text-center">
        <h1 className="text-4xl md:text-5xl font-bold mb-4 bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
          Alexis - {t('title')}
        </h1>
        <p className="text-lg text-gray-600 dark:text-gray-300 mb-6">
          {t('adaptedTo')} <span className="font-semibold">{theme}</span>
        </p>

        {/* Theme Selector & Export Button */}
        <div className="flex flex-col sm:flex-row gap-4 justify-center items-center">
          <CVThemeSelector currentTheme={theme} />
          <ExportPDFButton theme={theme} />
        </div>
      </header>

      {/* Portrait Dev — en-tête de crédibilité : stats pro réelles (LOC, commits, genre récent,
          momentum, repos chauds), branché sur /cv/loc + /cv/gitstats. null si endpoints indispo. */}
      <div className="mb-12">
        <Suspense fallback={<DevPortraitSkeleton />}>
          <DevPortrait locale={resolvedParams.locale} />
        </Suspense>
      </div>

      {/* Main Content */}
      <Suspense fallback={<CVSkeleton />}>
        <main className="space-y-16">
          {/* Experiences Section */}
          <section id="experiences">
            <h2 className="text-3xl font-bold mb-8 flex items-center gap-3">
              <span className="text-blue-600">💼</span>
              {t('sections.experiences')}
            </h2>
            <ExperienceTimeline experiences={cvData.experiences} />
          </section>

          {/* Skills Section */}
          <section id="skills">
            <h2 className="text-3xl font-bold mb-8 flex items-center gap-3">
              <span className="text-purple-600">🎯</span>
              {t('sections.skills')}
            </h2>
            <SkillsCloud
              skills={cvData.skills}
              projects={cvData.projects}
              experiences={cvData.experiences}
              langStats={locStats}
            />
          </section>

          {/* Projects Section */}
          <section id="projects">
            <h2 className="text-3xl font-bold mb-8 flex items-center gap-3">
              <span className="text-green-600">🚀</span>
              {t('sections.projects')}
            </h2>
            <ProjectsGrid projects={cvData.projects} />
          </section>
        </main>
      </Suspense>
    </div>
  );
}

// Skeleton de la bande Portrait Dev pendant le fetch (loc + gitstats).
function DevPortraitSkeleton() {
  return (
    <div className="rounded-xl border bg-card p-6 animate-pulse">
      <div className="h-7 w-40 bg-muted rounded mb-6" />
      <div className="grid grid-cols-3 gap-4 mb-6">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="h-20 bg-muted rounded-lg" />
        ))}
      </div>
      <div className="h-40 bg-muted rounded" />
    </div>
  );
}
