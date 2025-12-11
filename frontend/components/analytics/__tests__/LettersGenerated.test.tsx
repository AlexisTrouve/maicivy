import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import LettersGenerated from '../LettersGenerated';

// Mock environment variable
process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080';

// Mock global fetch
global.fetch = jest.fn();

describe('LettersGenerated', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  const mockLettersData = {
    total: 456,
    history: [
      { date: '2024-12-01', count: 12 },
      { date: '2024-12-02', count: 18 },
      { date: '2024-12-03', count: 15 },
      { date: '2024-12-04', count: 22 },
      { date: '2024-12-05', count: 28 },
      { date: '2024-12-06', count: 25 },
      { date: '2024-12-07', count: 30 },
    ],
  };

  it('should render loading skeleton initially', () => {
    (global.fetch as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    const { container } = render(<LettersGenerated />);

    expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
  });

  it('should fetch letters stats on mount', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/analytics/letters?period=week',
        { credentials: 'include' }
      );
    });
  });

  it('should render component title', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      expect(screen.getByText('Lettres IA')).toBeInTheDocument();
    });
  });

  it('should render FileText icon', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    const { container } = render(<LettersGenerated />);

    await waitFor(() => {
      const fileIcon = container.querySelector('svg');
      expect(fileIcon).toBeInTheDocument();
    });
  });

  it('should display total letters count', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      expect(screen.getByText('456')).toBeInTheDocument();
      expect(screen.getByText('lettres générées au total')).toBeInTheDocument();
    });
  });

  it('should render period selector buttons', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      expect(screen.getByText('Jour')).toBeInTheDocument();
      expect(screen.getByText('Semaine')).toBeInTheDocument();
      expect(screen.getByText('Mois')).toBeInTheDocument();
    });
  });

  it('should have "Semaine" selected by default', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      const semaineButton = screen.getByText('Semaine');
      expect(semaineButton).toHaveClass('bg-primary', 'text-primary-foreground');
    });
  });

  it('should change period when button clicked', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      expect(screen.getByText('Semaine')).toBeInTheDocument();
    });

    const jourButton = screen.getByText('Jour');
    fireEvent.click(jourButton);

    await waitFor(() => {
      expect(jourButton).toHaveClass('bg-primary', 'text-primary-foreground');
    });

    // Should trigger new fetch with updated period
    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/analytics/letters?period=day',
      { credentials: 'include' }
    );
  });

  it('should render SVG chart', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    const { container } = render(<LettersGenerated />);

    await waitFor(() => {
      // Select the chart SVG specifically (not the icon SVG)
      const svg = container.querySelector('.h-48 svg');
      expect(svg).toBeInTheDocument();
      expect(svg).toHaveAttribute('viewBox', '0 0 400 150');
    });
  });

  it('should render grid lines in chart', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    const { container } = render(<LettersGenerated />);

    await waitFor(() => {
      // Select lines from the chart SVG only (not from the icon)
      const chartSvg = container.querySelector('.h-48 svg');
      const lines = chartSvg?.querySelectorAll('line');
      expect(lines?.length).toBe(5); // 5 horizontal grid lines
    });
  });

  it('should render area fill in chart', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    const { container } = render(<LettersGenerated />);

    await waitFor(() => {
      const path = container.querySelector('path');
      expect(path).toBeInTheDocument();
    });
  });

  it('should render polyline for data', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    const { container } = render(<LettersGenerated />);

    await waitFor(() => {
      const polyline = container.querySelector('polyline');
      expect(polyline).toBeInTheDocument();
    });
  });

  it('should render data points as circles', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    const { container } = render(<LettersGenerated />);

    await waitFor(() => {
      const circles = container.querySelectorAll('circle');
      expect(circles.length).toBe(7); // One for each data point
    });
  });

  it('should display x-axis date labels', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      // Should show first, middle, and last date
      const dateLabels = screen.getAllByText(/\d{2}\s\w+/);
      expect(dateLabels.length).toBeGreaterThanOrEqual(2);
    });
  });

  it('should format dates in French locale', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    // Wait for data to load and check for French formatted date (multiple dates shown)
    const dateTexts = await screen.findAllByText(/\d{2}\s\w+/);
    expect(dateTexts.length).toBeGreaterThanOrEqual(2);
    expect(dateTexts[0]).toBeInTheDocument();
  });

  it('should display period description', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      expect(screen.getByText('Évolution sur 7 jours')).toBeInTheDocument();
    });
  });

  it('should update period description when period changes', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      expect(screen.getByText('Évolution sur 7 jours')).toBeInTheDocument();
    });

    // Switch to month
    fireEvent.click(screen.getByText('Mois'));

    await waitFor(() => {
      expect(screen.getByText('Évolution sur 30 jours')).toBeInTheDocument();
    });

    // Switch to day
    fireEvent.click(screen.getByText('Jour'));

    await waitFor(() => {
      expect(screen.getByText('Évolution sur 24h')).toBeInTheDocument();
    });
  });

  it('should show message when no data available', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ total: 0, history: [] }),
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      expect(screen.getByText('Aucune donnée disponible')).toBeInTheDocument();
    });
  });

  it('should use mock data on API error', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    (global.fetch as jest.Mock).mockRejectedValueOnce(new Error('API Error'));

    render(<LettersGenerated />);

    await waitFor(() => {
      // Should display mock data total
      expect(screen.getByText('456')).toBeInTheDocument();
    });

    consoleErrorSpy.mockRestore();
  });

  it('should auto-refresh every 30 seconds', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    // Initial fetch
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledTimes(1);
    });

    // Advance 30 seconds
    jest.advanceTimersByTime(30000);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledTimes(2);
    });
  });

  it('should cleanup interval on unmount', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => mockLettersData,
    });

    const { unmount } = render(<LettersGenerated />);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledTimes(1);
    });

    unmount();

    // Advance time after unmount
    jest.advanceTimersByTime(60000);

    // Should not fetch again
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it('should handle empty history array', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ total: 100, history: [] }),
    });

    render(<LettersGenerated />);

    await waitFor(() => {
      expect(screen.getByText('100')).toBeInTheDocument();
      expect(screen.getByText('Aucune donnée disponible')).toBeInTheDocument();
    });
  });

  it('should render with rounded-lg border', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    const { container } = render(<LettersGenerated />);

    await waitFor(() => {
      const card = container.querySelector('.rounded-lg.border.bg-card');
      expect(card).toBeInTheDocument();
    });
  });

  it('should display total in large font', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    const { container } = render(<LettersGenerated />);

    await waitFor(() => {
      const totalElement = screen.getByText('456');
      expect(totalElement).toHaveClass('text-3xl', 'font-bold', 'text-primary');
    });
  });

  it('should refetch when period changes', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => mockLettersData,
    });

    render(<LettersGenerated />);

    // Wait for initial load to complete - buttons should be visible
    await waitFor(() => {
      expect(screen.getByText('Semaine')).toBeInTheDocument();
    }, { timeout: 5000 });

    // Now find and click the month button
    const monthButton = await screen.findByText('Mois');
    fireEvent.click(monthButton);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/analytics/letters?period=month',
        { credentials: 'include' }
      );
    });
  });

  it('should apply hover effect to circles', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockLettersData,
    });

    const { container } = render(<LettersGenerated />);

    await waitFor(() => {
      const circles = container.querySelectorAll('circle');
      circles.forEach((circle) => {
        expect(circle).toHaveClass('hover:r-5', 'transition-all');
      });
    });
  });
});
