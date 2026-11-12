import { renderHook, waitFor } from '@testing-library/react'
import { useVisitCount } from '../useVisitCount'
import { server } from '@/__mocks__/server'
import { rest } from 'msw'

describe('useVisitCount', () => {
  beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should fetch visit status from API on mount', async () => {
    server.use(
      rest.get('*/api/v1/visitors/check', (req, res, ctx) => {
        return res(ctx.json({
          visitCount: 2,
          hasAccess: true,
          remainingVisits: 1,
          sessionId: 'test-session-123',
        }))
      })
    )

    const { result } = renderHook(() => useVisitCount())

    expect(result.current.loading).toBe(true)
    expect(result.current.status).toBeNull()

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.status).toEqual({
      visitCount: 2,
      hasAccess: true,
      remainingVisits: 1,
      sessionId: 'test-session-123',
    })
    expect(result.current.error).toBeNull()
  })

  it('should indicate no access when visit count >= 3', async () => {
    server.use(
      rest.get('*/api/v1/visitors/check', (req, res, ctx) => {
        return res(ctx.json({
          visitCount: 3,
          hasAccess: false,
          remainingVisits: 0,
          sessionId: 'test-session-456',
        }))
      })
    )

    const { result } = renderHook(() => useVisitCount())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.status?.hasAccess).toBe(false)
    expect(result.current.status?.visitCount).toBe(3)
    expect(result.current.status?.remainingVisits).toBe(0)
  })

  it('should handle API error gracefully with fallback access', async () => {
    let requestCount = 0
    server.use(
      rest.get('*/api/v1/visitors/check', (req, res, ctx) => {
        requestCount++
        return res(ctx.status(500), ctx.json({ message: 'Server error' }))
      })
    )

    const { result } = renderHook(() => useVisitCount())

    // Wait for retries to complete (3 retries with exponential backoff)
    await waitFor(
      () => {
        expect(result.current.loading).toBe(false)
      },
      { timeout: 10000 }
    )

    expect(result.current.error).toBeTruthy()
    expect(result.current.status).toEqual({
      visitCount: 0,
      hasAccess: true,
      remainingVisits: 0,
      sessionId: '',
    })
  })

  it('should handle network error gracefully', async () => {
    server.use(
      rest.get('*/api/v1/visitors/check', (req, res) => {
        return res.networkError('Network error')
      })
    )

    const { result } = renderHook(() => useVisitCount())

    // Wait for retries to complete
    await waitFor(
      () => {
        expect(result.current.loading).toBe(false)
      },
      { timeout: 10000 }
    )

    expect(result.current.error).toBeTruthy()
    expect(result.current.status?.hasAccess).toBe(true)
  })

  it('should refresh visit status when refresh is called', async () => {
    let successfulCalls = 0
    server.use(
      rest.get('*/api/v1/visitors/check', (req, res, ctx) => {
        // Only increment on successful response (not on retries)
        successfulCalls++
        return res(
          ctx.json({
            visitCount: successfulCalls,
            hasAccess: successfulCalls < 3,
            remainingVisits: Math.max(0, 3 - successfulCalls),
            sessionId: `session-${successfulCalls}`,
          })
        )
      })
    )

    const { result } = renderHook(() => useVisitCount())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.status?.visitCount).toBe(1)
    expect(result.current.status?.sessionId).toBe('session-1')

    // Call refresh
    result.current.refresh()

    // Wait for loading to start and finish
    await waitFor(() => {
      expect(result.current.status?.visitCount).toBe(2)
    })

    expect(result.current.status?.sessionId).toBe('session-2')
    expect(result.current.loading).toBe(false)
  })

  it('should set loading state correctly during fetch', async () => {
    const { result } = renderHook(() => useVisitCount())

    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.loading).toBe(false)
  })

  it('should clear error on successful retry after error', async () => {
    let shouldFail = true
    server.use(
      rest.get('*/api/v1/visitors/check', (req, res, ctx) => {
        if (shouldFail) {
          return res(ctx.status(500), ctx.json({ message: 'Temporary error' }))
        }
        return res(
          ctx.json({
            visitCount: 1,
            hasAccess: true,
            remainingVisits: 2,
            sessionId: 'success-session',
          })
        )
      })
    )

    const { result } = renderHook(() => useVisitCount())

    // Wait for retries to complete
    await waitFor(
      () => {
        expect(result.current.loading).toBe(false)
      },
      { timeout: 10000 }
    )

    expect(result.current.error).toBeTruthy()

    // Now make it succeed
    shouldFail = false
    result.current.refresh()

    await waitFor(() => {
      expect(result.current.error).toBeNull()
    })

    expect(result.current.status?.visitCount).toBe(1)
    expect(result.current.status?.hasAccess).toBe(true)
    expect(result.current.loading).toBe(false)
  })
})
