import { renderHook, waitFor, act, cleanup } from '@testing-library/react'
import { useAnalyticsWebSocket } from '../useAnalyticsWebSocket'

// Mock WebSocket with proper state management
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3
  static instances: MockWebSocket[] = []

  readyState = MockWebSocket.CONNECTING
  onopen: ((event: any) => void) | null = null
  onmessage: ((event: any) => void) | null = null
  onerror: ((event: any) => void) | null = null
  onclose: ((event: any) => void) | null = null

  constructor(public url: string) {
    MockWebSocket.instances.push(this)

    // Simulate async connection
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN
      if (this.onopen) {
        this.onopen({ type: 'open' })
      }
    }, 10)
  }

  close() {
    this.readyState = MockWebSocket.CLOSING
    setTimeout(() => {
      this.readyState = MockWebSocket.CLOSED
      if (this.onclose) {
        this.onclose({ type: 'close' })
      }
    }, 0)
  }

  send(data: string) {
    if (this.readyState !== MockWebSocket.OPEN) {
      throw new Error('WebSocket is not open')
    }
  }

  // Helper to trigger message from tests
  simulateMessage(data: any) {
    if (this.onmessage) {
      this.onmessage({
        type: 'message',
        data: typeof data === 'string' ? data : JSON.stringify(data)
      })
    }
  }

  // Helper to trigger error from tests
  simulateError(error: any) {
    if (this.onerror) {
      this.onerror({ type: 'error', error })
    }
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
    // Clear all instances
    MockWebSocket.instances = []
    jest.clearAllMocks()
    cleanup()
  })

  it('should connect to WebSocket on mount', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    }, { timeout: 1000 })

    expect(result.current.data).toBeNull()
    expect(result.current.error).toBeNull()
  })

  it('should receive and parse WebSocket messages', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    }, { timeout: 1000 })

    // Get the actual WebSocket instance created by the hook
    const ws = MockWebSocket.instances[0] as MockWebSocket

    // Simulate receiving a message
    act(() => {
      ws.simulateMessage({
        currentVisitors: 42,
        timestamp: Date.now(),
        recentEvents: [{ type: 'page_view', id: '1' }],
      })
    })

    await waitFor(() => {
      expect(result.current.data).toBeTruthy()
      expect(result.current.data?.currentVisitors).toBe(42)
    }, { timeout: 1000 })
  })

  it('should handle invalid JSON in messages', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    }, { timeout: 1000 })

    // Get the actual WebSocket instance created by the hook
    const ws = MockWebSocket.instances[0] as MockWebSocket

    // Simulate receiving invalid JSON
    act(() => {
      ws.simulateMessage('invalid json')
    })

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalled()
    }, { timeout: 1000 })

    consoleErrorSpy.mockRestore()
  })

  it('should handle WebSocket errors', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    }, { timeout: 1000 })

    // Get the actual WebSocket instance created by the hook
    const ws = MockWebSocket.instances[0] as MockWebSocket

    // Simulate WebSocket error
    act(() => {
      ws.simulateError('Connection error')
    })

    await waitFor(() => {
      expect(result.current.error).toBeTruthy()
      expect(result.current.error?.message).toBe('WebSocket error')
    }, { timeout: 1000 })

    consoleErrorSpy.mockRestore()
  })

  it('should handle disconnection', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    }, { timeout: 1000 })

    // Get the actual WebSocket instance created by the hook
    const ws = MockWebSocket.instances[0] as MockWebSocket

    // Simulate disconnection
    act(() => {
      ws.close()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(false)
    }, { timeout: 1000 })
  })

  it('should reconnect when reconnect is called', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    }, { timeout: 1000 })

    // Get the actual WebSocket instance created by the hook
    const ws = MockWebSocket.instances[0] as MockWebSocket

    // Disconnect
    act(() => {
      ws.close()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(false)
    }, { timeout: 1000 })

    // Reconnect
    act(() => {
      result.current.reconnect()
    })

    // Wait for new connection (new instance will be created)
    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
      expect(MockWebSocket.instances.length).toBe(2)
    }, { timeout: 1000 })
  })

  it('should close WebSocket on unmount', async () => {
    const { result, unmount } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    }, { timeout: 1000 })

    // Verify WebSocket instance was created
    expect(MockWebSocket.instances.length).toBe(1)
    const ws = MockWebSocket.instances[0] as MockWebSocket
    expect(ws.readyState).toBe(MockWebSocket.OPEN)

    // Spy on the WebSocket's close method
    const closeSpy = jest.spyOn(ws, 'close')

    // Unmount the hook
    unmount()

    // Wait for cleanup to execute
    await new Promise(resolve => setTimeout(resolve, 50))

    // WebSocket close should have been called during cleanup
    expect(closeSpy).toHaveBeenCalled()

    closeSpy.mockRestore()
  })

  it('should construct correct WebSocket URL', async () => {
    const originalEnv = process.env.NEXT_PUBLIC_API_URL

    process.env.NEXT_PUBLIC_API_URL = 'http://example.com:8080'

    const { result } = renderHook(() => useAnalyticsWebSocket())

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    }, { timeout: 1000 })

    // Get the actual WebSocket instance created by the hook
    const ws = MockWebSocket.instances[0] as MockWebSocket

    // WebSocket should be created with ws:// protocol
    expect(ws.url).toBe('ws://example.com:8080/ws/analytics')

    process.env.NEXT_PUBLIC_API_URL = originalEnv
  })
})
