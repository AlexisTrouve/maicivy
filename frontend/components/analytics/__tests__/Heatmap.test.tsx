import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import Heatmap from '../Heatmap';

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Flame: () => <svg data-testid="flame-icon" />,
}));

// Mock environment variable
process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080';

// Mock global fetch
global.fetch = jest.fn();

describe('Heatmap', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  const mockHeatmapData = {
    points: [
      { x: 20, y: 15, intensity: 45, element: 'theme_selector' },
      { x: 50, y: 30, intensity: 78, element: 'experience_timeline' },
      { x: 75, y: 25, intensity: 92, element: 'export_pdf_button' },
      { x: 30, y: 60, intensity: 34, element: 'skills_cloud' },
      { x: 65, y: 70, intensity: 56, element: 'projects_grid' },
    ],
    maxIntensity: 100,
  };

  const renderAndWait = async () => {
    const result = render(<Heatmap />);
    await act(async () => {
      await jest.runOnlyPendingTimersAsync();
    });
    return result;
  };

  it('should render loading skeleton initially', () => {
    (global.fetch as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    const { container } = render(<Heatmap />);

    expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
  });

  it('should fetch heatmap data on mount', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    await act(async () => {
      render(<Heatmap />);
      await jest.runOnlyPendingTimersAsync();
    });

    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/analytics/heatmap',
      { credentials: 'include' }
    );
  });

  it('should render heatmap title', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    await renderAndWait();

    expect(screen.getByText('Heatmap des Interactions')).toBeInTheDocument();
  });

  it('should render Flame icon', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const flameIcon = container.querySelector('svg');
    expect(flameIcon).toBeInTheDocument();
  });

  it('should display color legend', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    await renderAndWait();

    expect(screen.getByText('Faible')).toBeInTheDocument();
    expect(screen.getByText('Fort')).toBeInTheDocument();
  });

  it('should render color gradient boxes in legend', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const colorBoxes = container.querySelectorAll('.w-6.h-4.rounded');
    expect(colorBoxes.length).toBeGreaterThanOrEqual(4);
  });

  it('should render all heatmap points', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const points = container.querySelectorAll('.rounded-full.blur-xl');
    expect(points.length).toBe(5);
  });

  it('should display tooltip on point hover', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const points = container.querySelectorAll('.rounded-full.blur-xl');
    expect(points.length).toBe(5);

    // Hover over first point
    const firstPoint = points[0];
    await act(async () => {
      fireEvent.mouseEnter(firstPoint);
    });

    expect(screen.getByText('Theme Selector')).toBeInTheDocument();
    expect(screen.getByText('45 interactions')).toBeInTheDocument();
  });

  it('should hide tooltip on mouse leave', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const points = container.querySelectorAll('.rounded-full.blur-xl');
    const firstPoint = points[0];

    await act(async () => {
      fireEvent.mouseEnter(firstPoint);
    });

    expect(screen.getByText('Theme Selector')).toBeInTheDocument();

    await act(async () => {
      fireEvent.mouseLeave(firstPoint);
    });

    expect(screen.queryByText('Theme Selector')).not.toBeInTheDocument();
  });

  it('should format element names correctly', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const points = container.querySelectorAll('.rounded-full.blur-xl');
    expect(points.length).toBe(5);

    // Hover to see formatted name
    const firstPoint = points[0];
    await act(async () => {
      fireEvent.mouseEnter(firstPoint);
    });

    // 'theme_selector' should become 'Theme Selector'
    expect(screen.getByText('Theme Selector')).toBeInTheDocument();
  });

  it('should display point count', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    await renderAndWait();

    expect(screen.getByText('Basé sur 5 points d\'interaction')).toBeInTheDocument();
  });

  it('should use mock data on API error', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    (global.fetch as jest.Mock).mockRejectedValueOnce(new Error('API Error'));

    const { container } = await renderAndWait();

    // Should render mock data points
    const points = container.querySelectorAll('.rounded-full.blur-xl');
    expect(points.length).toBe(7); // Mock data has 7 points

    consoleErrorSpy.mockRestore();
  });

  it('should show message when no data available', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ points: [], maxIntensity: 0 }),
    });

    await renderAndWait();

    expect(
      screen.getByText('Aucune donnée d\'interaction disponible')
    ).toBeInTheDocument();
  });

  it('should render grid overlay', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const gridOverlay = container.querySelector('.grid.grid-cols-10.grid-rows-10');
    expect(gridOverlay).toBeInTheDocument();
  });

  it('should render 100 grid cells', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const gridCells = container.querySelectorAll('.border.border-foreground');
    expect(gridCells.length).toBe(100);
  });

  it('should auto-refresh every 60 seconds', async () => {
    let fetchCount = 0;
    (global.fetch as jest.Mock).mockImplementation(async () => {
      fetchCount++;
      return {
        ok: true,
        json: async () => mockHeatmapData,
      };
    });

    render(<Heatmap />);

    // Wait for initial render and fetch
    await act(async () => {
      await jest.runOnlyPendingTimersAsync();
    });

    const initialFetchCount = fetchCount;

    // Advance 60 seconds - this should trigger the interval
    await act(async () => {
      jest.advanceTimersByTime(60000);
      await jest.runOnlyPendingTimersAsync();
    });

    // Should have fetched at least once more (interval triggered)
    expect(fetchCount).toBeGreaterThan(initialFetchCount);
  });

  it('should cleanup interval on unmount', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { unmount } = render(<Heatmap />);

    // Wait for initial render and fetch
    await act(async () => {
      await jest.runOnlyPendingTimersAsync();
    });

    const initialCallCount = (global.fetch as jest.Mock).mock.calls.length;

    await act(async () => {
      unmount();
    });

    // Advance time after unmount
    jest.advanceTimersByTime(120000);

    // Should not fetch again after unmount
    expect((global.fetch as jest.Mock).mock.calls.length).toBe(initialCallCount);
  });

  it('should position points correctly', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const points = container.querySelectorAll('.rounded-full.blur-xl');
    const firstPoint = points[0] as HTMLElement;

    // First point should be at x: 20%, y: 15%
    expect(firstPoint.style.left).toBe('20%');
    expect(firstPoint.style.top).toBe('15%');
  });

  it('should apply appropriate colors based on intensity', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const points = container.querySelectorAll('.rounded-full.blur-xl');

    points.forEach((point) => {
      const bgColor = (point as HTMLElement).style.backgroundColor;
      expect(bgColor).toMatch(/rgba?\(/);
    });
  });

  it('should handle API response with 404', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      status: 404,
    });

    const { container } = await renderAndWait();

    // Should fall back to mock data
    const points = container.querySelectorAll('.rounded-full.blur-xl');
    expect(points.length).toBeGreaterThan(0);

    consoleErrorSpy.mockRestore();
  });

  it('should render with rounded-lg border', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const card = container.querySelector('.rounded-lg.border.bg-card');
    expect(card).toBeInTheDocument();
  });

  it('should have pointer-events-auto on heatmap points', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const points = container.querySelectorAll('.rounded-full.blur-xl');
    points.forEach((point) => {
      const style = (point as HTMLElement).style.pointerEvents;
      expect(style).toBe('auto');
    });
  });

  it('should position tooltip above hovered point', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockHeatmapData,
    });

    const { container } = await renderAndWait();

    const points = container.querySelectorAll('.rounded-full.blur-xl');

    await act(async () => {
      fireEvent.mouseEnter(points[0]);
    });

    const tooltip = container.querySelector('.bg-popover.shadow-lg');
    expect(tooltip).toBeInTheDocument();
  });
});
