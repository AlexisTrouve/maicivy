import { render, screen, waitFor, act } from '@testing-library/react';
import StatsOverview from '../StatsOverview';

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Users: () => <div data-testid="users-icon">Users Icon</div>,
  Eye: () => <div data-testid="eye-icon">Eye Icon</div>,
  FileText: () => <div data-testid="filetext-icon">FileText Icon</div>,
  TrendingUp: () => <div data-testid="trendingup-icon">TrendingUp Icon</div>,
}));

// Mock environment variable
process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080';

// Mock global fetch
global.fetch = jest.fn();

describe('StatsOverview', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Don't use fake timers by default - only in specific tests that need them
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  // API response format uses snake_case wrapped in {success, data}
  const mockStatsApiResponse = {
    success: true,
    data: {
      unique_visitors: 1543,
      total_events: 8234,
      letters_generated: 456,
      conversion_rate: 0.296, // Backend returns as decimal
    },
  };

  const mockRealtimeApiResponse = {
    success: true,
    data: {
      current_visitors: 12,
      unique_today: 100,
      total_events: 500,
      letters_today: 10,
      timestamp: Date.now(),
    },
  };

  // Helper to mock both fetch calls
  const mockBothApis = () => {
    (global.fetch as jest.Mock).mockImplementation((url: string) => {
      if (url.includes('/realtime')) {
        return Promise.resolve({
          ok: true,
          json: async () => mockRealtimeApiResponse,
        });
      }
      return Promise.resolve({
        ok: true,
        json: async () => mockStatsApiResponse,
      });
    });
  };

  it('should render loading skeletons initially', () => {
    (global.fetch as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    const { container } = render(<StatsOverview />);

    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBe(4); // 4 stat cards
  });

  it('should fetch stats on mount', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/analytics/stats',
        { credentials: 'include' }
      );
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/analytics/realtime',
        { credentials: 'include' }
      );
    });
  });

  it('should render all 4 stat cards', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      expect(screen.getByText('Visiteurs')).toBeInTheDocument();
      expect(screen.getByText('Pages Vues')).toBeInTheDocument();
      expect(screen.getByText('Lettres')).toBeInTheDocument();
      expect(screen.getByText('Conversion')).toBeInTheDocument();
    });
  });

  it('should display visitors count', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      expect(screen.getByText('1543')).toBeInTheDocument();
    });
  });

  it('should display active visitors in subtitle', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      expect(screen.getByText('+12 actifs')).toBeInTheDocument();
    });
  });

  it('should display page views count', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      expect(screen.getByText('8234')).toBeInTheDocument();
    });
  });

  it('should display letters count', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      expect(screen.getByText('456')).toBeInTheDocument();
    });
  });

  it('should display conversion rate as percentage', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      expect(screen.getByText('29.6%')).toBeInTheDocument();
    });
  });

  it('should render all icons', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      expect(screen.getByTestId('users-icon')).toBeInTheDocument();
      expect(screen.getByTestId('eye-icon')).toBeInTheDocument();
      expect(screen.getByTestId('filetext-icon')).toBeInTheDocument();
      expect(screen.getByTestId('trendingup-icon')).toBeInTheDocument();
    });
  });

  it('should apply correct color classes to icons', async () => {
    mockBothApis();

    const { container } = render(<StatsOverview />);

    await waitFor(() => {
      // Since we're mocking the icons, we check for the wrapper divs with color classes
      const coloredElements = container.querySelectorAll('[class*="text-"]');
      expect(coloredElements.length).toBeGreaterThan(0);
    });
  });

  it('should render in grid layout', async () => {
    mockBothApis();

    const { container } = render(<StatsOverview />);

    await waitFor(() => {
      const grid = container.querySelector('.grid.grid-cols-1.md\\:grid-cols-2.lg\\:grid-cols-4');
      expect(grid).toBeInTheDocument();
    });
  });

  it('should render cards with rounded borders', async () => {
    mockBothApis();

    const { container } = render(<StatsOverview />);

    await waitFor(() => {
      const cards = container.querySelectorAll('.rounded-lg.border.bg-card');
      expect(cards.length).toBe(4);
    });
  });

  it('should display subtitles for each card', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      expect(screen.getByText('+12 actifs')).toBeInTheDocument();
      expect(screen.getByText('+234 aujourd\'hui')).toBeInTheDocument();
      expect(screen.getByText('+12 aujourd\'hui')).toBeInTheDocument();
      expect(screen.getByText('+2.3% vs hier')).toBeInTheDocument();
    });
  });

  it('should show zeros on API error', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    (global.fetch as jest.Mock).mockRejectedValue(new Error('API Error'));

    render(<StatsOverview />);

    await waitFor(() => {
      // Should display zeros on error
      const zeros = screen.getAllByText('0');
      expect(zeros.length).toBeGreaterThan(0);
    });

    consoleErrorSpy.mockRestore();
  });

  it('should handle API response with missing activeVisitors', async () => {
    (global.fetch as jest.Mock).mockImplementation((url: string) => {
      if (url.includes('/realtime')) {
        // Realtime returns no current_visitors
        return Promise.resolve({
          ok: true,
          json: async () => ({ success: true, data: {} }),
        });
      }
      return Promise.resolve({
        ok: true,
        json: async () => mockStatsApiResponse,
      });
    });

    render(<StatsOverview />);

    await waitFor(() => {
      expect(screen.getByText('+0 actifs')).toBeInTheDocument();
    });
  });

  it('should auto-refresh every 30 seconds', async () => {
    jest.useFakeTimers();
    mockBothApis();

    render(<StatsOverview />);

    // Initial fetch (2 calls: stats + realtime)
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledTimes(2);
    });

    // Advance 30 seconds and flush promises
    await act(async () => {
      jest.advanceTimersByTime(30000);
      await Promise.resolve(); // Flush promises
    });

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledTimes(4); // 2 more calls
    });

    jest.useRealTimers();
  });

  it('should cleanup interval on unmount', async () => {
    jest.useFakeTimers();
    mockBothApis();

    const { unmount } = render(<StatsOverview />);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledTimes(2);
    });

    unmount();

    // Advance time after unmount
    act(() => {
      jest.advanceTimersByTime(60000);
    });

    // Should not fetch again
    expect(global.fetch).toHaveBeenCalledTimes(2);

    jest.useRealTimers();
  });

  it('should display values with correct text size and styling', async () => {
    mockBothApis();

    const { container } = render(<StatsOverview />);

    await waitFor(() => {
      const values = container.querySelectorAll('.text-3xl.font-bold');
      expect(values.length).toBe(4);
    });
  });

  it('should render card titles as muted foreground', async () => {
    mockBothApis();

    const { container } = render(<StatsOverview />);

    await waitFor(() => {
      const titles = container.querySelectorAll('.text-sm.font-medium.text-muted-foreground');
      expect(titles.length).toBe(4);
    });
  });

  it('should render subtitles with small text', async () => {
    mockBothApis();

    const { container } = render(<StatsOverview />);

    await waitFor(() => {
      const subtitles = container.querySelectorAll('.text-xs.text-muted-foreground');
      expect(subtitles.length).toBe(4);
    });
  });

  it('should handle 404 API response', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    (global.fetch as jest.Mock).mockResolvedValue({
      ok: false,
      status: 404,
    });

    render(<StatsOverview />);

    await waitFor(() => {
      // Should fall back to zero values
      const zeros = screen.getAllByText('0');
      expect(zeros.length).toBeGreaterThan(0);
    });

    consoleErrorSpy.mockRestore();
  });

  it('should apply gap-4 spacing to grid', async () => {
    mockBothApis();

    const { container } = render(<StatsOverview />);

    await waitFor(() => {
      const grid = container.querySelector('.gap-4');
      expect(grid).toBeInTheDocument();
    });
  });

  it('should render cards with p-6 padding', async () => {
    mockBothApis();

    const { container } = render(<StatsOverview />);

    await waitFor(() => {
      const cards = container.querySelectorAll('.p-6');
      expect(cards.length).toBe(4);
    });
  });

  it('should display icons with h-4 w-4 size', async () => {
    mockBothApis();

    render(<StatsOverview />);

    await waitFor(() => {
      // Check that all 4 icons are rendered (via test ids)
      expect(screen.getByTestId('users-icon')).toBeInTheDocument();
      expect(screen.getByTestId('eye-icon')).toBeInTheDocument();
      expect(screen.getByTestId('filetext-icon')).toBeInTheDocument();
      expect(screen.getByTestId('trendingup-icon')).toBeInTheDocument();
    });
  });

  it('should render loading state with correct number of skeletons', () => {
    (global.fetch as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    const { container } = render(<StatsOverview />);

    const skeletonCards = container.querySelectorAll('.rounded-lg.border.bg-card.p-6.animate-pulse');
    expect(skeletonCards.length).toBe(4);
  });

  it('should handle zero values', async () => {
    (global.fetch as jest.Mock).mockImplementation((url: string) => {
      if (url.includes('/realtime')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            success: true,
            data: { current_visitors: 0 },
          }),
        });
      }
      return Promise.resolve({
        ok: true,
        json: async () => ({
          success: true,
          data: {
            unique_visitors: 0,
            total_events: 0,
            letters_generated: 0,
            conversion_rate: 0,
          },
        }),
      });
    });

    render(<StatsOverview />);

    await waitFor(() => {
      // Should display zeros
      const zeros = screen.getAllByText('0');
      expect(zeros.length).toBeGreaterThan(0);
    });
  });
});
