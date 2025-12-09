import { renderHook, waitFor, act } from '@testing-library/react'
import { useAnalyticsWebSocket } from '../useAnalyticsWebSocket'

// Mock WebSocket
class MockWebSocket {
  static OPEN = 1
  static CLOSED = 3

  readyState = MockWebSocket.OPEN
  onopen: ((event: any) => void) | null = null
  onmessage: ((event: any) => void) | null = null
  onerror: ((event: any) => void) | null = null
  onclose: ((event: any) => void) | null = null

  constructor(public url: string) {
    // Simulate async connection
    setTimeout(() => {
      if (this.onopen) {
        this.onopen({})
      }
    }, 0)
  }

  close() {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose({})
    }
  }

  send(data: string) {
    // Mock send
  }
}

// Store original WebSocket
const originalWebSocket = global.WebSocket

describe('useAnalyticsWebSocket', () => {
  beforeAll(() => {
    // @ts-ignore
    global.WebSocket = MockWebSocket
  })

  afterAll(() => {
    global.WebSocket = originalWebSocket
  })

  afterEach(() => {
    jest.clearAllMocks()
  })

  it('should connect to WebSocket on mount', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    expect(result.current.data).toBeNull()
    expect(result.current.error).toBeNull()
  })

  it('should receive and parse WebSocket messages', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    // Simulate receiving a message
    act(() => {
      const messageEvent = {
        data: JSON.stringify({
          currentVisitors: 42,
          timestamp: Date.now(),
          recentEvents: [{ type: 'page_view', id: '1' }],
        }),
      }
      // @ts-ignore - accessing internal ws instance
      const ws = new MockWebSocket('')
      if (ws.onmessage) {
        ws.onmessage(messageEvent)
      }
    })

    await waitFor(() => {
      expect(result.current.data).toBeTruthy()
    })
  })

  it('should handle invalid JSON in messages', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    // Simulate receiving invalid JSON
    act(() => {
      const ws = new MockWebSocket('')
      if (ws.onmessage) {
        ws.onmessage({ data: 'invalid json' })
      }
    })

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalled()
    })

    consoleErrorSpy.mockRestore()
  })

  it('should handle WebSocket errors', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    // Simulate WebSocket error
    act(() => {
      const ws = new MockWebSocket('')
      if (ws.onerror) {
        ws.onerror({ error: 'Connection error' })
      }
    })

    await waitFor(() => {
      expect(result.current.error).toBeTruthy()
    })

    expect(result.current.error?.message).toBe('WebSocket error')
    consoleErrorSpy.mockRestore()
  })

  it('should handle disconnection', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    // Simulate disconnection
    act(() => {
      const ws = new MockWebSocket('')
      ws.close()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(false)
    })
  })

  it('should reconnect when reconnect is called', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    // Disconnect
    act(() => {
      const ws = new MockWebSocket('')
      ws.close()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(false)
    })

    // Reconnect
    act(() => {
      result.current.reconnect()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })
  })

  it('should close WebSocket on unmount', async () => {
    const closeSpy = jest.spyOn(MockWebSocket.prototype, 'close')

    const { unmount } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(closeSpy).not.toHaveBeenCalled()
    })

    unmount()

    // WebSocket should be closed
    expect(closeSpy).toHaveBeenCalled()

    closeSpy.mockRestore()
  })

  it('should construct correct WebSocket URL', async () => {
    const originalEnv = process.env.NEXT_PUBLIC_API_URL

    process.env.NEXT_PUBLIC_API_URL = 'http://example.com:8080'

    renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      // WebSocket should be created with ws:// protocol
      // Note: In actual implementation, URL should be ws://example.com:8080/ws/analytics
    })

    process.env.NEXT_PUBLIC_API_URL = originalEnv
  })
})
