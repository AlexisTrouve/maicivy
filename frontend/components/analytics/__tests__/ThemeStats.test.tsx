import { render, screen, waitFor, cleanup } from '@testing-library/react';
import ThemeStats from '../ThemeStats';
import { server } from '@/__mocks__/server';
import { rest } from 'msw';

// Mock lucide-react
jest.mock('lucide-react', () => ({
  BarChart3: () => <svg data-testid="barchart-icon">BarChart Icon</svg>,
}));

// Setup MSW
beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  jest.clearAllTimers();
  cleanup(); // Clean up rendered components
  jest.useRealTimers(); // Always reset timers
});
afterAll(() => server.close());

describe('ThemeStats', () => {
  // Mock data matching actual API format: { success: true, data: [...] }
  // Backend returns 'views' field (not 'count')
  const mockThemeStats = {
    success: true,
    data: [
      { theme: 'backend', views: 523 },
      { theme: 'full-stack', views: 312 },
      { theme: 'devops', views: 198 },
      { theme: 'ai', views: 167 },
      { theme: 'mobile', views: 89 },
    ],
  };

  it('should render loading state initially', () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', async (req, res, ctx) => {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        return res(ctx.json(mockThemeStats));
      })
    );

    render(<ThemeStats />);

    const skeleton = document.querySelector('.animate-pulse');
    expect(skeleton).toBeInTheDocument();
  });

  it('should fetch and display theme statistics', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json(mockThemeStats));
      })
    );

    render(<ThemeStats />);

    await waitFor(() => {
      expect(screen.getByText('backend')).toBeInTheDocument();
    });

    expect(screen.getByText('full-stack')).toBeInTheDocument();
    expect(screen.getByText('devops')).toBeInTheDocument();
    expect(screen.getByText('ai')).toBeInTheDocument();
    expect(screen.getByText('mobile')).toBeInTheDocument();
  });

  it('should display view counts for each theme', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json(mockThemeStats));
      })
    );

    render(<ThemeStats />);

    await waitFor(() => {
      expect(screen.getByText('523 vues')).toBeInTheDocument();
    });

    expect(screen.getByText('312 vues')).toBeInTheDocument();
    expect(screen.getByText('198 vues')).toBeInTheDocument();
    expect(screen.getByText('167 vues')).toBeInTheDocument();
    expect(screen.getByText('89 vues')).toBeInTheDocument();
  });

  it('should display percentages for each theme', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json(mockThemeStats));
      })
    );

    render(<ThemeStats />);

    // Percentages are calculated by component: count/total*100
    // Total = 523+312+198+167+89 = 1289
    await waitFor(() => {
      expect(screen.getByText('40.6%')).toBeInTheDocument();
    });

    expect(screen.getByText('24.2%')).toBeInTheDocument();
    expect(screen.getByText('15.4%')).toBeInTheDocument();
    expect(screen.getByText('13.0%')).toBeInTheDocument();
    expect(screen.getByText('6.9%')).toBeInTheDocument();
  });

  it('should render progress bars for each theme', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json(mockThemeStats));
      })
    );

    const { container } = render(<ThemeStats />);

    await waitFor(() => {
      const progressBars = container.querySelectorAll('.h-2.overflow-hidden');
      expect(progressBars.length).toBe(5);
    });
  });

  it('should apply correct color class for backend theme', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json(mockThemeStats));
      })
    );

    const { container } = render(<ThemeStats />);

    await waitFor(() => {
      const backendBar = container.querySelector('.bg-blue-500');
      expect(backendBar).toBeInTheDocument();
    });
  });

  it('should display ranking numbers', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json(mockThemeStats));
      })
    );

    render(<ThemeStats />);

    await waitFor(() => {
      expect(screen.getByText('#1')).toBeInTheDocument();
    });

    expect(screen.getByText('#2')).toBeInTheDocument();
    expect(screen.getByText('#3')).toBeInTheDocument();
    expect(screen.getByText('#4')).toBeInTheDocument();
    expect(screen.getByText('#5')).toBeInTheDocument();
  });

  it('should render BarChart icon', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json(mockThemeStats));
      })
    );

    const { container } = render(<ThemeStats />);

    await waitFor(() => {
      const icon = container.querySelector('svg');
      expect(icon).toBeInTheDocument();
    });
  });

  it('should display update frequency message', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json(mockThemeStats));
      })
    );

    render(<ThemeStats />);

    await waitFor(() => {
      expect(
        screen.getByText(/Mise à jour toutes les 30 secondes/i)
      ).toBeInTheDocument();
    });
  });

  it('should handle API error gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json(
          { error: 'Internal server error' }
        ));
      })
    );

    render(<ThemeStats />);

    await waitFor(() => {
      // Should not be in loading state anymore
      const skeleton = document.querySelector('.animate-pulse');
      expect(skeleton).not.toBeInTheDocument();
    });

    consoleErrorSpy.mockRestore();
  });

  it('should capitalize theme names', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json({
          success: true,
          data: [{ theme: 'backend', views: 100 }],
        }));
      })
    );

    const { container } = render(<ThemeStats />);

    await waitFor(() => {
      const themeElement = container.querySelector('.capitalize');
      expect(themeElement).toBeInTheDocument();
    });
  });

  it('should set correct width for progress bars based on percentage', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json({
          success: true,
          data: [{ theme: 'backend', views: 100 }],
        }));
      })
    );

    const { container } = render(<ThemeStats />);

    await waitFor(() => {
      const progressBar = container.querySelector('.bg-blue-500');
      // With single item, percentage is 100%
      expect(progressBar).toHaveStyle({ width: '100%' });
    });
  });

  it('should apply transition classes to progress bars', async () => {
    server.use(
      rest.get('*/api/v1/analytics/themes', (req, res, ctx) => {
        return res(ctx.json(mockThemeStats));
      })
    );

    const { container } = render(<ThemeStats />);

    await waitFor(() => {
      const progressBar = container.querySelector('.transition-all');
      expect(progressBar).toBeInTheDocument();
    });
  });
});
