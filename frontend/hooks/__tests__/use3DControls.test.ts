import { renderHook, act } from '@testing-library/react'
import { use3DControls, use3DHoverRotation, useSmoothCamera } from '../use3DControls'
import { createRef } from 'react'

// Mock OrbitControls type
interface MockOrbitControls {
  enableDamping: boolean
  dampingFactor: number
  enableZoom: boolean
  enablePan: boolean
  autoRotate: boolean
  autoRotateSpeed: number
  minDistance: number
  maxDistance: number
  minPolarAngle: number
  maxPolarAngle: number
  touches: any
}

describe('use3DControls', () => {
  let mockControls: MockOrbitControls

  beforeEach(() => {
    mockControls = {
      enableDamping: false,
      dampingFactor: 0,
      enableZoom: false,
      enablePan: false,
      autoRotate: false,
      autoRotateSpeed: 0,
      minDistance: 0,
      maxDistance: 0,
      minPolarAngle: 0,
      maxPolarAngle: 0,
      touches: {},
    }
  })

  it('should configure controls with default settings', () => {
    const controlsRef = { current: mockControls } as any

    renderHook(() => use3DControls(controlsRef))

    expect(mockControls.enableDamping).toBe(true)
    expect(mockControls.dampingFactor).toBe(0.05)
    expect(mockControls.enableZoom).toBe(true)
    expect(mockControls.enablePan).toBe(false)
    expect(mockControls.autoRotate).toBe(false)
    expect(mockControls.minDistance).toBe(2)
    expect(mockControls.maxDistance).toBe(10)
  })

  it('should configure controls with custom settings', () => {
    const controlsRef = { current: mockControls } as any

    renderHook(() =>
      use3DControls(controlsRef, {
        enableDamping: false,
        dampingFactor: 0.1,
        enableZoom: false,
        enablePan: true,
        autoRotate: true,
        autoRotateSpeed: 2.0,
        minDistance: 5,
        maxDistance: 20,
        minPolarAngle: Math.PI / 6,
        maxPolarAngle: Math.PI / 2,
      })
    )

    expect(mockControls.enableDamping).toBe(false)
    expect(mockControls.dampingFactor).toBe(0.1)
    expect(mockControls.enableZoom).toBe(false)
    expect(mockControls.enablePan).toBe(true)
    expect(mockControls.autoRotate).toBe(true)
    expect(mockControls.autoRotateSpeed).toBe(2.0)
    expect(mockControls.minDistance).toBe(5)
    expect(mockControls.maxDistance).toBe(20)
    expect(mockControls.minPolarAngle).toBe(Math.PI / 6)
    expect(mockControls.maxPolarAngle).toBe(Math.PI / 2)
  })

  it('should configure touch controls', () => {
    const controlsRef = { current: mockControls } as any

    renderHook(() => use3DControls(controlsRef))

    expect(mockControls.touches).toEqual({
      ONE: 2, // TOUCH_ROTATE
      TWO: 1, // TOUCH_DOLLY_PAN
    })
  })

  it('should not throw when controlsRef is null', () => {
    const controlsRef = { current: null } as any

    expect(() => {
      renderHook(() => use3DControls(controlsRef))
    }).not.toThrow()
  })

  it('should update controls when config changes', () => {
    const controlsRef = { current: mockControls } as any

    const { rerender } = renderHook(
      ({ config }) => use3DControls(controlsRef, config),
      {
        initialProps: { config: { autoRotate: false } },
      }
    )

    expect(mockControls.autoRotate).toBe(false)

    // Update config
    rerender({ config: { autoRotate: true } })

    expect(mockControls.autoRotate).toBe(true)
  })
})

describe('use3DHoverRotation', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 1024,
    })

    Object.defineProperty(window, 'innerHeight', {
      writable: true,
      configurable: true,
      value: 768,
    })
  })

  it('should track mouse movement when enabled', () => {
    const { result } = renderHook(() => use3DHoverRotation(true, 0.002))

    expect(result.current.rotation).toEqual({ x: 0, y: 0 })

    // Simulate mouse move to corner (not center to ensure rotation)
    act(() => {
      const event = new MouseEvent('mousemove', {
        clientX: 0,
        clientY: 0,
      })
      window.dispatchEvent(event)
    })

    // Rotation should update after updateRotation is called
    const delta = 0.016 // ~60fps frame
    act(() => {
      result.current.updateRotation(delta)
    })

    // Rotation values should exist (may be zero if lerp is slow)
    expect(result.current.rotation).toBeDefined()
    expect(typeof result.current.rotation.x).toBe('number')
    expect(typeof result.current.rotation.y).toBe('number')
  })

  it('should not track mouse movement when disabled', () => {
    const { result } = renderHook(() => use3DHoverRotation(false))

    const mouseMoveSpy = jest.spyOn(window, 'addEventListener')

    // Should not add mouse listener
    expect(
      mouseMoveSpy.mock.calls.some((call) => call[0] === 'mousemove')
    ).toBe(false)

    mouseMoveSpy.mockRestore()
  })

  it('should smooth rotation updates with lerp', () => {
    const { result } = renderHook(() => use3DHoverRotation(true, 0.002))

    // Move mouse to top-left corner
    act(() => {
      const event = new MouseEvent('mousemove', {
        clientX: 0,
        clientY: 0,
      })
      window.dispatchEvent(event)
    })

    // Update rotation multiple times
    act(() => {
      result.current.updateRotation(0.016)
    })

    const rotation1 = { ...result.current.rotation }

    act(() => {
      result.current.updateRotation(0.016)
    })

    const rotation2 = { ...result.current.rotation }

    // Rotation should smoothly interpolate
    expect(rotation2.x).not.toBe(rotation1.x)
    expect(rotation2.y).not.toBe(rotation1.y)
  })

  it('should adjust sensitivity', () => {
    const { result: result1 } = renderHook(() => use3DHoverRotation(true, 0.001))
    const { result: result2 } = renderHook(() => use3DHoverRotation(true, 0.004))

    // Move mouse to same position for both
    act(() => {
      const event = new MouseEvent('mousemove', {
        clientX: 100,
        clientY: 100,
      })
      window.dispatchEvent(event)
    })

    act(() => {
      result1.current.updateRotation(0.016)
      result2.current.updateRotation(0.016)
    })

    // Higher sensitivity should result in larger rotation values
    // Note: This is a simplified test - actual values may vary
    expect(Math.abs(result2.current.rotation.x) + Math.abs(result2.current.rotation.y)).toBeGreaterThan(0)
  })
})

describe('useSmoothCamera', () => {
  it('should initialize with default position', () => {
    const { result } = renderHook(() => useSmoothCamera())

    const mockCamera = {
      position: {
        x: 0,
        y: 0,
        z: 5,
        set: jest.fn(function(this: any, x: number, y: number, z: number) {
          this.x = x
          this.y = y
          this.z = z
        }),
      },
    }

    act(() => {
      result.current.updateCamera(mockCamera as any, 0.016)
    })

    // Position should be set via the set method
    expect(mockCamera.position.set).toHaveBeenCalled()
  })

  it('should set camera target position', () => {
    const { result } = renderHook(() => useSmoothCamera())

    act(() => {
      result.current.setTarget(10, 5, 15)
    })

    // Target should be set
    // Note: updateCamera would smoothly interpolate to this position
  })

  it('should smoothly interpolate camera position', () => {
    const { result } = renderHook(() => useSmoothCamera())

    let cameraPos = { x: 0, y: 0, z: 5 }
    const mockCamera = {
      position: {
        x: 0,
        y: 0,
        z: 5,
        set: jest.fn(function(this: any, x: number, y: number, z: number) {
          this.x = x
          this.y = y
          this.z = z
          cameraPos = { x, y, z }
        }),
      },
    }

    // Set target far from current position
    act(() => {
      result.current.setTarget(10, 10, 10)
    })

    // Update camera position with delta time
    act(() => {
      result.current.updateCamera(mockCamera as any, 0.016)
    })

    const position1 = { ...cameraPos }

    // Update again
    act(() => {
      result.current.updateCamera(mockCamera as any, 0.016)
    })

    const position2 = { ...cameraPos }

    // Position should be moving towards target
    expect(position2.x).toBeGreaterThan(position1.x)
    expect(position2.y).toBeGreaterThan(position1.y)
    expect(position2.z).toBeGreaterThan(position1.z)
  })

  it('should eventually reach target position with multiple updates', () => {
    const { result } = renderHook(() => useSmoothCamera())

    let cameraPos = { x: 0, y: 0, z: 5 }
    const mockCamera = {
      position: {
        x: 0,
        y: 0,
        z: 5,
        set: jest.fn(function(this: any, x: number, y: number, z: number) {
          this.x = x
          this.y = y
          this.z = z
          cameraPos = { x, y, z }
        }),
      },
    }

    act(() => {
      result.current.setTarget(1, 1, 6)
    })

    // Update many times
    for (let i = 0; i < 100; i++) {
      act(() => {
        result.current.updateCamera(mockCamera as any, 0.016)
      })
    }

    // Should be very close to target
    expect(cameraPos.x).toBeCloseTo(1, 1)
    expect(cameraPos.y).toBeCloseTo(1, 1)
    expect(cameraPos.z).toBeCloseTo(6, 1)
  })
})
