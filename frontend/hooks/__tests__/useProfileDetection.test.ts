import { renderHook, waitFor, act } from '@testing-library/react'
import {
  useProfileDetection,
  useProfileDetectionManual,
  useBypassStatus,
  useProfileStats,
} from '../useProfileDetection'
import { profileApi } from '@/lib/api'

// Mock the profileApi module
jest.mock('@/lib/api', () => ({
  profileApi: {
    getCurrent: jest.fn(),
    detect: jest.fn(),
    getBypassStatus: jest.fn(),
    getStats: jest.fn(),
  },
}))

const mockedProfileApi = profileApi as jest.Mocked<typeof profileApi>

describe('useProfileDetection', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('should fetch profile detection on mount', async () => {
    mockedProfileApi.getCurrent.mockResolvedValueOnce({
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
    })

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

    mockedProfileApi.getCurrent.mockRejectedValueOnce(
      new Error('Detection failed')
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
    mockedProfileApi.getCurrent.mockResolvedValueOnce({
      profile_type: 'other',
      confidence: 10,
      bypass_enabled: false,
    })

    const { result } = renderHook(() => useProfileDetection())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.profileType).toBe('other')
    expect(result.current.isDetected).toBe(false) // Should be false for 'other'
  })

  it('should handle bypass enabled status', async () => {
    mockedProfileApi.getCurrent.mockResolvedValueOnce({
      profile_type: 'developer',
      confidence: 75,
      bypass_enabled: true,
    })

    const { result } = renderHook(() => useProfileDetection())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.bypassEnabled).toBe(true)
  })
})

describe('useProfileDetectionManual', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('should manually detect profile when detect is called', async () => {
    mockedProfileApi.detect.mockResolvedValueOnce({
      profile_type: 'cto',
      confidence: 90,
      enrichment_data: { company_name: 'Startup Inc' },
      bypass_enabled: false,
    })

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

    mockedProfileApi.detect.mockRejectedValueOnce(
      new Error('Detection error')
    )

    const { result } = renderHook(() => useProfileDetectionManual())

    // The detect function should throw on error, but we need to wrap it properly
    await act(async () => {
      try {
        await result.current.detect()
        // If we get here, the test should fail because we expected an error
        expect(true).toBe(false) // Force failure if no error thrown
      } catch (error) {
        // Error was thrown as expected - the state update happens inside act()
      }
    })

    // After act completes, the error state should be set
    expect(result.current.error).toBe('Detection failed')

    consoleErrorSpy.mockRestore()
  })
})

describe('useBypassStatus', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('should fetch bypass status on mount', async () => {
    mockedProfileApi.getBypassStatus.mockResolvedValueOnce({
      success: true,
      bypass: true,
    })

    const { result } = renderHook(() => useBypassStatus())

    // Initially loading should be true
    expect(result.current.loading).toBe(true)

    // Wait for the hook to finish loading
    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.bypassed).toBe(true)
  })

  it('should refresh bypass status when refresh is called', async () => {
    mockedProfileApi.getBypassStatus
      .mockResolvedValueOnce({
        success: true,
        bypass: false,
      })
      .mockResolvedValueOnce({
        success: true,
        bypass: true,
      })

    const { result } = renderHook(() => useBypassStatus())

    // Wait for hook to initialize
    await waitFor(() => {
      expect(result.current).toBeTruthy()
    })

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.bypassed).toBe(false)

    // Refresh to get new status
    await act(async () => {
      await result.current.refresh()
    })

    expect(result.current.bypassed).toBe(true)
  })

  it('should handle bypass check errors gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    mockedProfileApi.getBypassStatus.mockRejectedValueOnce(
      new Error('Error checking bypass')
    )

    const { result } = renderHook(() => useBypassStatus())

    // Wait for hook to initialize
    await waitFor(() => {
      expect(result.current).toBeTruthy()
    })

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    // Should default to false on error
    expect(result.current.bypassed).toBe(false)

    consoleErrorSpy.mockRestore()
  })
})

describe('useProfileStats', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('should fetch profile stats on mount', async () => {
    mockedProfileApi.getStats.mockResolvedValueOnce({
      stats_by_type: [
        { profile_type: 'recruiter', count: 50, avg_confidence: 80 },
        { profile_type: 'developer', count: 30, avg_confidence: 75 },
      ],
      total_detected: 80,
      total_visitors: 100,
      detection_rate: 0.8,
    })

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

    mockedProfileApi.getStats.mockRejectedValueOnce(
      new Error('Stats unavailable')
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
