import { render, screen, act, waitFor, within } from '@testing-library/react';
import ProjectsGrid from '../ProjectsGrid';
import { Project } from '@/lib/types';

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  ExternalLink: ({ className }: any) => <svg data-testid="external-link-icon" className={className} />,
  Github: ({ className }: any) => <svg data-testid="github-icon" className={className} />,
  Star: ({ className }: any) => <svg data-testid="star-icon" className={className} />,
  Clock: ({ className }: any) => <svg data-testid="clock-icon" className={className} />,
}));

// framer-motion mocké globalement (cf. __mocks__/framer-motion.tsx)

// Mock next/link
jest.mock('next/link', () => {
  return ({ children, href, ...props }: any) => {
    return <a href={href} {...props}>{children}</a>;
  };
});

describe('ProjectsGrid', () => {
  afterEach(() => {
    // Clear all timers to prevent memory leaks
    jest.clearAllTimers();
  });
  const mockProjects: Project[] = [
    {
      id: '1',
      title: 'E-commerce Platform',
      description: 'Built scalable e-commerce platform handling 10k+ daily users',
      technologies: ['Go', 'React', 'PostgreSQL', 'Redis', 'Docker'],
      githubUrl: 'https://github.com/johndoe/ecommerce',
      demoUrl: 'https://demo.ecommerce.com',
      stars: 150,
      language: 'TypeScript',
      featured: true,
      score: 0.95,
    },
    {
      id: '2',
      title: 'Real-time Chat App',
      description: 'WebSocket-based chat application with rooms and file sharing',
      technologies: ['Node.js', 'Socket.io', 'MongoDB'],
      githubUrl: 'https://github.com/johndoe/chat',
      stars: 75,
      language: 'JavaScript',
      featured: false,
      score: 0.82,
    },
    {
      id: '3',
      title: 'ML Image Classifier',
      description: 'Deep learning model for image classification',
      technologies: ['Python', 'TensorFlow', 'FastAPI'],
      language: 'Python',
      featured: false,
      score: 0.70,
    },
  ];

  it('should render all projects', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    expect(screen.getByText('E-commerce Platform')).toBeInTheDocument();
    expect(screen.getByText('Real-time Chat App')).toBeInTheDocument();
    expect(screen.getByText('ML Image Classifier')).toBeInTheDocument();
  });

  it('should display project descriptions', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    expect(
      screen.getByText(/Built scalable e-commerce platform/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/WebSocket-based chat application/i)
    ).toBeInTheDocument();
  });

  it('should show featured badge for featured projects', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    expect(screen.getByText('Projet mis en avant')).toBeInTheDocument();
  });

  it('should display technologies with max 3 visible', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    // First project has 5 technologies, component shows 3 + "+2"
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('React')).toBeInTheDocument();
    expect(screen.getByText('PostgreSQL')).toBeInTheDocument();
    // Redis and Docker are hidden, overflow shown as +2
    expect(screen.getByText('+2')).toBeInTheDocument();
  });

  it('should display project categories in badge when available', () => {
    const projectWithCategory = {
      ...mockProjects[0],
      category: 'Backend',
    };
    const { container } = render(<ProjectsGrid projects={[projectWithCategory]} />);

    // Category badge is shown in top-right absolute div
    expect(screen.getByText('Backend')).toBeInTheDocument();
    // The badge has specific styling
    const badge = screen.getByText('Backend');
    expect(badge).toHaveClass('rounded-full');
  });

  it('should show click-for-details hint on all cards', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    // « Voir la fiche » apparaît sur chaque carte
    const hints = screen.getAllByText('Voir la fiche');
    expect(hints.length).toBe(mockProjects.length);
  });

  it('should render cards as accessible buttons with aria-label', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    // Each card is a role="button" div with aria-label containing the project title
    expect(screen.getByRole('button', { name: /E-commerce Platform/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Real-time Chat App/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /ML Image Classifier/i })).toBeInTheDocument();
  });

  it('should show in-progress badge when project is in progress', () => {
    const inProgressProject = {
      ...mockProjects[1],
      inProgress: true,
    };
    render(<ProjectsGrid projects={[inProgressProject]} />);

    // t('inProgress') = "En cours"
    expect(screen.getByText('En cours')).toBeInTheDocument();
  });

  it('should render the grid container and all three cards', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    const grid = container.querySelector('.grid');
    expect(grid).toBeInTheDocument();
    // Each project gets a card (role="button")
    const cards = screen.getAllByRole('button');
    expect(cards.length).toBe(mockProjects.length);
  });

  it('should display technology tags with correct styling', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    // Pastilles techno : jetons du thème uniquement (aucune couleur Tailwind en dur).
    const techSpans = within(container).getAllByTestId('tech-chip');
    expect(techSpans.length).toBeGreaterThan(0);
    techSpans.forEach((chip) => {
      expect(chip).toHaveClass('bg-secondary', 'text-muted-foreground');
      expect(chip.className).not.toMatch(/gray|blue|purple/);
    });
    // First project shows Go, React, PostgreSQL (3 visible)
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('React')).toBeInTheDocument();
  });

  // Projet mis en avant = projet « retenu » : un filet d'or en tête de carte, jamais un aplat.
  it('should mark featured cards with a gold top rule only', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    const cards = screen.getAllByTestId('project-card');
    const featured = cards.filter((c) => c.getAttribute('data-featured') === 'true');
    expect(featured.length).toBe(mockProjects.filter((p) => p.featured).length);
    expect(featured.length).toBeGreaterThan(0);
    featured.forEach((c) => {
      expect(c).toHaveClass('border-t-gold');
      expect(c.className).not.toMatch(/\bbg-gold\b|yellow/);
    });
    cards
      .filter((c) => c.getAttribute('data-featured') !== 'true')
      .forEach((c) => expect(c).not.toHaveClass('border-t-gold'));
  });

  it('should render in grid layout', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    const grid = container.querySelector('.grid.grid-cols-1.md\\:grid-cols-2.lg\\:grid-cols-3');
    expect(grid).toBeInTheDocument();
  });

  it('should handle empty projects array', () => {
    const { container } = render(<ProjectsGrid projects={[]} />);

    const grid = container.querySelector('.grid');
    expect(grid).toBeInTheDocument();
    expect(grid?.children.length).toBe(0);
  });

  it('should handle project without optional fields', () => {
    const minimalProject: Project = {
      id: '4',
      title: 'Minimal Project',
      description: 'A project with minimal information',
      technologies: ['JavaScript'],
      featured: false,
    };

    render(<ProjectsGrid projects={[minimalProject]} />);

    expect(screen.getByText('Minimal Project')).toBeInTheDocument();
    expect(screen.getByText('JavaScript')).toBeInTheDocument();
    expect(screen.queryByText('Code')).not.toBeInTheDocument();
    expect(screen.queryByText('Demo')).not.toBeInTheDocument();
  });

  it('should show star icon in the featured badge', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    // Featured project has a Star SVG icon
    const starIcon = container.querySelector('[data-testid="star-icon"]');
    expect(starIcon).toBeInTheDocument();
  });

  it('should render at least one svg icon in the grid', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    // Featured project shows Star SVG + potentially Clock for in-progress
    const svgElements = container.querySelectorAll('svg');
    // At minimum the Star icon from the featured badge is present
    expect(svgElements.length).toBeGreaterThan(0);
  });

  // Les cartes sont posées sur la surface « carte » du thème (le site n'a qu'une chrome sombre).
  it('should render cards on the theme card surface', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    const cards = screen.getAllByTestId('project-card');
    expect(cards).toHaveLength(3);
    cards.forEach((c) => expect(c).toHaveClass('bg-card'));
  });

  // ==========================================================================
  // Lazy-load des cartes — le CV rend 79 projets, on n'en MONTE que 12 au départ.
  // Le rendu serveur, lui, les contient tous (SEO) : l'élagage n'a lieu qu'après hydratation.
  // ==========================================================================
  describe('lazy-load', () => {
    // 30 projets : au-delà du premier palier (12), sous le total, donc les deux bornes sont
    // observables sur le même jeu de données.
    const beaucoup: Project[] = Array.from({ length: 30 }, (_, i) => ({
      id: `p${i}`,
      title: `Projet ${i}`,
      description: `Description ${i}`,
      technologies: ['Go'],
      featured: false,
    }));

    // Observer CONTRÔLÉ : le mock global déclenche l'intersection tout seul (setTimeout), ce qui
    // révélerait tous les paliers d'affilée et rendrait le test aveugle. Ici on garde la main sur
    // le moment du déclenchement.
    let declencher: (() => void) | null = null;
    let originel: typeof IntersectionObserver;

    beforeEach(() => {
      originel = global.IntersectionObserver;
      declencher = null;
      global.IntersectionObserver = class {
        callback: IntersectionObserverCallback;
        constructor(cb: IntersectionObserverCallback) {
          this.callback = cb;
          // Le dernier observer câblé est celui qui pilote le palier suivant (l'effet est recréé
          // à chaque changement de `nbAffichees`).
          declencher = () =>
            this.callback([{ isIntersecting: true } as IntersectionObserverEntry], this as never);
        }
        observe() {}
        unobserve() {}
        disconnect() {}
        takeRecords() {
          return [];
        }
        root = null;
        rootMargin = '';
        thresholds = [];
      } as unknown as typeof IntersectionObserver;
    });

    afterEach(() => {
      global.IntersectionObserver = originel;
    });

    it('ne monte que le premier palier de cartes après hydratation', async () => {
      render(<ProjectsGrid projects={beaucoup} />);

      // L'effet d'élagage tourne après le rendu initial → on retombe à 12.
      await waitFor(() => {
        expect(screen.getAllByTestId('project-card')).toHaveLength(12);
      });
      // Les 18 restants ne sont pas montés — c'est tout l'intérêt.
      expect(screen.queryByText('Projet 29')).not.toBeInTheDocument();
    });

    it('révèle un palier supplémentaire quand la sentinelle entre dans le viewport', async () => {
      render(<ProjectsGrid projects={beaucoup} />);
      await waitFor(() => {
        expect(screen.getAllByTestId('project-card')).toHaveLength(12);
      });

      act(() => declencher?.());

      await waitFor(() => {
        expect(screen.getAllByTestId('project-card')).toHaveLength(24);
      });
    });

    it('retire la sentinelle une fois tous les projets révélés', async () => {
      render(<ProjectsGrid projects={beaucoup} />);
      await waitFor(() => {
        expect(screen.getAllByTestId('project-card')).toHaveLength(12);
      });

      // 12 → 24 → 30 (le dernier palier est tronqué au total, pas à 36).
      act(() => declencher?.());
      await waitFor(() => expect(screen.getAllByTestId('project-card')).toHaveLength(24));
      act(() => declencher?.());

      await waitFor(() => {
        expect(screen.getAllByTestId('project-card')).toHaveLength(30);
      });
      // Plus rien à révéler → la sentinelle disparaît (et l'observer est débranché).
      expect(screen.queryByTestId('projects-sentinel')).not.toBeInTheDocument();
    });

    it('affiche tout sans jamais tronquer quand la liste tient dans le premier palier', async () => {
      const peu = beaucoup.slice(0, 5);
      render(<ProjectsGrid projects={peu} />);

      await waitFor(() => {
        expect(screen.getAllByTestId('project-card')).toHaveLength(5);
      });
      // Aucune sentinelle : il n'y a rien à charger plus tard.
      expect(screen.queryByTestId('projects-sentinel')).not.toBeInTheDocument();
    });

    it('affiche tout si IntersectionObserver est indisponible (repli)', async () => {
      // Navigateur ancien / environnement sans observer : ne pas emprisonner les projets
      // derrière un mécanisme qui ne se déclenchera jamais.
      // @ts-expect-error — on simule l'absence de l'API.
      delete global.IntersectionObserver;

      render(<ProjectsGrid projects={beaucoup} />);

      await waitFor(() => {
        expect(screen.getAllByTestId('project-card')).toHaveLength(30);
      });
    });
  });

  it('should handle technologies exactly at 3 limit (no overflow)', () => {
    const projectWith3Techs: Project = {
      id: '5',
      title: 'Project With 3 Techs',
      description: 'Test project',
      technologies: ['Tech1', 'Tech2', 'Tech3'],
      featured: false,
    };

    render(<ProjectsGrid projects={[projectWith3Techs]} />);

    expect(screen.getByText('Tech1')).toBeInTheDocument();
    expect(screen.getByText('Tech2')).toBeInTheDocument();
    expect(screen.getByText('Tech3')).toBeInTheDocument();
    // No overflow badge since exactly 3 techs shown
    expect(screen.queryByText(/^\+/)).not.toBeInTheDocument();
  });
});
