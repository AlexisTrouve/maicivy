import { render, screen } from '@testing-library/react';
import ProjectsGrid from '../ProjectsGrid';
import { Project } from '@/lib/types';

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  ExternalLink: ({ className }: any) => <svg data-testid="external-link-icon" className={className} />,
  Github: ({ className }: any) => <svg data-testid="github-icon" className={className} />,
  Star: ({ className }: any) => <svg data-testid="star-icon" className={className} />,
}));

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, initial, whileInView, whileHover, viewport, variants, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
}));

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

  it('should display technologies with max 4 visible', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    // First project has 5 technologies, should show 4 + "+1"
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('React')).toBeInTheDocument();
    expect(screen.getByText('PostgreSQL')).toBeInTheDocument();
    expect(screen.getByText('Redis')).toBeInTheDocument();
    expect(screen.getByText('+1')).toBeInTheDocument();
  });

  it('should display language indicator with color', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    const languages = screen.getAllByText('TypeScript');
    expect(languages.length).toBeGreaterThan(0);

    // Check for language color indicators
    const languageDots = container.querySelectorAll('.w-3.h-3.rounded-full');
    expect(languageDots.length).toBeGreaterThan(0);
  });

  it('should display star counts', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    expect(screen.getByText('150')).toBeInTheDocument();
    expect(screen.getByText('75')).toBeInTheDocument();
  });

  it('should render GitHub links when provided', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    const codeLinks = screen.getAllByText('Code');
    expect(codeLinks).toHaveLength(2); // Only 2 projects have GitHub URLs

    expect(codeLinks[0].closest('a')).toHaveAttribute('href', 'https://github.com/johndoe/ecommerce');
    expect(codeLinks[1].closest('a')).toHaveAttribute('href', 'https://github.com/johndoe/chat');
  });

  it('should render demo links when provided', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    const demoLink = screen.getByText('Demo');
    expect(demoLink).toBeInTheDocument();
    expect(demoLink.closest('a')).toHaveAttribute('href', 'https://demo.ecommerce.com');
  });

  it('should display relevance score as percentage', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    expect(screen.getByText('95%')).toBeInTheDocument();
    expect(screen.getByText('82%')).toBeInTheDocument();
    expect(screen.getByText('70%')).toBeInTheDocument();
  });

  it('should render score progress bar with correct width', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    const progressBars = container.querySelectorAll('.bg-blue-600.rounded-full');
    expect(progressBars.length).toBe(3);

    // First project has 95% score
    expect(progressBars[0]).toHaveStyle({ width: '95%' });
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

  it('should render Pertinence label for scored projects', () => {
    render(<ProjectsGrid projects={mockProjects} />);

    const pertinenceLabels = screen.getAllByText('Pertinence');
    expect(pertinenceLabels.length).toBe(3);
  });

  it('should render external link icons', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    const svgElements = container.querySelectorAll('svg');
    // Should have: Star icons (featured + counts), ExternalLink, Github icons
    expect(svgElements.length).toBeGreaterThan(5);
  });

  it('should apply dark mode classes', () => {
    const { container } = render(<ProjectsGrid projects={mockProjects} />);

    const cards = container.querySelectorAll('.dark\\:bg-gray-800');
    expect(cards.length).toBe(3);
  });

  it('should handle technologies exactly at 4 limit', () => {
    const projectWith4Techs: Project = {
      id: '5',
      title: 'Project With 4 Techs',
      description: 'Test project',
      technologies: ['Tech1', 'Tech2', 'Tech3', 'Tech4'],
      featured: false,
    };

    render(<ProjectsGrid projects={[projectWith4Techs]} />);

    expect(screen.getByText('Tech1')).toBeInTheDocument();
    expect(screen.getByText('Tech2')).toBeInTheDocument();
    expect(screen.getByText('Tech3')).toBeInTheDocument();
    expect(screen.getByText('Tech4')).toBeInTheDocument();
    expect(screen.queryByText(/^\+/)).not.toBeInTheDocument();
  });
});
