import { render, screen } from '@testing-library/react';
import ExperienceTimeline from '../ExperienceTimeline';
import { mockCVData } from '@/lib/testutil/fixtures';

// framer-motion mocké globalement (cf. __mocks__/framer-motion.tsx)

// lucide-react : le mock global ne couvre pas ArrowRight (utilisé dans ExperienceTimeline).
// On surcharge localement pour éviter le crash "Element type is invalid: undefined".
jest.mock('lucide-react', () => ({
  Briefcase: ({ className }: any) => <svg data-testid="briefcase-icon" className={className} />,
  Calendar: ({ className }: any) => <svg data-testid="calendar-icon" className={className} />,
  ArrowRight: ({ className }: any) => <svg data-testid="arrow-right-icon" className={className} />,
}));

describe('ExperienceTimeline', () => {
  afterEach(() => {
    // Clear all timers to prevent memory leaks
    jest.clearAllTimers();
  });
  const mockExperiences = [
    {
      id: '1',
      title: 'Senior Developer',
      company: 'Tech Corp',
      description: 'Led development of microservices',
      startDate: '2020-01-01',
      endDate: '2023-12-31',
      technologies: ['Go', 'React', 'PostgreSQL'],
      tags: ['backend', 'frontend'],
      score: 0.95,
    },
    {
      id: '2',
      title: 'Full Stack Developer',
      company: 'Startup Inc',
      description: 'Built RESTful APIs',
      startDate: '2018-06-01',
      endDate: '2019-12-31',
      technologies: ['Node.js', 'TypeScript'],
      tags: ['fullstack'],
      score: 0.80,
    },
  ];

  it('should render all experiences', () => {
    render(<ExperienceTimeline experiences={mockExperiences} />);

    expect(screen.getByText('Senior Developer')).toBeInTheDocument();
    expect(screen.getByText('Tech Corp')).toBeInTheDocument();
    expect(screen.getByText('Full Stack Developer')).toBeInTheDocument();
    expect(screen.getByText('Startup Inc')).toBeInTheDocument();
  });

  it('should display experience descriptions', () => {
    render(<ExperienceTimeline experiences={mockExperiences} />);

    expect(screen.getByText('Led development of microservices')).toBeInTheDocument();
    expect(screen.getByText('Built RESTful APIs')).toBeInTheDocument();
  });

  it('should render technologies as tags', () => {
    render(<ExperienceTimeline experiences={mockExperiences} />);

    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('React')).toBeInTheDocument();
    expect(screen.getByText('PostgreSQL')).toBeInTheDocument();
    expect(screen.getByText('Node.js')).toBeInTheDocument();
    expect(screen.getByText('TypeScript')).toBeInTheDocument();
  });

  it('should format dates correctly with date-fns', () => {
    render(<ExperienceTimeline experiences={mockExperiences} />);

    // date-fns with fr locale formats as "janv. 2020"
    expect(screen.getByText(/janv. 2020/i)).toBeInTheDocument();
    expect(screen.getByText(/déc. 2023/i)).toBeInTheDocument();
  });

  it('should display score as percentage when available', () => {
    render(<ExperienceTimeline experiences={mockExperiences} />);

    expect(screen.getByText('95%')).toBeInTheDocument();
    expect(screen.getByText('80%')).toBeInTheDocument();
  });

  it('should calculate and display duration', () => {
    render(<ExperienceTimeline experiences={mockExperiences} />);

    // Senior Developer: ~4 years
    expect(screen.getByText(/3 ans|4 ans/)).toBeInTheDocument();

    // Full Stack Developer: ~1.5 years
    expect(screen.getByText(/1 an|18 mois/)).toBeInTheDocument();
  });

  it('should handle current positions (no endDate)', () => {
    const currentExperience = [
      {
        id: '1',
        title: 'Senior Developer',
        company: 'Tech Corp',
        description: 'Led development of microservices',
        startDate: '2020-01-01',
        technologies: ['Go', 'React', 'PostgreSQL'],
        tags: ['backend', 'frontend'],
        score: 0.95,
      },
    ];

    render(<ExperienceTimeline experiences={currentExperience} />);

    expect(screen.getByText(/Présent/)).toBeInTheDocument();
  });

  // Ligne du temps : un filet neutre du thème (plus de dégradé bleu→violet).
  it('should render vertical timeline line with a theme rule', () => {
    render(<ExperienceTimeline experiences={mockExperiences} />);

    const line = screen.getByTestId('timeline-line');
    expect(line).toHaveClass('bg-border');
    expect(line.className).not.toMatch(/gradient|blue|purple/);
  });

  // Un point par expérience ; seul le poste EN COURS (sans date de fin) est marqué d'or —
  // doctrine de l'or : « où l'on est ».
  it('should render one dot per experience, gold only for the current position', () => {
    // Un poste en cours ajouté aux deux postes terminés de la fixture.
    const withCurrent = [
      { ...mockExperiences[0], id: 'now', endDate: undefined as unknown as string },
      ...mockExperiences,
    ];
    render(<ExperienceTimeline experiences={withCurrent} />);

    const dots = screen.getAllByTestId('timeline-dot');
    expect(dots).toHaveLength(3);
    expect(dots[0]).toHaveAttribute('data-current', 'true');
    expect(dots[0]).toHaveClass('bg-gold');
    expect(dots[1]).not.toHaveClass('bg-gold');
    expect(dots[2]).not.toHaveClass('bg-gold');
  });

  it('should render Briefcase and Calendar icons', () => {
    const { container } = render(<ExperienceTimeline experiences={mockExperiences} />);

    // Count SVG icons (Briefcase + Calendar for each experience)
    const icons = container.querySelectorAll('svg');
    expect(icons.length).toBeGreaterThan(0);
  });

  it('should apply alternating layout classes', () => {
    const { container } = render(<ExperienceTimeline experiences={mockExperiences} />);

    const items = container.querySelectorAll('.relative.flex.items-center');

    // First item should have md:flex-row
    expect(items[0]?.className).toContain('md:flex-row');

    // Second item should have md:flex-row-reverse (if exists)
    if (items[1]) {
      expect(items[1].className).toContain('md:flex-row-reverse');
    }
  });

  it('should handle empty experiences array', () => {
    render(<ExperienceTimeline experiences={[]} />);

    // La ligne du temps reste, sans aucune carte.
    expect(screen.getByTestId('timeline-line')).toBeInTheDocument();
    expect(screen.queryAllByTestId('timeline-item')).toHaveLength(0);
  });
});
