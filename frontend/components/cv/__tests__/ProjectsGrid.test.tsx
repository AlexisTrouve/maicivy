import { render, screen } from '@testing-library/react';
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

    expect(screen.getByText('Projet Vedette')).toBeInTheDocument();
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

    // "Cliquez pour plus de details" appears on each card
    const hints = screen.getAllByText('Cliquez pour plus de details');
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

    // Technology spans use bg-gray-100 styling
    const techSpans = container.querySelectorAll('.bg-gray-100.dark\\:bg-gray-700.text-gray-600');
    expect(techSpans.length).toBeGreaterThan(0);
    // First project shows Go, React, PostgreSQL (3 visible)
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('React')).toBeInTheDocument();
  });

  it('should apply featured border styling', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    const featuredCard = container.querySelector('.border-yellow-400');
    expect(featuredCard).toBeInTheDocument();
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

  it('should apply dark mode classes', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    const cards = container.querySelectorAll('.dark\\:bg-gray-800');
    expect(cards.length).toBe(3);
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
