import { render, screen, waitFor } from '@testing-library/react';
import CVPage, { generateMetadata } from '../page';

// Mock Next.js components
jest.mock('next/link', () => {
  return ({ children, href }: { children: React.ReactNode; href: string }) => {
    return <a href={href}>{children}</a>;
  };
});

// Sections async (DevPortraitSection / SkillsSection) = composants SERVEUR qui await la promesse
// LOC sous <Suspense>. RTL rend hors du runtime Next et ne sait pas résoudre un composant serveur
// async imbriqué dans une frontière Suspense → on les stubbe, comme DevPortrait l'était déjà.
// Ce test couvre le LAYOUT du CV ; le rendu interne est couvert par DevPortrait.test.tsx et
// SkillsCloud.test.tsx.
jest.mock('../sections', () => ({
  DevPortraitSection: function MockDevPortraitSection() {
    return <div data-testid="dev-portrait">Dev Portrait Component</div>;
  },
  // SkillsSection rend SkillsCloud une fois les LOC résolues. Le stub court-circuite l'await mais
  // reproduit la même sortie (testid + noms des skills) : l'assertion « les skills du CV arrivent
  // bien jusqu'à la nuée » reste vérifiée, sans dépendre d'un composant serveur async.
  SkillsSection: function MockSkillsSection({ cvData }: { cvData: any }) {
    return (
      <div data-testid="skills-cloud">
        {cvData.skills.map((skill: any) => (
          <div key={skill.id}>{skill.name}</div>
        ))}
      </div>
    );
  },
}));

jest.mock('@/components/cv/CVThemeSelector', () => {
  return function MockCVThemeSelector({ currentTheme }: { currentTheme: string }) {
    return <div data-testid="theme-selector">Theme: {currentTheme}</div>;
  };
});

jest.mock('@/components/cv/ExperienceTimeline', () => {
  return function MockExperienceTimeline({ experiences }: { experiences: any[] }) {
    return (
      <div data-testid="experience-timeline">
        {experiences.map((exp: any) => (
          <div key={exp.id}>{exp.company}</div>
        ))}
      </div>
    );
  };
});

jest.mock('@/components/cv/SkillsCloud', () => {
  return function MockSkillsCloud({ skills }: { skills: any[] }) {
    return (
      <div data-testid="skills-cloud">
        {skills.map((skill: any) => (
          <div key={skill.id}>{skill.name}</div>
        ))}
      </div>
    );
  };
});

jest.mock('@/components/cv/ProjectsGrid', () => {
  return function MockProjectsGrid({ projects }: { projects: any[] }) {
    return (
      <div data-testid="projects-grid">
        {projects.map((project: any) => (
          <div key={project.id}>{project.name}</div>
        ))}
      </div>
    );
  };
});

jest.mock('@/components/cv/ExportPDFButton', () => {
  return function MockExportPDFButton({ theme }: { theme: string }) {
    return <button data-testid="export-pdf">Export PDF ({theme})</button>;
  };
});

jest.mock('@/components/cv/CVSkeleton', () => {
  return {
    CVSkeleton: function MockCVSkeleton() {
      return <div data-testid="cv-skeleton">Loading...</div>;
    },
  };
});

const mockCVData = {
  name: 'Alexis',
  title: 'Full Stack Developer',
  experiences: [
    {
      id: '1',
      company: 'Test Company',
      position: 'Developer',
      startDate: '2020-01-01',
      endDate: '2023-01-01',
      description: 'Test description',
      technologies: ['React', 'Node.js'],
    },
  ],
  skills: [
    { id: '1', name: 'JavaScript', level: 'Expert', category: 'Frontend' },
    { id: '2', name: 'TypeScript', level: 'Advanced', category: 'Frontend' },
  ],
  projects: [
    {
      id: '1',
      name: 'Test Project',
      description: 'A test project',
      technologies: ['React'],
      url: 'https://example.com',
    },
  ],
};

// Mock fetch
global.fetch = jest.fn();

describe('CVPage', () => {
  beforeEach(() => {
    // Mock environment variable
    process.env.NEXT_PUBLIC_API_URL = 'http://localhost:3001';

    // Setup default fetch mock
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => mockCVData,
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should render page heading', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) }));

    expect(
      screen.getByRole('heading', { level: 1, name: 'Curriculum vitæ' })
    ).toBeInTheDocument();
    // Plus de titre-slogan en dégradé bleu→violet.
    expect(screen.queryByText(/CV Dynamique/i)).toBeNull();
  });

  // Profil par défaut = agents IA (positionnement « Technomancer ») : sélecteur, export, requête
  // au backend et description du profil partent tous de `ai` quand l'URL ne précise rien.
  it('should render with default theme (ai — Technomancer)', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) }));

    expect(screen.getByTestId('theme-selector')).toHaveTextContent('Theme: ai');
    expect(screen.getByTestId('export-pdf')).toHaveTextContent('Export PDF (ai)');
    expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('theme=ai'), expect.any(Object));
    expect(screen.getByTestId('cv-profile-description')).toHaveTextContent(/Systèmes multi-agents en production/);
  });

  it('should name the profile selector with a visible label', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) }));

    const label = screen.getByText('Profil');
    expect(label.tagName).toBe('LABEL');
    expect(label).toHaveAttribute('for', 'cv-profile');
  });

  it('should render with specified theme', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({ theme: 'backend' }) }));

    expect(screen.getByTestId('theme-selector')).toHaveTextContent('Theme: backend');
    expect(screen.getByTestId('cv-profile-description')).toHaveTextContent('APIs, bases de données, services.');
  });

  // Un profil inconnu de l'interface (ajouté au backend plus tard) : pas de description inventée.
  it('should not render a description for an unknown theme', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({ theme: 'frontend' }) }));

    expect(screen.queryByTestId('cv-profile-description')).toBeNull();
  });

  it('should render CVThemeSelector component', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({ theme: 'frontend' }) }));

    const themeSelector = screen.getByTestId('theme-selector');
    expect(themeSelector).toBeInTheDocument();
    expect(themeSelector).toHaveTextContent('Theme: frontend');
  });

  it('should render ExportPDFButton component', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({ theme: 'devops' }) }));

    const exportButton = screen.getByTestId('export-pdf');
    expect(exportButton).toBeInTheDocument();
    expect(exportButton).toHaveTextContent('Export PDF (devops)');
  });

  it('should render experiences section', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) }));

    expect(
      screen.getByRole('heading', { level: 2, name: 'Expériences' })
    ).toBeInTheDocument();

    const timeline = screen.getByTestId('experience-timeline');
    expect(timeline).toBeInTheDocument();
    expect(timeline).toHaveTextContent('Test Company');
  });

  it('should render skills section', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) }));

    expect(
      screen.getByRole('heading', { level: 2, name: 'Compétences' })
    ).toBeInTheDocument();

    const skillsCloud = screen.getByTestId('skills-cloud');
    expect(skillsCloud).toBeInTheDocument();
    expect(skillsCloud).toHaveTextContent('JavaScript');
    expect(skillsCloud).toHaveTextContent('TypeScript');
  });

  it('should render projects section', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) }));

    expect(
      screen.getByRole('heading', { level: 2, name: 'Projets' })
    ).toBeInTheDocument();

    const projectsGrid = screen.getByTestId('projects-grid');
    expect(projectsGrid).toBeInTheDocument();
    expect(projectsGrid).toHaveTextContent('Test Project');
  });

  it('should fetch CV data with correct theme parameter', async () => {
    render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({ theme: 'backend' }) }));

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('theme=backend'),
      expect.any(Object)
    );
  });

  it('should handle API error gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    (global.fetch as jest.Mock).mockResolvedValue({
      ok: false,
      status: 500,
    });

    await expect(CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) })).rejects.toThrow(
      'Failed to fetch CV data'
    );

    consoleErrorSpy.mockRestore();
  });

  it('should have proper section IDs for navigation', async () => {
    const { container } = render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) }));

    expect(container.querySelector('#experiences')).toBeInTheDocument();
    expect(container.querySelector('#skills')).toBeInTheDocument();
    expect(container.querySelector('#projects')).toBeInTheDocument();
  });

  it('should render responsive container', async () => {
    const { container } = render(await CVPage({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) }));

    const mainContainer = container.querySelector('.container');
    expect(mainContainer).toBeInTheDocument();
    expect(mainContainer).toHaveClass('container', 'mx-auto');
  });
});

describe('CVPage - generateMetadata', () => {
  it('should generate metadata with default theme (ai)', async () => {
    const metadata = await generateMetadata({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({}) });

    // themes.ai.name = "Agents IA — Technomancer" dans messages/fr.json
    expect(metadata.title).toBe('CV — Agents IA — Technomancer · Alexis Trouvé');
    expect(metadata.description).toContain('Agents IA — Technomancer');
  });

  // Profil inconnu de l'interface : son identifiant brut, plutôt qu'une clé de message affichée.
  it('should fall back to the raw id for an unknown theme', async () => {
    const metadata = await generateMetadata({ params: Promise.resolve({ locale: 'fr' }), searchParams: Promise.resolve({ theme: 'quantum' }) });

    expect(metadata.title).toBe('CV — quantum · Alexis Trouvé');
  });

  it('should generate metadata with custom theme', async () => {
    const metadata = await generateMetadata({
      params: Promise.resolve({ locale: 'fr' }),
      searchParams: Promise.resolve({ theme: 'backend' }),
    });

    // themes.backend.name = "Backend" dans messages/fr.json
    expect(metadata.title).toBe('CV — Backend · Alexis Trouvé');
    expect(metadata.description).toContain('Backend');
  });

  it('should use the translated profile name in the title', async () => {
    const metadata = await generateMetadata({
      params: Promise.resolve({ locale: 'fr' }),
      searchParams: Promise.resolve({ theme: 'cpp' }),
    });

    // themes.cpp.name = "C++" dans messages/fr.json
    expect(metadata.title).toBe('CV — C++ · Alexis Trouvé');
  });

  it('should include OpenGraph metadata', async () => {
    const metadata = await generateMetadata({
      params: Promise.resolve({ locale: 'fr' }),
      searchParams: Promise.resolve({ theme: 'devops' }),
    });

    // themes.devops.name = "DevOps" dans messages/fr.json
    expect(metadata.openGraph?.title).toBe('CV — DevOps · Alexis Trouvé');
    expect(metadata.openGraph?.description).toContain('DevOps');
    expect(metadata.openGraph?.type).toBe('profile');
  });
});
