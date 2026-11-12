import { render, screen } from '@testing-library/react';
import AnalyticsPage, { generateMetadata } from '../page';

// Mock analytics components
// TestStatsCard lit test-stats.json + i18n → stub pour ce test de LAYOUT (rendu interne testé à part).
jest.mock('@/components/analytics/TestStatsCard', () => {
  return function MockTestStatsCard() {
    return <div data-testid="test-stats-card">Test Stats Card Component</div>;
  };
});

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

// Paramètres par défaut pour le composant serveur async
const defaultParams = { params: Promise.resolve({ locale: 'fr' }) };

describe('AnalyticsPage', () => {
  it('should render main heading', async () => {
    render(await AnalyticsPage(defaultParams));

    // t('title') = "Le site, en direct"
    expect(screen.getByRole('heading', { name: /Le site, en direct/i, level: 1 })).toBeInTheDocument();
  });

  it('should render the lead text', async () => {
    render(await AnalyticsPage(defaultParams));

    expect(screen.getByText(/Statistiques publiques, mises à jour en temps réel/i)).toBeInTheDocument();
  });

  it('should render components in correct order', async () => {
    const { container } = render(await AnalyticsPage(defaultParams));

    const components = container.querySelectorAll('[data-testid]');
    const componentIds = Array.from(components).map((el) => el.getAttribute('data-testid'));

    expect(componentIds).toEqual([
      'test-stats-card',
      'date-filter',
      'stats-overview',
      'realtime-visitors',
      'theme-stats',
      'letters-generated',
      'heatmap',
      'analytics-privacy',
    ]);
  });

  it('places the wide panels full width and the theme/letters pair side by side', async () => {
    render(await AnalyticsPage(defaultParams));

    expect(screen.getByTestId('realtime-visitors').parentElement).toHaveClass('lg:col-span-3');
    expect(screen.getByTestId('theme-stats').parentElement).toHaveClass('lg:col-span-2');
    expect(screen.getByTestId('letters-generated').parentElement).toHaveClass('lg:col-span-1');
    expect(screen.getByTestId('heatmap').parentElement).toHaveClass('lg:col-span-3');
  });

  it('shows the privacy note', async () => {
    render(await AnalyticsPage(defaultParams));

    expect(screen.getByTestId('analytics-privacy')).toHaveTextContent(/anonyme/);
  });
});

describe('AnalyticsPage - Metadata', () => {
  it('uses the page title (the layout template appends the site name)', async () => {
    const metadata = await generateMetadata({ params: Promise.resolve({ locale: 'fr' }) });
    expect(metadata.title).toBe('Statistiques publiques');
  });

  it('should have a description and OpenGraph metadata', async () => {
    const metadata = await generateMetadata({ params: Promise.resolve({ locale: 'fr' }) });
    expect(metadata.description).toContain('temps réel');
    expect(metadata.openGraph?.title).toBe('Statistiques publiques');
  });
});
