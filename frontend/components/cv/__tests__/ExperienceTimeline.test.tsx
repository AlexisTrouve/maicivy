import { render, screen } from '@testing-library/react';
import ExperienceTimeline from '../ExperienceTimeline';
import { mockCVData } from '@/lib/testutil/fixtures';

// Mock framer-motion to avoid animation issues in tests
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, variants, initial, animate, whileInView, whileHover, viewport, transition, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
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

  it('should render vertical timeline line', () => {
    const { container } = render(<ExperienceTimeline experiences={mockExperiences} />);

    const timelineLine = container.querySelector('.bg-gradient-to-b');
    expect(timelineLine).toBeInTheDocument();
  });

  it('should render timeline dots for each experience', () => {
    const { container } = render(<ExperienceTimeline experiences={mockExperiences} />);

    const dots = container.querySelectorAll('.rounded-full.bg-blue-600');
    expect(dots.length).toBe(mockExperiences.length);
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
    const { container } = render(<ExperienceTimeline experiences={[]} />);

    // Should render container but no items
    expect(container.querySelector('.space-y-12')).toBeInTheDocument();
    expect(container.querySelectorAll('.bg-white.dark\\:bg-gray-800').length).toBe(0);
  });
});
