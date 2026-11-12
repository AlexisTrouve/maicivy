import { render, screen, waitFor } from '@testing-library/react';
import { RepoList } from '../github/RepoList';
import { server } from '@/__mocks__/server';
import { rest } from 'msw';

// Setup MSW handlers
beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  jest.clearAllMocks();
});
afterAll(() => server.close());

const mockRepos = [
  {
    id: 1,
    repo_name: 'awesome-project',
    full_name: 'testuser/awesome-project',
    description: 'An awesome React project',
    url: 'https://github.com/testuser/awesome-project',
    stars: 42,
    language: 'TypeScript',
    topics: ['react', 'typescript', 'frontend'],
    is_private: false,
    pushed_at: '2024-01-15T10:00:00Z',
  },
  {
    id: 2,
    repo_name: 'backend-api',
    full_name: 'testuser/backend-api',
    description: 'RESTful API in Go',
    url: 'https://github.com/testuser/backend-api',
    stars: 15,
    language: 'Go',
    topics: ['go', 'api', 'backend'],
    is_private: true,
    pushed_at: '2024-01-10T08:00:00Z',
  },
  {
    id: 3,
    repo_name: 'ml-experiments',
    full_name: 'testuser/ml-experiments',
    description: 'Machine learning experiments',
    url: 'https://github.com/testuser/ml-experiments',
    stars: 8,
    language: 'Python',
    topics: ['python', 'machine-learning', 'ai'],
    is_private: false,
    pushed_at: '2024-01-05T12:00:00Z',
  },
];

describe('RepoList', () => {
  it('should render loading state initially', () => {
    render(<RepoList username="testuser" />);

    const loadingElements = document.querySelectorAll('.animate-pulse');
    expect(loadingElements.length).toBeGreaterThan(0);
  });

  it('should fetch and display repositories', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText('awesome-project')).toBeInTheDocument();
    });

    expect(screen.getByText('backend-api')).toBeInTheDocument();
    expect(screen.getByText('ml-experiments')).toBeInTheDocument();
  });

  it('should display repository descriptions', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText('An awesome React project')).toBeInTheDocument();
    });

    expect(screen.getByText('RESTful API in Go')).toBeInTheDocument();
    expect(screen.getByText('Machine learning experiments')).toBeInTheDocument();
  });

  it('should show star counts', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText('42')).toBeInTheDocument();
    });

    expect(screen.getByText('15')).toBeInTheDocument();
    expect(screen.getByText('8')).toBeInTheDocument();
  });

  it('should display language badges with correct colors', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      const typescriptBadge = screen.getByText('TypeScript');
      expect(typescriptBadge).toBeInTheDocument();
      expect(typescriptBadge).toHaveClass('bg-blue-100', 'text-blue-800');
    });

    const goBadge = screen.getByText('Go');
    expect(goBadge).toHaveClass('bg-cyan-100', 'text-cyan-800');

    const pythonBadge = screen.getByText('Python');
    expect(pythonBadge).toHaveClass('bg-green-100', 'text-green-800');
  });

  it('should show private badge for private repos', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText(/🔒 Privé/i)).toBeInTheDocument();
    });
  });

  it('should display topics as badges', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText('react')).toBeInTheDocument();
    });

    expect(screen.getByText('typescript')).toBeInTheDocument();
    expect(screen.getByText('frontend')).toBeInTheDocument();
    expect(screen.getByText('go')).toBeInTheDocument();
    expect(screen.getByText('api')).toBeInTheDocument();
  });

  it('should format dates correctly', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      // Check for French date format
      const dateElements = screen.getAllByText(/Mis à jour/i);
      expect(dateElements.length).toBeGreaterThan(0);
    });
  });

  it('should render repository links with correct hrefs', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      const link = screen.getByRole('link', { name: 'awesome-project' });
      expect(link).toHaveAttribute('href', 'https://github.com/testuser/awesome-project');
      expect(link).toHaveAttribute('target', '_blank');
      expect(link).toHaveAttribute('rel', 'noopener noreferrer');
    });
  });

  it('should handle error state', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json(
          { message: 'Failed to fetch repos' }
        ));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText(/erreur/i)).toBeInTheDocument();
    });

    expect(screen.getByText(/failed to fetch repos/i)).toBeInTheDocument();

    consoleErrorSpy.mockRestore();
  });

  it('should show empty state when no repos found', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: [],
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText(/aucun repo trouvé/i)).toBeInTheDocument();
    });
  });

  it('should filter private repos when showPrivate is false', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        const url = new URL(req.url);
        const includePrivate = url.searchParams.get('include_private');

        expect(includePrivate).toBe('false');

        return res(ctx.json({
          repositories: mockRepos.filter(r => !r.is_private),
        }));
      })
    );

    render(<RepoList username="testuser" showPrivate={false} />);

    await waitFor(() => {
      expect(screen.getByText('awesome-project')).toBeInTheDocument();
    });

    expect(screen.queryByText('backend-api')).not.toBeInTheDocument();
  });

  it('should include private repos when showPrivate is true', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        const url = new URL(req.url);
        const includePrivate = url.searchParams.get('include_private');

        expect(includePrivate).toBe('true');

        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" showPrivate={true} />);

    await waitFor(() => {
      expect(screen.getByText('awesome-project')).toBeInTheDocument();
    });

    expect(screen.getByText('backend-api')).toBeInTheDocument();
  });

  it('should display full_name for each repo', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText('testuser/awesome-project')).toBeInTheDocument();
    });

    expect(screen.getByText('testuser/backend-api')).toBeInTheDocument();
    expect(screen.getByText('testuser/ml-experiments')).toBeInTheDocument();
  });

  it('should show GitHub badge on each repo card', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      const githubBadges = screen.getAllByText('GitHub');
      expect(githubBadges.length).toBe(mockRepos.length);
    });
  });

  it('should apply hover shadow effect on cards', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: mockRepos,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      const cards = document.querySelectorAll('.hover\\:shadow-md');
      expect(cards.length).toBe(mockRepos.length);
    });
  });

  it('should handle repos without description', async () => {
    const reposWithoutDescription = [
      {
        ...mockRepos[0],
        description: '',
      },
    ];

    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({
          repositories: reposWithoutDescription,
        }));
      })
    );

    render(<RepoList username="testuser" />);

    await waitFor(() => {
      expect(screen.getByText('awesome-project')).toBeInTheDocument();
    });

    // Description should not be rendered if empty
    expect(screen.queryByText('An awesome React project')).not.toBeInTheDocument();
  });

  it('should refetch when username changes', async () => {
    let requestCount = 0;

    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        requestCount++;
        const url = new URL(req.url);
        const username = url.searchParams.get('username');

        return res(ctx.json({
          repositories: username === 'user1' ? [mockRepos[0]] : [mockRepos[1]],
        }));
      })
    );

    const { rerender } = render(<RepoList username="user1" />);

    await waitFor(() => {
      expect(requestCount).toBe(1);
    });

    rerender(<RepoList username="user2" />);

    await waitFor(() => {
      expect(requestCount).toBe(2);
    });
  });
});
