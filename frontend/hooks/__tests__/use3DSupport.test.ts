import { renderHook } from '@testing-library/react'
import { use3DSupport, useHas3DSupport, use3DQualitySettings } from '../use3DSupport'

describe('use3DSupport', () => {
  let mockCanvas: any
  let mockGLContext: any

  beforeEach(() => {
    mockGLContext = {
      getExtension: jest.fn(),
      getParameter: jest.fn(),
    }

    mockCanvas = {
      getContext: jest.fn((type) => {
        if (type === 'webgl2' || type === 'experimental-webgl2') {
          return mockGLContext
        }
        if (type === 'webgl' || type === 'experimental-webgl') {
          return mockGLContext
        }
        return null
      }),
    }

    document.createElement = jest.fn((tag) => {
      if (tag === 'canvas') {
        return mockCanvas
      }
      return {} as any
    })

    // Mock navigator.userAgent
    Object.defineProperty(navigator, 'userAgent', {
      writable: true,
      configurable: true,
      value: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    })
  })

  it('should detect WebGL 2.0 support on desktop', () => {
    mockGLContext.getExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGLContext.getParameter.mockReturnValue('NVIDIA GeForce GTX 1080')

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isSupported).toBe(true)
    expect(result.current.webGLVersion).toBe(2)
    expect(result.current.performanceLevel).toBe('high')
    expect(result.current.isMobile).toBe(false)
  })

  it('should detect WebGL 1.0 support when WebGL 2.0 is unavailable', () => {
    mockCanvas.getContext = jest.fn((type) => {
      if (type === 'webgl2' || type === 'experimental-webgl2') {
        return null
      }
      if (type === 'webgl' || type === 'experimental-webgl') {
        return mockGLContext
      }
      return null
    })

    mockGLContext.getExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGLContext.getParameter.mockReturnValue('Intel HD Graphics')

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isSupported).toBe(true)
    expect(result.current.webGLVersion).toBe(1)
    expect(result.current.performanceLevel).toBe('medium')
  })

  it('should detect mobile devices', () => {
    Object.defineProperty(navigator, 'userAgent', {
      writable: true,
      value: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)',
    })

    mockGLContext.getExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGLContext.getParameter.mockReturnValue('Apple A14 GPU')

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isMobile).toBe(true)
    expect(result.current.performanceLevel).toBe('medium')
  })

  it('should not support 3D on low-end mobile devices', () => {
    Object.defineProperty(navigator, 'userAgent', {
      writable: true,
      value: 'Mozilla/5.0 (Linux; Android 9; SM-G960F)',
    })

    mockGLContext.getExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGLContext.getParameter.mockReturnValue('Mali-G72')

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isMobile).toBe(true)
    expect(result.current.performanceLevel).toBe('low')
    expect(result.current.isSupported).toBe(false)
    expect(result.current.reason).toBe('Low-end mobile device')
  })

  it('should handle WebGL not available', () => {
    mockCanvas.getContext = jest.fn(() => null)

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isSupported).toBe(false)
    expect(result.current.webGLVersion).toBeNull()
    expect(result.current.reason).toBe('WebGL context creation failed')
  })

  it('should handle WebGL context creation errors', () => {
    mockCanvas.getContext = jest.fn(() => {
      throw new Error('WebGL not supported')
    })

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isSupported).toBe(false)
    expect(result.current.reason).toBe('WebGL not available')
  })

  it('should adjust performance level based on device memory', () => {
    Object.defineProperty(navigator, 'deviceMemory', {
      writable: true,
      configurable: true,
      value: 2, // 2GB RAM
    })

    mockGLContext.getExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGLContext.getParameter.mockReturnValue('NVIDIA GeForce GTX 1080')

    const { result } = renderHook(() => use3DSupport())

    // Should be low because of low RAM even with good GPU
    expect(result.current.performanceLevel).toBe('low')
  })
})

describe('useHas3DSupport', () => {
  beforeEach(() => {
    const mockGLContext = {
      getExtension: jest.fn(),
      getParameter: jest.fn(),
    }

    const mockCanvas = {
      getContext: jest.fn(() => mockGLContext),
    }

    document.createElement = jest.fn(() => mockCanvas as any)

    mockGLContext.getExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGLContext.getParameter.mockReturnValue('NVIDIA GeForce GTX 1080')
  })

  it('should return boolean support status', () => {
    const { result } = renderHook(() => useHas3DSupport())

    expect(typeof result.current).toBe('boolean')
    expect(result.current).toBe(true)
  })
})

describe('use3DQualitySettings', () => {
  beforeEach(() => {
    const mockGLContext = {
      getExtension: jest.fn().mockReturnValue({
        UNMASKED_RENDERER_WEBGL: 37446,
      }),
      getParameter: jest.fn(),
    }

    const mockCanvas = {
      getContext: jest.fn(() => mockGLContext),
    }

    document.createElement = jest.fn(() => mockCanvas as any)

    Object.defineProperty(window, 'devicePixelRatio', {
      writable: true,
      configurable: true,
      value: 2,
    })
  })

  it('should return high quality settings for high performance', () => {
    const mockGLContext = {
      getExtension: jest.fn().mockReturnValue({
        UNMASKED_RENDERER_WEBGL: 37446,
      }),
      getParameter: jest.fn(() => 'NVIDIA GeForce RTX 3080'),
    }

    const mockCanvas = {
      getContext: jest.fn(() => mockGLContext),
    }

    document.createElement = jest.fn(() => mockCanvas as any)

    const { result } = renderHook(() => use3DQualitySettings())

    expect(result.current.antialias).toBe(true)
    expect(result.current.shadows).toBe(true)
    expect(result.current.particleCount).toBe(1000)
    expect(result.current.maxFPS).toBe(60)
    expect(result.current.pixelRatio).toBe(2)
  })

  it('should return medium quality settings for medium performance', () => {
    const mockGLContext = {
      getExtension: jest.fn().mockReturnValue({
        UNMASKED_RENDERER_WEBGL: 37446,
      }),
      getParameter: jest.fn(() => 'Intel UHD Graphics'),
    }

    const mockCanvas = {
      getContext: jest.fn(() => mockGLContext),
    }

    document.createElement = jest.fn(() => mockCanvas as any)

    const { result } = renderHook(() => use3DQualitySettings())

    expect(result.current.antialias).toBe(true)
    expect(result.current.shadows).toBe(false)
    expect(result.current.particleCount).toBe(500)
    expect(result.current.maxFPS).toBe(45)
    expect(result.current.pixelRatio).toBe(1)
  })

  it('should return low quality settings for low performance', () => {
    const mockGLContext = {
      getExtension: jest.fn().mockReturnValue({
        UNMASKED_RENDERER_WEBGL: 37446,
      }),
      getParameter: jest.fn(() => 'Intel HD Graphics 4000'),
    }

    const mockCanvas = {
      getContext: jest.fn(() => mockGLContext),
    }

    document.createElement = jest.fn(() => mockCanvas as any)

    Object.defineProperty(navigator, 'userAgent', {
      writable: true,
      value: 'Mozilla/5.0 (iPhone; CPU iPhone OS 12_0 like Mac OS X)',
    })

    const { result } = renderHook(() => use3DQualitySettings())

    expect(result.current.antialias).toBe(false)
    expect(result.current.shadows).toBe(false)
    expect(result.current.particleCount).toBe(200)
    expect(result.current.maxFPS).toBe(30)
    expect(result.current.pixelRatio).toBe(1)
  })
})
