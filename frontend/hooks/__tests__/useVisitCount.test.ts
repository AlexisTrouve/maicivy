import { renderHook, waitFor } from '@testing-library/react'
import { useVisitCount } from '../useVisitCount'
import { server } from '@/__mocks__/server'
import { rest } from 'msw'

describe('useVisitCount', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should fetch visit status from API on mount', async () => {
    server.use(
      rest.get('/api/v1/visitors/check', () => {
        return res(ctx.json({
          visitCount: 2,
          hasAccess: true,
          remainingVisits: 1,
          sessionId: 'test-session-123',
        })
      })
    )

    const { result } = renderHook(() => useVisitCount())

    // Initially loading
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
      rest.get('/api/v1/visitors/check', () => {
        return res(ctx.json({
          visitCount: 3,
          hasAccess: false,
          remainingVisits: 0,
          sessionId: 'test-session-456',
        })
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
    server.use(
      rest.get('/api/v1/visitors/check', () => {
        return res(ctx.json(
          { message: 'Server error' },
          { status: 500 }
        )
      })
    )

    const { result } = renderHook(() => useVisitCount())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    // Should have error
    expect(result.current.error).toBeTruthy()

    // Should fallback to allowing access (server will verify)
    expect(result.current.status).toEqual({
      visitCount: 0,
      hasAccess: true,
      remainingVisits: 0,
      sessionId: '',
    })
  })

  it('should handle network error gracefully', async () => {
    server.use(
      rest.get('/api/v1/visitors/check', () => {
        return HttpResponse.error()
      })
    )

    const { result } = renderHook(() => useVisitCount())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBeTruthy()
    expect(result.current.status?.hasAccess).toBe(true) // Fallback allows access
  })

  it('should refresh visit status when refresh is called', async () => {
    let callCount = 0
    server.use(
      rest.get('/api/v1/visitors/check', () => {
        callCount++
        return res(ctx.json({
          visitCount: callCount,
          hasAccess: callCount < 3,
          remainingVisits: Math.max(0, 3 - callCount),
          sessionId: `session-${callCount}`,
        })
      })
    )

    const { result } = renderHook(() => useVisitCount())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.status?.visitCount).toBe(1)

    // Call refresh
    result.current.refresh()

    await waitFor(() => {
      expect(result.current.status?.visitCount).toBe(2)
    })

    expect(result.current.status?.sessionId).toBe('session-2')
  })

  it('should set loading state correctly during fetch', async () => {
    const { result } = renderHook(() => useVisitCount())

    // Should start loading
    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    // Should not be loading after fetch completes
    expect(result.current.loading).toBe(false)
  })

  it('should clear error on successful retry after error', async () => {
    let shouldFail = true
    server.use(
      rest.get('/api/v1/visitors/check', () => {
        if (shouldFail) {
          return res(ctx.json(
            { message: 'Temporary error' },
            { status: 500 }
          )
        }
        return res(ctx.json({
          visitCount: 1,
          hasAccess: true,
          remainingVisits: 2,
          sessionId: 'success-session',
        })
      })
    )

    const { result } = renderHook(() => useVisitCount())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    // Should have error initially
    expect(result.current.error).toBeTruthy()

    // Fix the error and refresh
    shouldFail = false
    result.current.refresh()

    await waitFor(() => {
      expect(result.current.error).toBeNull()
    })

    expect(result.current.status?.visitCount).toBe(1)
    expect(result.current.status?.hasAccess).toBe(true)
  })
})
