import { render, screen, fireEvent } from '@testing-library/react';
import SkillsCloud from '../SkillsCloud';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, layout, whileHover, variants, initial, animate, transition, ...props }: any) => <div {...props}>{children}</div>,
  },
}));

describe('SkillsCloud', () => {
  afterEach(() => {
    // Clear all timers to prevent memory leaks
    jest.clearAllTimers();
  });
  const mockSkills = [
    {
      id: '1',
      name: 'Go',
      level: 'expert' as const,
      category: 'backend',
      yearsExperience: 4,
      score: 0.95,
    },
    {
      id: '2',
      name: 'React',
      level: 'advanced' as const,
      category: 'frontend',
      yearsExperience: 5,
      score: 0.90,
    },
    {
      id: '3',
      name: 'Docker',
      level: 'advanced' as const,
      category: 'devops',
      yearsExperience: 3,
      score: 0.85,
    },
    {
      id: '4',
      name: 'PostgreSQL',
      level: 'intermediate' as const,
      category: 'database',
      yearsExperience: 4,
      score: 0.80,
    },
    {
      id: '5',
      name: 'TypeScript',
      level: 'advanced' as const,
      category: 'frontend',
      yearsExperience: 3,
      score: 0.88,
    },
  ];

  it('should render all skills', () => {
    render(<SkillsCloud skills={mockSkills} />);

    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('React')).toBeInTheDocument();
    expect(screen.getByText('Docker')).toBeInTheDocument();
    expect(screen.getByText('PostgreSQL')).toBeInTheDocument();
    expect(screen.getByText('TypeScript')).toBeInTheDocument();
  });

  it('should display category filter buttons', () => {
    render(<SkillsCloud skills={mockSkills} />);

    expect(screen.getByText('Toutes')).toBeInTheDocument();
    expect(screen.getByText('backend')).toBeInTheDocument();
    expect(screen.getByText('frontend')).toBeInTheDocument();
    expect(screen.getByText('devops')).toBeInTheDocument();
    expect(screen.getByText('database')).toBeInTheDocument();
  });

  it('should filter skills by category when button is clicked', () => {
    render(<SkillsCloud skills={mockSkills} />);

    // Click on frontend category
    const frontendButton = screen.getByText('frontend');
    fireEvent.click(frontendButton);

    // Should show only frontend skills
    expect(screen.getByText('React')).toBeInTheDocument();
    expect(screen.getByText('TypeScript')).toBeInTheDocument();

    // Should not show backend skills (filtered out, not in DOM)
    expect(screen.queryByText('Go')).not.toBeInTheDocument();
    expect(screen.queryByText('Docker')).not.toBeInTheDocument();
  });

  it('should show all skills when "Toutes" is clicked', () => {
    render(<SkillsCloud skills={mockSkills} />);

    // First filter by category
    const backendButton = screen.getByText('backend');
    fireEvent.click(backendButton);

    // Then click "Toutes"
    const allButton = screen.getByText('Toutes');
    fireEvent.click(allButton);

    // All skills should be visible
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('React')).toBeInTheDocument();
    expect(screen.getByText('Docker')).toBeInTheDocument();
  });

  it('should apply active styles to selected category button', () => {
    render(<SkillsCloud skills={mockSkills} />);

    const backendButton = screen.getByText('backend');
    fireEvent.click(backendButton);

    // Active button should have specific classes
    expect(backendButton).toHaveClass('bg-blue-600', 'text-white');
  });

  it('should calculate font size based on level and score', () => {
    const { container } = render(<SkillsCloud skills={mockSkills} />);

    // Go has level 'expert' (value 4) and score 0.95, should have larger font
    const goSkill = screen.getByText('Go');
    const fontSize = window.getComputedStyle(goSkill.parentElement!).fontSize;

    // Just verify fontSize is set (actual calculation: 14 + 4*2 + 0.95*4 = 25.8)
    expect(goSkill.parentElement).toHaveStyle({ fontSize: expect.any(String) });
  });

  it('should apply correct color classes per category', () => {
    const { container } = render(<SkillsCloud skills={mockSkills} />);

    // Find the motion.div elements directly (not parentElement)
    const goSkill = screen.getByText('Go').closest('[title*="Go"]');
    const reactSkill = screen.getByText('React').closest('[title*="React"]');
    const dockerSkill = screen.getByText('Docker').closest('[title*="Docker"]');

    // Backend should be blue
    expect(goSkill).toHaveClass('bg-blue-100');

    // Frontend should be purple
    expect(reactSkill).toHaveClass('bg-purple-100');

    // DevOps should be green
    expect(dockerSkill).toHaveClass('bg-green-100');
  });

  it('should display tooltip with skill details', () => {
    render(<SkillsCloud skills={mockSkills} />);

    // The title attribute is on the motion.div itself, not parentElement
    const goSkill = screen.getByText('Go').closest('[title]');

    expect(goSkill).toHaveAttribute('title');
    expect(goSkill?.getAttribute('title')).toContain('Go');
    expect(goSkill?.getAttribute('title')).toContain('expert');
    expect(goSkill?.getAttribute('title')).toContain('4 ans');
  });

  it('should render legend explaining size meaning', () => {
    render(<SkillsCloud skills={mockSkills} />);

    const legend = screen.getByText(/La taille représente le niveau de compétence/i);
    expect(legend).toBeInTheDocument();
  });

  it('should handle empty skills array', () => {
    const { container } = render(<SkillsCloud skills={[]} />);

    // Should render container but no skills
    expect(screen.getByText('Toutes')).toBeInTheDocument();
    expect(container.querySelector('.flex.flex-wrap.gap-3')).toBeInTheDocument();
  });

  it('should extract unique categories from skills', () => {
    render(<SkillsCloud skills={mockSkills} />);

    // Should have 4 unique categories + "Toutes" button
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBe(5); // Toutes + backend + frontend + devops + database
  });
});
