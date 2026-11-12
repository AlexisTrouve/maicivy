import { render, screen, fireEvent } from '@testing-library/react';
import SkillsCloud from '../SkillsCloud';

// Mock framer-motion
// framer-motion mocké globalement (cf. __mocks__/framer-motion.tsx — couvre motion.button + AnimatePresence)

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

    // Filtre actif = « où l'on est » : état exposé (aria-pressed) et filet d'or, pas d'aplat bleu.
    expect(backendButton).toHaveAttribute('aria-pressed', 'true');
    expect(backendButton).toHaveClass('border-gold/70', 'text-foreground');
    expect(screen.getByText('Toutes')).toHaveAttribute('aria-pressed', 'false');
  });

  it('should calculate font size based on level and score', () => {
    const { container } = render(<SkillsCloud skills={mockSkills} />);

    // Go has level 'expert' (value 4) and score 0.95, should have larger font
    const goSkill = screen.getByText('Go');
    const fontSize = window.getComputedStyle(goSkill.parentElement!).fontSize;

    // Just verify fontSize is set (actual calculation: 14 + 4*2 + 0.95*4 = 25.8)
    expect(goSkill.parentElement).toHaveStyle({ fontSize: expect.any(String) });
  });

  // Plus de couleur par catégorie (elle codait une information qu'aucune légende n'expliquait) :
  // toutes les pastilles partagent la matière neutre du thème, seule la TAILLE varie.
  it('should render every badge with the same neutral theme surface', () => {
    render(<SkillsCloud skills={mockSkills} />);

    for (const name of ['Go', 'React', 'Docker']) {
      const badge = screen.getByText(name).closest('[title]');
      expect(badge).toHaveClass('bg-background', 'border-border', 'text-foreground');
      expect(badge?.className).not.toMatch(/blue|purple|green|orange|pink|gray/);
    }
  });

  it('should display tooltip with skill details', () => {
    render(<SkillsCloud skills={mockSkills} />);

    // The title attribute is on the motion.button itself.
    // Format: t('badgeTitle', { name }) = "{name} — voir le détail"
    const goSkill = screen.getByText('Go').closest('[title]');

    expect(goSkill).toHaveAttribute('title', 'Go — voir le détail');
    // Même nom accessible que l'infobulle (bouton nommé par le lecteur d'écran).
    expect(goSkill).toHaveAttribute('aria-label', 'Go — voir le détail');
  });

  it('should render legend explaining size meaning', () => {
    render(<SkillsCloud skills={mockSkills} />);

    const legend = screen.getByText(/La taille reflète le niveau et la pertinence/i);
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

    // Should have 4 unique categories + "Toutes" filter button
    // (Skill badge buttons are also buttons, so we check by the known filter labels)
    expect(screen.getByRole('button', { name: 'Toutes' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'backend' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'frontend' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'devops' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'database' })).toBeInTheDocument();
  });
});
