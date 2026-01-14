import { render, screen, waitFor, act } from '@testing-library/react';
import RealtimeVisitors from '../RealtimeVisitors';

// Mock lucide-react
jest.mock('lucide-react', () => ({
  Activity: () => <div data-testid="activity-icon">Activity Icon</div>,
}));

// Mock WebSocket
class MockWebSocket {
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((error: Event) => void) | null = null;
  onclose: (() => void) | null = null;
  readyState: number = WebSocket.CONNECTING;

  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  constructor(url: string) {
    this.url = url;
    // Simulate connection opening
    setTimeout(() => {
      this.readyState = WebSocket.OPEN;
      if (this.onopen) this.onopen();
    }, 50);
  }

  send(data: string) {
    // Mock send
  }

  close() {
    this.readyState = WebSocket.CLOSED;
    if (this.onclose) this.onclose();
  }

  // Helper to simulate receiving message
  simulateMessage(data: any) {
    if (this.onmessage) {
      const event = new MessageEvent('message', {
        data: JSON.stringify(data),
      });
      this.onmessage(event);
    }
  }
}

let mockWebSocket: MockWebSocket | null = null;

describe('RealtimeVisitors', () => {
  beforeEach(() => {
    // Mock WebSocket
    global.WebSocket = jest.fn((url: string) => {
      mockWebSocket = new MockWebSocket(url);
      return mockWebSocket as any;
    }) as any;

    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.clearAllTimers();
    jest.useRealTimers();
    mockWebSocket = null;
  });

  it('should render component with initial state', () => {
    render(<RealtimeVisitors />);

    expect(screen.getByText(/Visiteurs Actuels/i)).toBeInTheDocument();
    expect(screen.getByText('0')).toBeInTheDocument();
  });

  it('should show disconnected status initially', () => {
    render(<RealtimeVisitors />);

    expect(screen.getByText('Déconnecté')).toBeInTheDocument();
  });

  it('should connect to WebSocket on mount', async () => {
    render(<RealtimeVisitors />);

    act(() => {
      jest.advanceTimersByTime(100);
    });

    await waitFor(() => {
      expect(screen.getByText('En ligne')).toBeInTheDocument();
    });
  });

  it('should update visitor count when receiving WebSocket message', async () => {
    render(<RealtimeVisitors />);

    act(() => {
      jest.advanceTimersByTime(100);
    });

    await waitFor(() => {
      expect(screen.getByText('En ligne')).toBeInTheDocument();
    });

    // Simulate WebSocket message
    act(() => {
      if (mockWebSocket) {
        mockWebSocket.simulateMessage({
          currentVisitors: 5,
          timestamp: Date.now(),
        });
      }
    });

    // Allow time for count animation
    act(() => {
      jest.advanceTimersByTime(1100);
    });

    await waitFor(() => {
      expect(screen.getByText('5')).toBeInTheDocument();
    });
  });

  it('should handle multiple WebSocket messages', async () => {
    render(<RealtimeVisitors />);

    act(() => {
      jest.advanceTimersByTime(100);
    });

    await waitFor(() => {
      expect(screen.getByText('En ligne')).toBeInTheDocument();
    });

    // First message
    act(() => {
      if (mockWebSocket) {
        mockWebSocket.simulateMessage({
          currentVisitors: 3,
          timestamp: Date.now(),
        });
      }
      jest.advanceTimersByTime(1100);
    });

    await waitFor(() => {
      expect(screen.getByText('3')).toBeInTheDocument();
    });

    // Second message
    act(() => {
      if (mockWebSocket) {
        mockWebSocket.simulateMessage({
          currentVisitors: 7,
          timestamp: Date.now(),
        });
      }
      jest.advanceTimersByTime(1100);
    });

    await waitFor(() => {
      const sevenElements = screen.getAllByText('7');
      expect(sevenElements.length).toBeGreaterThanOrEqual(1);
    }, { timeout: 2000 });
  });

  it('should handle WebSocket disconnection', async () => {
    render(<RealtimeVisitors />);

    act(() => {
      jest.advanceTimersByTime(100);
    });

    await waitFor(() => {
      expect(screen.getByText('En ligne')).toBeInTheDocument();
    });

    // Simulate disconnection
    act(() => {
      if (mockWebSocket && mockWebSocket.onclose) {
        mockWebSocket.onclose();
      }
    });

    await waitFor(() => {
      expect(screen.getByText('Déconnecté')).toBeInTheDocument();
    });
  });

  it('should show correct singular/plural text for visitor count', async () => {
    render(<RealtimeVisitors />);

    act(() => {
      jest.advanceTimersByTime(100);
    });

    // 0 visitors
    expect(screen.getByText(/personne en ce moment/i)).toBeInTheDocument();

    // 1 visitor
    act(() => {
      if (mockWebSocket) {
        mockWebSocket.simulateMessage({
          currentVisitors: 1,
          timestamp: Date.now(),
        });
      }
      jest.advanceTimersByTime(1100);
    });

    await waitFor(() => {
      expect(screen.getByText(/personne en ce moment/i)).toBeInTheDocument();
    });

    // Multiple visitors
    act(() => {
      if (mockWebSocket) {
        mockWebSocket.simulateMessage({
          currentVisitors: 5,
          timestamp: Date.now(),
        });
      }
      jest.advanceTimersByTime(1100);
    });

    await waitFor(() => {
      expect(screen.getByText(/personnes en ce moment/i)).toBeInTheDocument();
    });
  });

  it('should render connection status indicator', async () => {
    const { container } = render(<RealtimeVisitors />);

    // Initially disconnected (red)
    let statusDot = container.querySelector('.bg-red-500');
    expect(statusDot).toBeInTheDocument();

    // After connection (green)
    act(() => {
      jest.advanceTimersByTime(100);
    });

    await waitFor(() => {
      statusDot = container.querySelector('.bg-green-500');
      expect(statusDot).toBeInTheDocument();
    });
  });

  it('should render Activity icon', () => {
    render(<RealtimeVisitors />);

    // Since we mocked Activity component, check for the mocked element
    expect(screen.getByTestId('activity-icon')).toBeInTheDocument();
  });

  it('should display realtime update message', () => {
    render(<RealtimeVisitors />);

    expect(
      screen.getByText(/Mise à jour en temps réel via WebSocket/i)
    ).toBeInTheDocument();
  });

  it('should handle invalid JSON message gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    render(<RealtimeVisitors />);

    act(() => {
      jest.advanceTimersByTime(100);
    });

    await waitFor(() => {
      expect(screen.getByText('En ligne')).toBeInTheDocument();
    });

    // Send invalid JSON
    act(() => {
      if (mockWebSocket && mockWebSocket.onmessage) {
        const event = new MessageEvent('message', {
          data: 'invalid json',
        });
        mockWebSocket.onmessage(event);
      }
    });

    // Should still show previous count (0)
    expect(screen.getByText('0')).toBeInTheDocument();

    consoleErrorSpy.mockRestore();
  });

  it('should cleanup WebSocket on unmount', async () => {
    const { unmount } = render(<RealtimeVisitors />);

    act(() => {
      jest.advanceTimersByTime(100);
    });

    await waitFor(() => {
      expect(mockWebSocket?.readyState).toBe(WebSocket.OPEN);
    });

    const closeSpy = jest.spyOn(mockWebSocket!, 'close');

    unmount();

    expect(closeSpy).toHaveBeenCalled();
  });
});
