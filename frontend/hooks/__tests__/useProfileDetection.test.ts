import { renderHook, waitFor, act } from '@testing-library/react'
import {
  useProfileDetection,
  useProfileDetectionManual,
  useBypassStatus,
  useProfileStats,
} from '../useProfileDetection'
import { server } from '@/__mocks__/server'
import { rest } from 'msw'

describe('useProfileDetection', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should fetch profile detection on mount', async () => {
    server.use(
      rest.get('/api/v1/profile/current', (req, res, ctx) => {
        return res(ctx.json({
          profile_type: 'recruiter',
          confidence: 85,
          enrichment_data: {
            company_name: 'Tech Corp',
            job_title: 'Senior Recruiter',
          },
          device_info: {
            browser: 'Chrome',
            os: 'Windows',
            deviceType: 'desktop',
            isBot: false,
          },
          detection_sources: ['linkedin', 'clearbit'],
          bypass_enabled: false,
        }))
      })
    )

    const { result } = renderHook(() => useProfileDetection())

    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.profileType).toBe('recruiter')
    expect(result.current.confidence).toBe(85)
    expect(result.current.isDetected).toBe(true)
    expect(result.current.enrichmentData?.company_name).toBe('Tech Corp')
    expect(result.current.deviceInfo?.browser).toBe('Chrome')
    expect(result.current.error).toBeNull()
  })

  it('should handle profile detection failure gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    server.use(
      rest.get('/api/v1/profile/current', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json(
          { message: 'Detection failed' }
        ))
      })
    )

    const { result } = renderHook(() => useProfileDetection())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('Failed to load profile')
    expect(result.current.profileType).toBe('other')
    expect(result.current.isDetected).toBe(false)

    consoleErrorSpy.mockRestore()
  })

  it('should correctly detect when profile is "other" with low confidence', async () => {
    server.use(
      rest.get('/api/v1/profile/current', (req, res, ctx) => {
        return res(ctx.json({
          profile_type: 'other',
          confidence: 10,
          bypass_enabled: false,
        }))
      })
    )

    const { result } = renderHook(() => useProfileDetection())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.profileType).toBe('other')
    expect(result.current.isDetected).toBe(false) // Should be false for 'other'
  })

  it('should handle bypass enabled status', async () => {
    server.use(
      rest.get('/api/v1/profile/current', (req, res, ctx) => {
        return res(ctx.json({
          profile_type: 'developer',
          confidence: 75,
          bypass_enabled: true,
        }))
      })
    )

    const { result } = renderHook(() => useProfileDetection())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.bypassEnabled).toBe(true)
  })
})

describe('useProfileDetectionManual', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should manually detect profile when detect is called', async () => {
    server.use(
      rest.get('/api/v1/profile/detect', (req, res, ctx) => {
        return res(ctx.json({
          profile_type: 'cto',
          confidence: 90,
          enrichment_data: { company_name: 'Startup Inc' },
          bypass_enabled: false,
        }))
      })
    )

    const { result } = renderHook(() => useProfileDetectionManual())

    expect(result.current.profileData).toBeNull()
    expect(result.current.loading).toBe(false)

    await act(async () => {
      await result.current.detect()
    })

    expect(result.current.loading).toBe(false)
    expect(result.current.profileData?.profileType).toBe('cto')
    expect(result.current.profileData?.confidence).toBe(90)
    expect(result.current.error).toBeNull()
  })

  it('should handle detection errors', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    server.use(
      rest.get('/api/v1/profile/detect', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json(
          { message: 'Detection error' }
        ))
      })
    )

    const { result } = renderHook(() => useProfileDetectionManual())

    await expect(
      act(async () => {
        await result.current.detect()
      })
    ).rejects.toThrow()

    expect(result.current.error).toBe('Detection failed')

    consoleErrorSpy.mockRestore()
  })
})

describe('useBypassStatus', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should fetch bypass status on mount', async () => {
    server.use(
      rest.get('/api/v1/profile/bypass', (req, res, ctx) => {
        return res(ctx.json({
          success: true,
          bypass: true,
        }))
      })
    )

    const { result } = renderHook(() => useBypassStatus())

    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.bypassed).toBe(true)
  })

  it('should refresh bypass status when refresh is called', async () => {
    let bypassValue = false

    server.use(
      rest.get('/api/v1/profile/bypass', (req, res, ctx) => {
        return res(ctx.json({
          success: true,
          bypass: bypassValue,
        }))
      })
    )

    const { result } = renderHook(() => useBypassStatus())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.bypassed).toBe(false)

    // Change bypass status and refresh
    bypassValue = true

    await act(async () => {
      await result.current.refresh()
    })

    expect(result.current.bypassed).toBe(true)
  })

  it('should handle bypass check errors gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    server.use(
      rest.get('/api/v1/profile/bypass', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json(
          { message: 'Error checking bypass' }
        ))
      })
    )

    const { result } = renderHook(() => useBypassStatus())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    // Should default to false on error
    expect(result.current.bypassed).toBe(false)

    consoleErrorSpy.mockRestore()
  })
})

describe('useProfileStats', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should fetch profile stats on mount', async () => {
    server.use(
      rest.get('/api/v1/profile/stats', (req, res, ctx) => {
        return res(ctx.json({
          stats_by_type: [
            { profile_type: 'recruiter', count: 50, avg_confidence: 80 },
            { profile_type: 'developer', count: 30, avg_confidence: 75 },
          ],
          total_detected: 80,
          total_visitors: 100,
          detection_rate: 0.8,
        }))
      })
    )

    const { result } = renderHook(() => useProfileStats())

    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.stats).toBeTruthy()
    expect(result.current.stats.total_detected).toBe(80)
    expect(result.current.stats.detection_rate).toBe(0.8)
    expect(result.current.error).toBeNull()
  })

  it('should handle stats fetch errors', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    server.use(
      rest.get('/api/v1/profile/stats', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json(
          { message: 'Stats unavailable' }
        ))
      })
    )

    const { result } = renderHook(() => useProfileStats())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('Failed to load stats')
    expect(result.current.stats).toBeNull()

    consoleErrorSpy.mockRestore()
  })
})
