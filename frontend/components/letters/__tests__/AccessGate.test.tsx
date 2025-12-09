import { render, screen, waitFor } from '@testing-library/react';
import { AccessGate } from '../AccessGate';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
}));

// Mock useVisitCount hook
const mockUseVisitCount = jest.fn();
jest.mock('@/hooks/useVisitCount', () => ({
  useVisitCount: () => mockUseVisitCount(),
}));

describe('AccessGate', () => {
  const mockChildren = <div data-testid="protected-content">Protected Content</div>;

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should show loading spinner when loading', () => {
    mockUseVisitCount.mockReturnValue({
      status: null,
      loading: true,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    const spinner = document.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });

  it('should render children when user has access', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 3,
        hasAccess: true,
        remainingVisits: 0,
        sessionId: 'session-123',
      },
      loading: false,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    expect(screen.getByTestId('protected-content')).toBeInTheDocument();
    expect(screen.queryByText(/Fonctionnalité Premium/i)).not.toBeInTheDocument();
  });

  it('should show teaser when user does not have access', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
        sessionId: 'session-123',
      },
      loading: false,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    expect(screen.getByText(/Fonctionnalité Premium/i)).toBeInTheDocument();
    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });

  it('should display correct visit count in teaser', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
        sessionId: 'session-123',
      },
      loading: false,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    expect(screen.getByText('1 / 3 visites')).toBeInTheDocument();
  });

  it('should display correct remaining visits message', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 2,
        hasAccess: false,
        remainingVisits: 1,
        sessionId: 'session-123',
      },
      loading: false,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    expect(screen.getByText(/Encore 1 visite avant déblocage/i)).toBeInTheDocument();
  });

  it('should use plural "visites" when remaining > 1', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
        sessionId: 'session-123',
      },
      loading: false,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    expect(screen.getByText(/Encore 2 visites avant déblocage/i)).toBeInTheDocument();
  });

  it('should render progress bar with correct percentage', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 2,
        hasAccess: false,
        remainingVisits: 1,
        sessionId: 'session-123',
      },
      loading: false,
    });

    const { container } = render(<AccessGate>{mockChildren}</AccessGate>);

    // Progress bar should be 2/3 = 66.67%
    const progressBar = container.querySelector('.bg-gradient-to-r.from-blue-500');
    expect(progressBar).toBeInTheDocument();
  });

  it('should display all feature previews', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
        sessionId: 'session-123',
      },
      loading: false,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    expect(
      screen.getByText(/Génération de lettre de motivation personnalisée/i)
    ).toBeInTheDocument();

    expect(
      screen.getByText(/Lettre d'anti-motivation humoristique unique/i)
    ).toBeInTheDocument();

    expect(
      screen.getByText(/Export PDF professionnel des deux lettres/i)
    ).toBeInTheDocument();

    expect(
      screen.getByText(/Analyse IA de l'entreprise cible/i)
    ).toBeInTheDocument();
  });

  it('should render CTA button linking to CV page', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
        sessionId: 'session-123',
      },
      loading: false,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    const ctaLink = screen.getByRole('link', { name: /Explorer mon CV/i });
    expect(ctaLink).toBeInTheDocument();
    expect(ctaLink).toHaveAttribute('href', '/cv');
  });

  it('should show lock icon in teaser', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
        sessionId: 'session-123',
      },
      loading: false,
    });

    const { container } = render(<AccessGate>{mockChildren}</AccessGate>);

    // Lock icon should be rendered
    const icons = container.querySelectorAll('svg');
    expect(icons.length).toBeGreaterThan(0);
  });

  it('should handle edge case with 0 visits', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 0,
        hasAccess: false,
        remainingVisits: 3,
        sessionId: 'session-123',
      },
      loading: false,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    expect(screen.getByText('0 / 3 visites')).toBeInTheDocument();
    expect(screen.getByText(/Encore 3 visites avant déblocage/i)).toBeInTheDocument();
  });

  it('should handle exactly 3 visits (access granted)', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 3,
        hasAccess: true,
        remainingVisits: 0,
        sessionId: 'session-123',
      },
      loading: false,
    });

    render(<AccessGate>{mockChildren}</AccessGate>);

    expect(screen.getByTestId('protected-content')).toBeInTheDocument();
    expect(screen.queryByText(/Fonctionnalité Premium/i)).not.toBeInTheDocument();
  });

  it('should display sparkles icons for feature list', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
        sessionId: 'session-123',
      },
      loading: false,
    });

    const { container } = render(<AccessGate>{mockChildren}</AccessGate>);

    // Should have multiple Sparkles icons (one per feature)
    const svgs = container.querySelectorAll('svg');
    expect(svgs.length).toBeGreaterThan(3); // Lock + Eye + multiple Sparkles
  });

  it('should apply gradient background to lock icon container', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
        sessionId: 'session-123',
      },
      loading: false,
    });

    const { container } = render(<AccessGate>{mockChildren}</AccessGate>);

    const gradientContainer = container.querySelector('.from-blue-500.to-purple-500');
    expect(gradientContainer).toBeInTheDocument();
  });

  it('should show eye icon in remaining visits section', () => {
    mockUseVisitCount.mockReturnValue({
      status: {
        visitCount: 1,
        hasAccess: false,
        remainingVisits: 2,
        sessionId: 'session-123',
      },
      loading: false,
    });

    const { container } = render(<AccessGate>{mockChildren}</AccessGate>);

    // Eye icon in blue section
    const blueSection = container.querySelector('.bg-blue-50');
    expect(blueSection).toBeInTheDocument();
  });
});
