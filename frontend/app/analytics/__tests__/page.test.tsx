import { render, screen } from '@testing-library/react';
import AnalyticsPage, { metadata } from '../page';

// Mock analytics components
jest.mock('@/components/analytics/RealtimeVisitors', () => {
  return function MockRealtimeVisitors() {
    return <div data-testid="realtime-visitors">Realtime Visitors Component</div>;
  };
});

jest.mock('@/components/analytics/ThemeStats', () => {
  return function MockThemeStats() {
    return <div data-testid="theme-stats">Theme Stats Component</div>;
  };
});

jest.mock('@/components/analytics/LettersGenerated', () => {
  return function MockLettersGenerated() {
    return <div data-testid="letters-generated">Letters Generated Component</div>;
  };
});

jest.mock('@/components/analytics/Heatmap', () => {
  return function MockHeatmap() {
    return <div data-testid="heatmap">Heatmap Component</div>;
  };
});

jest.mock('@/components/analytics/DateFilter', () => {
  return function MockDateFilter() {
    return <div data-testid="date-filter">Date Filter Component</div>;
  };
});

jest.mock('@/components/analytics/StatsOverview', () => {
  return function MockStatsOverview() {
    return <div data-testid="stats-overview">Stats Overview Component</div>;
  };
});

// Mock Suspense since we're testing the layout
jest.mock('react', () => ({
  ...jest.requireActual('react'),
  Suspense: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

describe('AnalyticsPage', () => {
  it('should render main heading', () => {
    render(<AnalyticsPage />);

    expect(
      screen.getByRole('heading', { name: /Analytics Dashboard/i, level: 1 })
    ).toBeInTheDocument();
  });

  it('should render description text', () => {
    render(<AnalyticsPage />);

    expect(
      screen.getByText(/Statistiques publiques en temps réel/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Découvrez comment les visiteurs interagissent/i)
    ).toBeInTheDocument();
  });

  it('should render DateFilter component', () => {
    render(<AnalyticsPage />);

    expect(screen.getByTestId('date-filter')).toBeInTheDocument();
  });

  it('should render StatsOverview component', () => {
    render(<AnalyticsPage />);

    expect(screen.getByTestId('stats-overview')).toBeInTheDocument();
  });

  it('should render RealtimeVisitors component', () => {
    render(<AnalyticsPage />);

    expect(screen.getByTestId('realtime-visitors')).toBeInTheDocument();
  });

  it('should render ThemeStats component', () => {
    render(<AnalyticsPage />);

    expect(screen.getByTestId('theme-stats')).toBeInTheDocument();
  });

  it('should render LettersGenerated component', () => {
    render(<AnalyticsPage />);

    expect(screen.getByTestId('letters-generated')).toBeInTheDocument();
  });

  it('should render Heatmap component', () => {
    render(<AnalyticsPage />);

    expect(screen.getByTestId('heatmap')).toBeInTheDocument();
  });

  it('should render footer note about privacy', () => {
    render(<AnalyticsPage />);

    expect(
      screen.getByText(/Ce dashboard est public et mis à jour en temps réel/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/données sont collectées de manière anonyme/i)
    ).toBeInTheDocument();
  });

  it('should have proper grid layout', () => {
    const { container } = render(<AnalyticsPage />);

    const grid = container.querySelector('.grid.grid-cols-1.lg\\:grid-cols-3');
    expect(grid).toBeInTheDocument();
  });

  it('should render container with proper classes', () => {
    const { container } = render(<AnalyticsPage />);

    const containerDiv = container.querySelector('.container');
    expect(containerDiv).toBeInTheDocument();
    expect(containerDiv).toHaveClass('mx-auto', 'px-4', 'py-8');
  });

  it('should have RealtimeVisitors in full width section', () => {
    const { container } = render(<AnalyticsPage />);

    const realtimeVisitors = screen.getByTestId('realtime-visitors');
    const parentDiv = realtimeVisitors.parentElement;

    expect(parentDiv).toHaveClass('lg:col-span-3');
  });

  it('should have ThemeStats in 2-column section', () => {
    const { container } = render(<AnalyticsPage />);

    const themeStats = screen.getByTestId('theme-stats');
    const parentDiv = themeStats.parentElement;

    expect(parentDiv).toHaveClass('lg:col-span-2');
  });

  it('should have LettersGenerated in 1-column section', () => {
    const { container } = render(<AnalyticsPage />);

    const lettersGenerated = screen.getByTestId('letters-generated');
    const parentDiv = lettersGenerated.parentElement;

    expect(parentDiv).toHaveClass('lg:col-span-1');
  });

  it('should have Heatmap in full width section', () => {
    const { container } = render(<AnalyticsPage />);

    const heatmap = screen.getByTestId('heatmap');
    const parentDiv = heatmap.parentElement;

    expect(parentDiv).toHaveClass('lg:col-span-3');
  });

  it('should render components in correct order', () => {
    const { container } = render(<AnalyticsPage />);

    const components = container.querySelectorAll('[data-testid]');
    const componentIds = Array.from(components).map(
      (el) => el.getAttribute('data-testid')
    );

    expect(componentIds).toEqual([
      'date-filter',
      'stats-overview',
      'realtime-visitors',
      'theme-stats',
      'letters-generated',
      'heatmap',
    ]);
  });

  it('should have header with proper spacing', () => {
    const { container } = render(<AnalyticsPage />);

    const header = container.querySelector('.mb-8');
    expect(header).toBeInTheDocument();
  });

  it('should render footer with proper styling', () => {
    const { container } = render(<AnalyticsPage />);

    const footer = container.querySelector('.mt-8.p-4.border.rounded-lg');
    expect(footer).toBeInTheDocument();
  });

  it('should have responsive grid gaps', () => {
    const { container } = render(<AnalyticsPage />);

    const grid = container.querySelector('.grid');
    expect(grid).toHaveClass('gap-6');
  });
});

describe('AnalyticsPage - Metadata', () => {
  it('should have correct title', () => {
    expect(metadata.title).toBe('Analytics Dashboard - maicivy');
  });

  it('should have correct description', () => {
    expect(metadata.description).toContain(
      'Dashboard analytics en temps réel'
    );
    expect(metadata.description).toContain('CV interactif avec IA');
  });

  it('should have OpenGraph metadata', () => {
    expect(metadata.openGraph).toBeDefined();
    expect(metadata.openGraph?.title).toBe('Analytics Dashboard - maicivy');
    expect(metadata.openGraph?.description).toContain(
      'Statistiques publiques en temps réel'
    );
  });
});
