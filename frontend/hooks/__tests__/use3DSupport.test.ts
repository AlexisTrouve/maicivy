import { renderHook } from '@testing-library/react'
import { use3DSupport, useHas3DSupport, use3DQualitySettings } from '../use3DSupport'

describe('use3DSupport', () => {
  let mockGetContext: jest.SpyInstance
  let mockGetExtension: jest.SpyInstance
  let mockGetParameter: jest.SpyInstance

  beforeEach(() => {
    // Spy on HTMLCanvasElement.prototype.getContext instead of mocking createElement
    mockGetContext = jest.spyOn(HTMLCanvasElement.prototype, 'getContext')

    // Create mock functions for GL context methods
    mockGetExtension = jest.fn()
    mockGetParameter = jest.fn()

    // Configure getContext to return our mocked GL context
    mockGetContext.mockImplementation(function(this: HTMLCanvasElement, type: string) {
      if (type === 'webgl2' || type === 'experimental-webgl2') {
        return {
          canvas: this,
          getExtension: mockGetExtension,
          getParameter: mockGetParameter,
        }
      }
      if (type === 'webgl' || type === 'experimental-webgl') {
        return {
          canvas: this,
          getExtension: mockGetExtension,
          getParameter: mockGetParameter,
        }
      }
      return null
    })

    // Mock navigator.userAgent
    Object.defineProperty(navigator, 'userAgent', {
      writable: true,
      configurable: true,
      value: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    })
  })

  afterEach(() => {
    mockGetContext.mockRestore()
  })

  it('should detect WebGL 2.0 support on desktop', () => {
    mockGetExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGetParameter.mockReturnValue('NVIDIA GeForce GTX 1080')

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isSupported).toBe(true)
    expect(result.current.webGLVersion).toBe(2)
    expect(result.current.performanceLevel).toBe('high')
    expect(result.current.isMobile).toBe(false)
  })

  it('should detect WebGL 1.0 support when WebGL 2.0 is unavailable', () => {
    mockGetContext.mockImplementation(function(this: HTMLCanvasElement, type: string) {
      if (type === 'webgl2' || type === 'experimental-webgl2') {
        return null
      }
      if (type === 'webgl' || type === 'experimental-webgl') {
        return {
          canvas: this,
          getExtension: mockGetExtension,
          getParameter: mockGetParameter,
        }
      }
      return null
    })

    mockGetExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGetParameter.mockReturnValue('Intel HD Graphics')

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

    mockGetExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGetParameter.mockReturnValue('Apple M1 GPU')

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isMobile).toBe(true)
    expect(result.current.performanceLevel).toBe('medium')
  })

  it('should not support 3D on low-end mobile devices', () => {
    Object.defineProperty(navigator, 'userAgent', {
      writable: true,
      value: 'Mozilla/5.0 (Linux; Android 9; SM-G960F)',
    })

    mockGetExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGetParameter.mockReturnValue('Mali-G72')

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isMobile).toBe(true)
    expect(result.current.performanceLevel).toBe('low')
    expect(result.current.isSupported).toBe(false)
    expect(result.current.reason).toBe('Low-end mobile device')
  })

  it('should handle WebGL not available', () => {
    mockGetContext.mockReturnValue(null)

    const { result } = renderHook(() => use3DSupport())

    expect(result.current.isSupported).toBe(false)
    expect(result.current.webGLVersion).toBeNull()
    expect(result.current.reason).toBe('WebGL context creation failed')
  })

  it('should handle WebGL context creation errors', () => {
    mockGetContext.mockImplementation(() => {
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

    mockGetExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGetParameter.mockReturnValue('NVIDIA GeForce GTX 1080')

    const { result } = renderHook(() => use3DSupport())

    // Should be low because of low RAM even with good GPU
    expect(result.current.performanceLevel).toBe('low')
  })
})

describe('useHas3DSupport', () => {
  let mockGetContext: jest.SpyInstance
  let mockGetExtension: jest.SpyInstance
  let mockGetParameter: jest.SpyInstance

  beforeEach(() => {
    mockGetContext = jest.spyOn(HTMLCanvasElement.prototype, 'getContext')
    mockGetExtension = jest.fn()
    mockGetParameter = jest.fn()

    mockGetContext.mockImplementation(function(this: HTMLCanvasElement, type: string) {
      if (type === 'webgl2' || type === 'experimental-webgl2' || type === 'webgl' || type === 'experimental-webgl') {
        return {
          canvas: this,
          getExtension: mockGetExtension,
          getParameter: mockGetParameter,
        }
      }
      return null
    })

    mockGetExtension.mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    mockGetParameter.mockReturnValue('NVIDIA GeForce GTX 1080')
  })

  afterEach(() => {
    mockGetContext.mockRestore()
  })

  it('should return boolean support status', () => {
    const { result } = renderHook(() => useHas3DSupport())

    expect(typeof result.current).toBe('boolean')
    expect(result.current).toBe(true)
  })
})

describe('use3DQualitySettings', () => {
  let mockGetContext: jest.SpyInstance

  beforeEach(() => {
    mockGetContext = jest.spyOn(HTMLCanvasElement.prototype, 'getContext')

    Object.defineProperty(window, 'devicePixelRatio', {
      writable: true,
      configurable: true,
      value: 2,
    })

    // Set desktop user agent by default
    Object.defineProperty(navigator, 'userAgent', {
      writable: true,
      configurable: true,
      value: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    })

    // Set device memory high enough to not limit performance
    Object.defineProperty(navigator, 'deviceMemory', {
      writable: true,
      configurable: true,
      value: 8,
    })
  })

  afterEach(() => {
    mockGetContext.mockRestore()
  })

  it('should return high quality settings for high performance', () => {
    const mockGetExtension = jest.fn().mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    const mockGetParameter = jest.fn(() => 'NVIDIA GeForce RTX 3080')

    mockGetContext.mockImplementation(function(this: HTMLCanvasElement, type: string) {
      if (type === 'webgl2' || type === 'experimental-webgl2' || type === 'webgl' || type === 'experimental-webgl') {
        return {
          canvas: this,
          getExtension: mockGetExtension,
          getParameter: mockGetParameter,
        }
      }
      return null
    })

    const { result } = renderHook(() => use3DQualitySettings())

    expect(result.current.antialias).toBe(true)
    expect(result.current.shadows).toBe(true)
    expect(result.current.particleCount).toBe(1000)
    expect(result.current.maxFPS).toBe(60)
    expect(result.current.pixelRatio).toBe(2)
  })

  it('should return medium quality settings for medium performance', () => {
    const mockGetExtension = jest.fn().mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    const mockGetParameter = jest.fn(() => 'Intel UHD Graphics')

    mockGetContext.mockImplementation(function(this: HTMLCanvasElement, type: string) {
      if (type === 'webgl2' || type === 'experimental-webgl2' || type === 'webgl' || type === 'experimental-webgl') {
        return {
          canvas: this,
          getExtension: mockGetExtension,
          getParameter: mockGetParameter,
        }
      }
      return null
    })

    const { result } = renderHook(() => use3DQualitySettings())

    expect(result.current.antialias).toBe(true)
    expect(result.current.shadows).toBe(false)
    expect(result.current.particleCount).toBe(500)
    expect(result.current.maxFPS).toBe(45)
    expect(result.current.pixelRatio).toBe(1)
  })

  it('should return low quality settings for low performance', () => {
    const mockGetExtension = jest.fn().mockReturnValue({
      UNMASKED_RENDERER_WEBGL: 37446,
    })
    const mockGetParameter = jest.fn(() => 'Intel HD Graphics 4000')

    mockGetContext.mockImplementation(function(this: HTMLCanvasElement, type: string) {
      if (type === 'webgl2' || type === 'experimental-webgl2' || type === 'webgl' || type === 'experimental-webgl') {
        return {
          canvas: this,
          getExtension: mockGetExtension,
          getParameter: mockGetParameter,
        }
      }
      return null
    })

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
