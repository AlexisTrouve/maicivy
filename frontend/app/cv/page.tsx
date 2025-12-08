import { Suspense } from 'react';
import { Metadata } from 'next';
import CVThemeSelector from '@/components/cv/CVThemeSelector';
import ExperienceTimeline from '@/components/cv/ExperienceTimeline';
import SkillsCloud from '@/components/cv/SkillsCloud';
import ProjectsGrid from '@/components/cv/ProjectsGrid';
import ExportPDFButton from '@/components/cv/ExportPDFButton';
import { CVSkeleton } from '@/components/cv/CVSkeleton';
import { CVData } from '@/lib/types';

interface CVPageProps {
  searchParams: {
    theme?: string;
  };
}

// Fetch CV data
async function getCVData(theme: string = 'fullstack'): Promise<CVData> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/cv?theme=${theme}`,
    {
      next: { revalidate: 3600 }, // Cache 1 hour
    }
  );

  if (!res.ok) {
    throw new Error('Failed to fetch CV data');
  }

  return res.json();
}

// Generate dynamic metadata
export async function generateMetadata({
  searchParams,
}: CVPageProps): Promise<Metadata> {
  const theme = searchParams.theme || 'fullstack';

  return {
    title: `CV ${theme.charAt(0).toUpperCase() + theme.slice(1)} - Alexi`,
    description: `Découvrez mon profil ${theme} avec mes expériences, compétences et projets pertinents.`,
    openGraph: {
      title: `CV ${theme.charAt(0).toUpperCase() + theme.slice(1)} - Alexi`,
      description: `Profil ${theme} adapté`,
      type: 'profile',
    },
  };
}

// Main CV Page Component
export default async function CVPage({ searchParams }: CVPageProps) {
  const theme = searchParams.theme || 'fullstack';
  const cvData = await getCVData(theme);

  return (
    <div className="container mx-auto px-4 py-8 max-w-6xl">
      {/* Header */}
      <header className="mb-12 text-center">
        <h1 className="text-4xl md:text-5xl font-bold mb-4 bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
          Alexi - CV Dynamique
        </h1>
        <p className="text-lg text-gray-600 dark:text-gray-300 mb-6">
          CV adapté au profil : <span className="font-semibold">{theme}</span>
        </p>

        {/* Theme Selector & Export Button */}
        <div className="flex flex-col sm:flex-row gap-4 justify-center items-center">
          <CVThemeSelector currentTheme={theme} />
          <ExportPDFButton theme={theme} />
        </div>
      </header>

      {/* Main Content */}
      <Suspense fallback={<CVSkeleton />}>
        <main className="space-y-16">
          {/* Experiences Section */}
          <section id="experiences">
            <h2 className="text-3xl font-bold mb-8 flex items-center gap-3">
              <span className="text-blue-600">💼</span>
              Expériences Professionnelles
            </h2>
            <ExperienceTimeline experiences={cvData.experiences} />
          </section>

          {/* Skills Section */}
          <section id="skills">
            <h2 className="text-3xl font-bold mb-8 flex items-center gap-3">
              <span className="text-purple-600">🎯</span>
              Compétences
            </h2>
            <SkillsCloud skills={cvData.skills} />
          </section>

          {/* Projects Section */}
          <section id="projects">
            <h2 className="text-3xl font-bold mb-8 flex items-center gap-3">
              <span className="text-green-600">🚀</span>
              Projets
            </h2>
            <ProjectsGrid projects={cvData.projects} />
          </section>
        </main>
      </Suspense>
    </div>
  );
}
