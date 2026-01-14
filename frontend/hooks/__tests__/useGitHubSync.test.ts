import { renderHook, waitFor, act } from '@testing-library/react'
import { useGitHubSync } from '../useGitHubSync'
import { server } from '@/__mocks__/server'
import { rest } from 'msw'

describe('useGitHubSync', () => {
  beforeAll(() => server.listen())
  afterEach(() => {
    server.resetHandlers()
    localStorage.clear()
  })
  afterAll(() => server.close())

  it('should fetch status on mount when username is provided', async () => {
    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        const url = new URL(req.url)
        const username = url.searchParams.get('username')

        if (username === 'testuser') {
          return res(ctx.json({
            connected: true,
            username: 'testuser',
            last_sync: Date.now(),
            repo_count: 15,
          }))
        }
        return res(ctx.json({ connected: false, repo_count: 0 }))
      })
    )

    const { result } = renderHook(() => useGitHubSync('testuser'))

    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.status?.connected).toBe(true)
    expect(result.current.status?.username).toBe('testuser')
    expect(result.current.state).toBe('connected')
    expect(result.current.error).toBeNull()
  })

  it('should not fetch status when username is not provided', () => {
    const fetchSpy = jest.spyOn(global, 'fetch')

    renderHook(() => useGitHubSync())

    expect(fetchSpy).not.toHaveBeenCalled()

    fetchSpy.mockRestore()
  })

  it('should handle status fetch errors', async () => {
    server.use(
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json({ message: 'Server error' }))
      })
    )

    const { result } = renderHook(() => useGitHubSync('testuser'))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBeTruthy()
    expect(result.current.state).toBe('error')
  })

  it('should fetch repos when fetchRepos is called', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        const url = new URL(req.url)
        const username = url.searchParams.get('username')

        if (username === 'testuser') {
          return res(ctx.json({
            repositories: [
              {
                id: 1,
                repo_name: 'test-repo',
                full_name: 'testuser/test-repo',
                description: 'Test repository',
                url: 'https://github.com/testuser/test-repo',
                stars: 42,
                language: 'TypeScript',
                topics: ['react', 'nextjs'],
                is_private: false,
                pushed_at: '2024-01-01T00:00:00Z',
              },
            ],
          }))
        }
        return res(ctx.json({ repositories: [] }))
      })
    )

    const { result } = renderHook(() => useGitHubSync('testuser'))

    await act(async () => {
      await result.current.fetchRepos()
    })

    expect(result.current.repos).toHaveLength(1)
    expect(result.current.repos[0].repo_name).toBe('test-repo')
    expect(result.current.error).toBeNull()
  })

  it('should handle repos fetch errors', async () => {
    server.use(
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json(
          { message: 'Failed to fetch repos' }
        ))
      })
    )

    const { result } = renderHook(() => useGitHubSync('testuser'))

    await act(async () => {
      await result.current.fetchRepos()
    })

    expect(result.current.error).toBeTruthy()
    expect(result.current.repos).toHaveLength(0)
  })

  it('should initiate GitHub OAuth connection', async () => {
    const mockPopup = {
      closed: false,
      close: jest.fn(),
    }

    window.open = jest.fn(() => mockPopup as any)

    server.use(
      rest.get('*/api/v1/github/auth-url', (req, res, ctx) => {
        return res(ctx.json({
          auth_url: 'https://github.com/login/oauth/authorize?client_id=test',
        }))
      })
    )

    const { result } = renderHook(() => useGitHubSync())

    await act(async () => {
      await result.current.connect()
    })

    expect(window.open).toHaveBeenCalledWith(
      'https://github.com/login/oauth/authorize?client_id=test',
      'GitHub OAuth',
      expect.any(String)
    )
    expect(result.current.state).toBe('connecting')
  })

  it('should sync GitHub repositories', async () => {
    server.use(
      rest.post('*/api/v1/github/sync', async (req, res, ctx) => {
        const body = await req.json()
        return res(ctx.json({
          status: 'success',
          username: body.username,
        }))
      }),
      rest.get('*/api/v1/github/status', (req, res, ctx) => {
        return res(ctx.json({
          connected: true,
          username: 'testuser',
          last_sync: Date.now(),
          repo_count: 20,
        }))
      }),
      rest.get('*/api/v1/github/repos', (req, res, ctx) => {
        return res(ctx.json({ repositories: [] }))
      })
    )

    const { result } = renderHook(() => useGitHubSync('testuser'))

    await act(async () => {
      await result.current.sync()
    })

    await waitFor(() => {
      expect(result.current.state).toBe('connected')
    }, { timeout: 5000 })

    expect(result.current.error).toBeNull()
  })

  it('should handle sync errors', async () => {
    server.use(
      rest.post('*/api/v1/github/sync', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json({ message: 'Sync failed' }))
      })
    )

    const { result } = renderHook(() => useGitHubSync('testuser'))

    await act(async () => {
      await result.current.sync()
    })

    expect(result.current.error).toBeTruthy()
    expect(result.current.state).toBe('error')
  })

  it('should disconnect GitHub account', async () => {
    server.use(
      rest.delete('*/api/v1/github/disconnect', (req, res, ctx) => {
        return res(ctx.json({
          success: true,
          message: 'Disconnected successfully',
        }))
      })
    )

    const { result } = renderHook(() => useGitHubSync('testuser'))

    // Wait for initial status fetch
    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    await act(async () => {
      await result.current.disconnect()
    })

    await waitFor(() => {
      expect(result.current.status).toBeNull()
      expect(result.current.state).toBe('idle')
    })

    expect(result.current.repos).toHaveLength(0)
    expect(result.current.error).toBeNull()
  })

  it('should require username for sync', async () => {
    const { result } = renderHook(() => useGitHubSync())

    await act(async () => {
      await result.current.sync()
    })

    expect(result.current.error).toBe('Username required')
  })
})
