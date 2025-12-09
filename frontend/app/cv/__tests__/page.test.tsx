import { render, screen, waitFor } from '@testing-library/react';
import CVPage, { generateMetadata } from '../page';

// Mock Next.js components
jest.mock('next/link', () => {
  return ({ children, href }: { children: React.ReactNode; href: string }) => {
    return <a href={href}>{children}</a>;
  };
});

// Mock CV components
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
  name: 'Alexi',
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
    render(await CVPage({ searchParams: {} }));

    expect(
      screen.getByRole('heading', { name: /Alexi - CV Dynamique/i })
    ).toBeInTheDocument();
  });

  it('should render with default theme (fullstack)', async () => {
    render(await CVPage({ searchParams: {} }));

    expect(screen.getByText(/CV adapté au profil :/i)).toBeInTheDocument();
    const fullstackTexts = screen.getAllByText(/fullstack/i);
    expect(fullstackTexts.length).toBeGreaterThan(0);
  });

  it('should render with specified theme', async () => {
    render(await CVPage({ searchParams: { theme: 'backend' } }));

    const backendTexts = screen.getAllByText(/backend/i);
    expect(backendTexts.length).toBeGreaterThan(0);
  });

  it('should render CVThemeSelector component', async () => {
    render(await CVPage({ searchParams: { theme: 'frontend' } }));

    const themeSelector = screen.getByTestId('theme-selector');
    expect(themeSelector).toBeInTheDocument();
    expect(themeSelector).toHaveTextContent('Theme: frontend');
  });

  it('should render ExportPDFButton component', async () => {
    render(await CVPage({ searchParams: { theme: 'devops' } }));

    const exportButton = screen.getByTestId('export-pdf');
    expect(exportButton).toBeInTheDocument();
    expect(exportButton).toHaveTextContent('Export PDF (devops)');
  });

  it('should render experiences section', async () => {
    render(await CVPage({ searchParams: {} }));

    expect(
      screen.getByRole('heading', { name: /Expériences Professionnelles/i })
    ).toBeInTheDocument();

    const timeline = screen.getByTestId('experience-timeline');
    expect(timeline).toBeInTheDocument();
    expect(timeline).toHaveTextContent('Test Company');
  });

  it('should render skills section', async () => {
    render(await CVPage({ searchParams: {} }));

    expect(
      screen.getByRole('heading', { name: /Compétences/i })
    ).toBeInTheDocument();

    const skillsCloud = screen.getByTestId('skills-cloud');
    expect(skillsCloud).toBeInTheDocument();
    expect(skillsCloud).toHaveTextContent('JavaScript');
    expect(skillsCloud).toHaveTextContent('TypeScript');
  });

  it('should render projects section', async () => {
    render(await CVPage({ searchParams: {} }));

    expect(
      screen.getByRole('heading', { name: /Projets/i })
    ).toBeInTheDocument();

    const projectsGrid = screen.getByTestId('projects-grid');
    expect(projectsGrid).toBeInTheDocument();
    expect(projectsGrid).toHaveTextContent('Test Project');
  });

  it('should fetch CV data with correct theme parameter', async () => {
    render(await CVPage({ searchParams: { theme: 'backend' } }));

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

    await expect(CVPage({ searchParams: {} })).rejects.toThrow(
      'Failed to fetch CV data'
    );

    consoleErrorSpy.mockRestore();
  });

  it('should have proper section IDs for navigation', async () => {
    const { container } = render(await CVPage({ searchParams: {} }));

    expect(container.querySelector('#experiences')).toBeInTheDocument();
    expect(container.querySelector('#skills')).toBeInTheDocument();
    expect(container.querySelector('#projects')).toBeInTheDocument();
  });

  it('should render responsive container', async () => {
    const { container } = render(await CVPage({ searchParams: {} }));

    const mainContainer = container.querySelector('.container');
    expect(mainContainer).toBeInTheDocument();
    expect(mainContainer).toHaveClass('container', 'mx-auto');
  });
});

describe('CVPage - generateMetadata', () => {
  it('should generate metadata with default theme', async () => {
    const metadata = await generateMetadata({ searchParams: {} });

    expect(metadata.title).toBe('CV Fullstack - Alexi');
    expect(metadata.description).toContain('profil fullstack');
  });

  it('should generate metadata with custom theme', async () => {
    const metadata = await generateMetadata({
      searchParams: { theme: 'backend' },
    });

    expect(metadata.title).toBe('CV Backend - Alexi');
    expect(metadata.description).toContain('profil backend');
  });

  it('should capitalize theme in title', async () => {
    const metadata = await generateMetadata({
      searchParams: { theme: 'frontend' },
    });

    expect(metadata.title).toBe('CV Frontend - Alexi');
  });

  it('should include OpenGraph metadata', async () => {
    const metadata = await generateMetadata({
      searchParams: { theme: 'devops' },
    });

    expect(metadata.openGraph?.title).toBe('CV Devops - Alexi');
    expect(metadata.openGraph?.description).toContain('Profil devops');
    expect(metadata.openGraph?.type).toBe('profile');
  });
});
