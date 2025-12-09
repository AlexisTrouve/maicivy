import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { GitHubConnect } from '../github/GitHubConnect';
import { server } from '@/__mocks__/server';
import { http, HttpResponse } from 'msw';

// Setup MSW handlers
beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  jest.clearAllMocks();
  localStorage.clear();
});
afterAll(() => server.close());

// Mock window.open
const mockPopup = {
  closed: false,
  close: jest.fn(),
};

describe('GitHubConnect', () => {
  beforeEach(() => {
    global.window.open = jest.fn().mockReturnValue(mockPopup);
  });

  it('should render connect button', () => {
    render(<GitHubConnect />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    expect(button).toBeInTheDocument();
    expect(button).not.toBeDisabled();
  });

  it('should display GitHub icon', () => {
    render(<GitHubConnect />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    const svg = button.querySelector('svg');
    expect(svg).toBeInTheDocument();
  });

  it('should fetch auth URL when button is clicked', async () => {
    const mockAuthUrl = 'https://github.com/login/oauth/authorize?client_id=test';

    server.use(
      http.get('*/api/v1/github/auth-url', () => {
        return HttpResponse.json({
          auth_url: mockAuthUrl,
        });
      })
    );

    render(<GitHubConnect />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(window.open).toHaveBeenCalledWith(
        mockAuthUrl,
        'GitHub OAuth',
        expect.stringContaining('width=600')
      );
    });
  });

  it('should show loading state during connection', async () => {
    server.use(
      http.get('*/api/v1/github/auth-url', async () => {
        await new Promise(resolve => setTimeout(resolve, 100));
        return HttpResponse.json({
          auth_url: 'https://github.com/login/oauth/authorize?client_id=test',
        });
      })
    );

    render(<GitHubConnect />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText(/connexion/i)).toBeInTheDocument();
    });

    const loadingButton = screen.getByRole('button');
    expect(loadingButton).toBeDisabled();

    // Check for loading spinner
    const spinner = document.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
  });

  it('should handle API error when fetching auth URL', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    server.use(
      http.get('*/api/v1/github/auth-url', () => {
        return HttpResponse.json(
          { error: 'Failed to get auth URL' },
          { status: 500 }
        );
      })
    );

    render(<GitHubConnect />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText(/erreur/i)).toBeInTheDocument();
    });

    expect(screen.getByText(/failed to get github auth url/i)).toBeInTheDocument();

    consoleErrorSpy.mockRestore();
  });

  it('should call onConnectSuccess when connection succeeds', async () => {
    const onConnectSuccess = jest.fn();
    const mockUsername = 'testuser';

    server.use(
      http.get('*/api/v1/github/auth-url', () => {
        return HttpResponse.json({
          auth_url: 'https://github.com/login/oauth/authorize?client_id=test',
        });
      })
    );

    render(<GitHubConnect onConnectSuccess={onConnectSuccess} />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(window.open).toHaveBeenCalled();
    });

    // Simulate popup closing after successful auth
    localStorage.setItem('github_connected', 'true');
    localStorage.setItem('github_username', mockUsername);
    mockPopup.closed = true;

    // Trigger the interval check by advancing timers
    jest.useFakeTimers();
    jest.advanceTimersByTime(500);

    await waitFor(() => {
      expect(onConnectSuccess).toHaveBeenCalledWith(mockUsername);
    });

    jest.useRealTimers();
  });

  it('should call onConnectError when connection fails', async () => {
    const onConnectError = jest.fn();

    server.use(
      http.get('*/api/v1/github/auth-url', () => {
        return HttpResponse.json(
          { error: 'Auth failed' },
          { status: 500 }
        );
      })
    );

    render(<GitHubConnect onConnectError={onConnectError} />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(onConnectError).toHaveBeenCalledWith(expect.stringContaining('Failed to get GitHub auth URL'));
    });
  });

  it('should open popup with correct dimensions and position', async () => {
    server.use(
      http.get('*/api/v1/github/auth-url', () => {
        return HttpResponse.json({
          auth_url: 'https://github.com/login/oauth/authorize?client_id=test',
        });
      })
    );

    // Mock screen dimensions
    Object.defineProperty(window, 'screen', {
      writable: true,
      value: {
        width: 1920,
        height: 1080,
      },
    });

    render(<GitHubConnect />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(window.open).toHaveBeenCalledWith(
        expect.any(String),
        'GitHub OAuth',
        expect.stringMatching(/width=600.*height=700/)
      );
    });
  });

  it('should handle popup being blocked', async () => {
    global.window.open = jest.fn().mockReturnValue(null);

    server.use(
      http.get('*/api/v1/github/auth-url', () => {
        return HttpResponse.json({
          auth_url: 'https://github.com/login/oauth/authorize?client_id=test',
        });
      })
    );

    render(<GitHubConnect />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(window.open).toHaveBeenCalled();
    });

    // Should not crash when popup is null
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should clean up interval when popup closes without auth', async () => {
    jest.useFakeTimers();

    server.use(
      http.get('*/api/v1/github/auth-url', () => {
        return HttpResponse.json({
          auth_url: 'https://github.com/login/oauth/authorize?client_id=test',
        });
      })
    );

    render(<GitHubConnect />);

    const button = screen.getByRole('button', { name: /connecter github/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(window.open).toHaveBeenCalled();
    });

    // Simulate popup closing without success
    mockPopup.closed = true;
    jest.advanceTimersByTime(500);

    // Button should be enabled again
    await waitFor(() => {
      const btn = screen.getByRole('button');
      expect(btn).not.toBeDisabled();
    });

    jest.useRealTimers();
  });
});
