import { renderHook, waitFor, act } from '@testing-library/react'
import { useAnalyticsData } from '../useAnalyticsData'
import { server } from '@/__mocks__/server'
import { rest } from 'msw'

describe('useAnalyticsData', () => {
  beforeAll(() => server.listen())

  afterEach(() => {
    server.resetHandlers()
    jest.clearAllTimers()
    jest.useRealTimers()
  })

  afterAll(() => server.close())

  it('should fetch data from endpoint on mount', async () => {
    server.use(
      rest.get('http://localhost:8080/api/analytics/stats', (req, res, ctx) => {
        return res(ctx.json({
          totalVisits: 100,
          uniqueVisitors: 50,
          avgSessionDuration: 180,
        }))
      })
    )

    const { result } = renderHook(() =>
      useAnalyticsData({
        endpoint: '/api/analytics/stats',
        refreshInterval: 0,
      })
    )

    expect(result.current.isLoading).toBe(true)

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.data).toEqual({
      totalVisits: 100,
      uniqueVisitors: 50,
      avgSessionDuration: 180,
    })
    expect(result.current.error).toBeNull()
  })

  it('should not fetch when enabled is false', async () => {
    const fetchSpy = jest.spyOn(global, 'fetch')

    const { result } = renderHook(() =>
      useAnalyticsData({
        endpoint: '/api/analytics/stats',
        enabled: false,
      })
    )

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(fetchSpy).not.toHaveBeenCalled()
    expect(result.current.data).toBeNull()

    fetchSpy.mockRestore()
  })

  it('should handle API errors gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    server.use(
      rest.get('http://localhost:8080/api/analytics/error', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json(
          { message: 'Internal server error' }
        ))
      })
    )

    const { result } = renderHook(() =>
      useAnalyticsData({
        endpoint: '/api/analytics/error',
        refreshInterval: 0,
      })
    )

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.error).toBeTruthy()
    expect(result.current.error?.message).toContain('Failed to fetch data')
    expect(result.current.data).toBeNull()

    consoleErrorSpy.mockRestore()
  })

  it('should refetch data when refetch is called', async () => {
    let callCount = 0
    server.use(
      rest.get('http://localhost:8080/api/analytics/refetch', (req, res, ctx) => {
        callCount++
        return res(ctx.json({
          count: callCount,
        }))
      })
    )

    const { result } = renderHook(() =>
      useAnalyticsData({
        endpoint: '/api/analytics/refetch',
        refreshInterval: 0,
      })
    )

    await waitFor(() => {
      expect(result.current.data).toEqual({ count: 1 })
    })

    // Call refetch wrapped in act()
    await act(async () => {
      await result.current.refetch()
    })

    await waitFor(() => {
      expect(result.current.data).toEqual({ count: 2 })
    })
  })

  it('should clear error on successful retry', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()
    let shouldFail = true

    server.use(
      rest.get('http://localhost:8080/api/analytics/retry', (req, res, ctx) => {
        if (shouldFail) {
          return res(ctx.status(500), ctx.json(
            { message: 'Temporary error' }
          ))
        }
        return res(ctx.json({ success: true }))
      })
    )

    const { result } = renderHook(() =>
      useAnalyticsData({
        endpoint: '/api/analytics/retry',
        refreshInterval: 0,
      })
    )

    await waitFor(() => {
      expect(result.current.error).toBeTruthy()
    })

    // Fix the error and refetch, wrapped in act()
    await act(async () => {
      shouldFail = false
      await result.current.refetch()
    })

    await waitFor(() => {
      expect(result.current.error).toBeNull()
    })

    expect(result.current.data).toEqual({ success: true })

    consoleErrorSpy.mockRestore()
  })

  it('should include credentials in fetch request', async () => {
    const fetchSpy = jest.spyOn(global, 'fetch')

    server.use(
      rest.get('http://localhost:8080/api/analytics/credentials', (req, res, ctx) => {
        return res(ctx.json({ authenticated: true }))
      })
    )

    renderHook(() =>
      useAnalyticsData({
        endpoint: '/api/analytics/credentials',
        refreshInterval: 0,
      })
    )

    await waitFor(() => {
      expect(fetchSpy).toHaveBeenCalled()
    })

    const fetchCall = fetchSpy.mock.calls[0]
    const options = fetchCall[1] as RequestInit
    expect(options.credentials).toBe('include')

    fetchSpy.mockRestore()
  })
})
