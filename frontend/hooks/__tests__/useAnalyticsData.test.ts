import { renderHook, waitFor } from '@testing-library/react'
import { useAnalyticsData } from '../useAnalyticsData'
import { server } from '@/__mocks__/server'
import { http, HttpResponse } from 'msw'

describe('useAnalyticsData', () => {
  beforeAll(() => server.listen())
  afterEach(() => {
    server.resetHandlers()
    jest.clearAllTimers()
  })
  afterAll(() => server.close())

  beforeEach(() => {
    jest.useFakeTimers()
  })

  afterEach(() => {
    jest.runOnlyPendingTimers()
    jest.useRealTimers()
  })

  it('should fetch data from endpoint on mount', async () => {
    server.use(
      http.get('http://localhost:8080/api/analytics/stats', () => {
        return HttpResponse.json({
          totalVisits: 100,
          uniqueVisitors: 50,
          avgSessionDuration: 180,
        })
      })
    )

    const { result } = renderHook(() =>
      useAnalyticsData({
        endpoint: '/api/analytics/stats',
        refreshInterval: 0, // Disable polling for this test
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
      http.get('http://localhost:8080/api/analytics/error', () => {
        return HttpResponse.json(
          { message: 'Internal server error' },
          { status: 500 }
        )
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

  it('should poll data at specified interval', async () => {
    let callCount = 0
    server.use(
      http.get('http://localhost:8080/api/analytics/polling', () => {
        callCount++
        return HttpResponse.json({
          count: callCount,
          timestamp: Date.now(),
        })
      })
    )

    const { result } = renderHook(() =>
      useAnalyticsData({
        endpoint: '/api/analytics/polling',
        refreshInterval: 1000, // Poll every 1 second
      })
    )

    // Initial fetch
    await waitFor(() => {
      expect(result.current.data).toEqual({ count: 1, timestamp: expect.any(Number) })
    })

    // Fast-forward 1 second
    jest.advanceTimersByTime(1000)

    await waitFor(() => {
      expect(result.current.data).toEqual({ count: 2, timestamp: expect.any(Number) })
    })

    // Fast-forward another second
    jest.advanceTimersByTime(1000)

    await waitFor(() => {
      expect(result.current.data).toEqual({ count: 3, timestamp: expect.any(Number) })
    })
  })

  it('should refetch data when refetch is called', async () => {
    let callCount = 0
    server.use(
      http.get('http://localhost:8080/api/analytics/refetch', () => {
        callCount++
        return HttpResponse.json({
          count: callCount,
        })
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

    // Call refetch
    await result.current.refetch()

    await waitFor(() => {
      expect(result.current.data).toEqual({ count: 2 })
    })
  })

  it('should clear error on successful retry', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()
    let shouldFail = true

    server.use(
      http.get('http://localhost:8080/api/analytics/retry', () => {
        if (shouldFail) {
          return HttpResponse.json(
            { message: 'Temporary error' },
            { status: 500 }
          )
        }
        return HttpResponse.json({ success: true })
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

    // Fix the error and refetch
    shouldFail = false
    await result.current.refetch()

    await waitFor(() => {
      expect(result.current.error).toBeNull()
    })

    expect(result.current.data).toEqual({ success: true })

    consoleErrorSpy.mockRestore()
  })

  it('should include credentials in fetch request', async () => {
    const fetchSpy = jest.spyOn(global, 'fetch')

    server.use(
      http.get('http://localhost:8080/api/analytics/credentials', () => {
        return HttpResponse.json({ authenticated: true })
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
