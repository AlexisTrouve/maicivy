import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { GitHubStatus } from '../github/GitHubStatus';
import { server } from '@/__mocks__/server';
import { rest } from 'msw';

// Setup MSW handlers
beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  jest.clearAllMocks();
});
afterAll(() => server.close());

const mockStatus = {
  connected: true,
  username: 'testuser',
  last_sync: Math.floor(Date.now() / 1000) - 3600, // 1 hour ago
  repo_count: 15,
};

describe('GitHubStatus', () => {
  it('should render loading state initially', () => {
    render(<GitHubStatus username="testuser" />);

    const loadingElement = document.querySelector('.animate-pulse');
    expect(loadingElement).toBeInTheDocument();
  });

  it('should fetch and display status', async () => {
    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json(mockStatus));
      })
    );

    render(<GitHubStatus username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText(/connecté à github/i)).toBeInTheDocument();
    });

    expect(screen.getByText('@testuser')).toBeInTheDocument();
    expect(screen.getByText('15')).toBeInTheDocument();
  });

  it('should display "not connected" when status is disconnected', async () => {
    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json({
          connected: false,
        }));
      })
    );

    render(<GitHubStatus username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText(/github non connecté/i)).toBeInTheDocument();
    });
  });

  it('should format last sync time correctly', async () => {
    const now = Math.floor(Date.now() / 1000);

    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json({
          ...mockStatus,
          last_sync: now - 120, // 2 minutes ago
        }));
      })
    );

    render(<GitHubStatus username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText(/Il y a 2 min/i)).toBeInTheDocument();
    });
  });

  it('should show "À l\'instant" for very recent syncs', async () => {
    const now = Math.floor(Date.now() / 1000);

    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json({
          ...mockStatus,
          last_sync: now - 30, // 30 seconds ago
        }));
      })
    );

    render(<GitHubStatus username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText(/À l'instant/i)).toBeInTheDocument();
    });
  });

  it('should handle sync button click', async () => {
    const onSync = jest.fn();

    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json(mockStatus));
      }),
      rest.post('*/api/v1/github/sync', (req, res, ctx) => {
        return res(ctx.json({ status: 'success' }));
      })
    );

    render(<GitHubStatus username="testuser" onSync={onSync} />);

    await waitFor(() => {
      expect(screen.getByText(/synchroniser/i)).toBeInTheDocument();
    });

    const syncButton = screen.getByRole('button', { name: /synchroniser/i });
    fireEvent.click(syncButton);

    // Should show syncing state
    await waitFor(() => {
      expect(screen.getByText(/synchro\.\.\./i)).toBeInTheDocument();
    });

    // Button should be disabled during sync
    expect(syncButton).toBeDisabled();

    // After 2 seconds, should call onSync callback
    await waitFor(() => {
      expect(onSync).toHaveBeenCalled();
    }, { timeout: 3000 });
  });

  it('should handle sync error gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json(mockStatus));
      }),
      rest.post('*/api/v1/github/sync', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json(
          { error: 'Sync failed' }
        ));
      })
    );

    render(<GitHubStatus username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText(/synchroniser/i)).toBeInTheDocument();
    });

    const syncButton = screen.getByRole('button', { name: /synchroniser/i });
    fireEvent.click(syncButton);

    // Should stop loading even on error
    await waitFor(() => {
      expect(screen.getByText(/synchroniser/i)).toBeInTheDocument();
    });

    consoleErrorSpy.mockRestore();
  });

  it('should handle disconnect button click with confirmation', async () => {
    const onDisconnect = jest.fn();
    global.confirm = jest.fn().mockReturnValue(true);

    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json(mockStatus));
      }),
      rest.delete('*/api/v1/github/disconnect', (req, res, ctx) => {
        return res(ctx.json({ success: true }));
      })
    );

    render(<GitHubStatus username="testuser" onDisconnect={onDisconnect} />);

    await waitFor(() => {
      expect(screen.getByText(/déconnecter/i)).toBeInTheDocument();
    });

    const disconnectButton = screen.getByRole('button', { name: /déconnecter/i });
    fireEvent.click(disconnectButton);

    expect(global.confirm).toHaveBeenCalledWith(
      expect.stringContaining('déconnecter GitHub')
    );

    await waitFor(() => {
      expect(onDisconnect).toHaveBeenCalled();
    });
  });

  it('should cancel disconnect when user declines confirmation', async () => {
    const onDisconnect = jest.fn();
    global.confirm = jest.fn().mockReturnValue(false);

    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json(mockStatus));
      })
    );

    render(<GitHubStatus username="testuser" onDisconnect={onDisconnect} />);

    await waitFor(() => {
      expect(screen.getByText(/déconnecter/i)).toBeInTheDocument();
    });

    const disconnectButton = screen.getByRole('button', { name: /déconnecter/i });
    fireEvent.click(disconnectButton);

    expect(global.confirm).toHaveBeenCalled();
    expect(onDisconnect).not.toHaveBeenCalled();
  });

  it('should show loading spinner during sync', async () => {
    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json(mockStatus));
      }),
      rest.post('*/api/v1/github/sync', async (req, res, ctx) => {
        await new Promise(resolve => setTimeout(resolve, 100));
        return res(ctx.json({ status: 'success' }));
      })
    );

    render(<GitHubStatus username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText(/synchroniser/i)).toBeInTheDocument();
    });

    const syncButton = screen.getByRole('button', { name: /synchroniser/i });
    fireEvent.click(syncButton);

    await waitFor(() => {
      const spinner = document.querySelector('.animate-spin');
      expect(spinner).toBeInTheDocument();
    });
  });

  it('should display repo count correctly', async () => {
    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json({
          ...mockStatus,
          repo_count: 42,
        }));
      })
    );

    render(<GitHubStatus username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText('42')).toBeInTheDocument();
      expect(screen.getByText(/repos importés/i)).toBeInTheDocument();
    });
  });

  it('should show green pulse indicator when connected', async () => {
    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json(mockStatus));
      })
    );

    render(<GitHubStatus username="testuser" />);

    await waitFor(() => {
      const pulseIndicator = document.querySelector('.bg-green-500.animate-pulse');
      expect(pulseIndicator).toBeInTheDocument();
    });
  });

  it('should refetch status after successful sync', async () => {
    let fetchCount = 0;

    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        fetchCount++;
        return res(ctx.json(mockStatus));
      }),
      rest.post('*/api/v1/github/sync', (req, res, ctx) => {
        return res(ctx.json({ status: 'success' }));
      })
    );

    render(<GitHubStatus username="testuser" />);

    await waitFor(() => {
      expect(fetchCount).toBe(1);
    });

    const syncButton = screen.getByRole('button', { name: /synchroniser/i });
    fireEvent.click(syncButton);

    await waitFor(() => {
      expect(fetchCount).toBeGreaterThan(1);
    }, { timeout: 3000 });
  });
});
