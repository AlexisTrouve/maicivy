import { renderHook, act, waitFor } from '@testing-library/react'
import { useTimelineScroll, useScrollDirection, useScrollSnap } from '../useTimelineScroll'

describe('useTimelineScroll', () => {
  beforeEach(() => {
    // Mock window properties
    Object.defineProperty(window, 'innerHeight', {
      writable: true,
      configurable: true,
      value: 800,
    })

    Object.defineProperty(document.documentElement, 'scrollHeight', {
      writable: true,
      configurable: true,
      value: 3200,
    })

    Object.defineProperty(window, 'scrollY', {
      writable: true,
      configurable: true,
      value: 0,
    })

    // Mock IntersectionObserver
    global.IntersectionObserver = jest.fn().mockImplementation((callback) => ({
      observe: jest.fn(),
      disconnect: jest.fn(),
      unobserve: jest.fn(),
    }))

    // Mock scrollTo
    window.scrollTo = jest.fn()
  })

  it('should calculate scroll progress correctly', () => {
    const { result } = renderHook(() => useTimelineScroll())

    expect(result.current.scrollProgress).toBe(0)
    expect(result.current.isNearTop).toBe(true)
    expect(result.current.isNearBottom).toBe(false)

    // Simulate scroll to middle
    Object.defineProperty(window, 'scrollY', {
      writable: true,
      value: 1200, // Middle of page (scrollHeight 3200 - innerHeight 800 = 2400, half = 1200)
    })

    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    expect(result.current.scrollProgress).toBe(50)
    expect(result.current.isNearTop).toBe(false)
  })

  it('should detect near top correctly', () => {
    const { result } = renderHook(() => useTimelineScroll({ offset: 100 }))

    expect(result.current.isNearTop).toBe(true)

    // Scroll down 50px (still within offset)
    Object.defineProperty(window, 'scrollY', { writable: true, value: 50 })
    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    expect(result.current.isNearTop).toBe(true)

    // Scroll down 150px (beyond offset)
    Object.defineProperty(window, 'scrollY', { writable: true, value: 150 })
    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    expect(result.current.isNearTop).toBe(false)
  })

  it('should detect near bottom correctly', () => {
    const { result } = renderHook(() => useTimelineScroll({ offset: 100 }))

    // Scroll to near bottom (scrollHeight 3200 - innerHeight 800 - offset 100 = 2300)
    Object.defineProperty(window, 'scrollY', { writable: true, value: 2350 })
    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    expect(result.current.isNearBottom).toBe(true)
  })

  it('should scroll to specific year', () => {
    // Mock year element
    const mockElement = {
      getAttribute: jest.fn(() => '2023'),
      getBoundingClientRect: jest.fn(() => ({ top: 500 })),
    }

    document.querySelectorAll = jest.fn(() => [mockElement] as any)

    const { result } = renderHook(() => useTimelineScroll({ offset: 100 }))

    act(() => {
      result.current.scrollToYear(2023)
    })

    expect(window.scrollTo).toHaveBeenCalledWith({
      top: 400, // 500 (element top) + 0 (window.scrollY) - 100 (offset) = 400
      behavior: 'smooth',
    })
  })

  it('should scroll to top', () => {
    const { result } = renderHook(() => useTimelineScroll())

    act(() => {
      result.current.scrollToTop()
    })

    expect(window.scrollTo).toHaveBeenCalledWith({
      top: 0,
      behavior: 'smooth',
    })
  })

  it('should scroll to element by ID', () => {
    const mockElement = {
      getBoundingClientRect: jest.fn(() => ({ top: 300 })),
    }

    document.getElementById = jest.fn(() => mockElement as any)

    const { result } = renderHook(() => useTimelineScroll({ offset: 100 }))

    act(() => {
      result.current.scrollToElement('timeline-section-1')
    })

    expect(document.getElementById).toHaveBeenCalledWith('timeline-section-1')
    expect(window.scrollTo).toHaveBeenCalled()
  })

  it('should setup IntersectionObserver for sections', () => {
    // Create mock DOM sections
    const section1 = document.createElement('div')
    section1.setAttribute('data-timeline-section', 'true')
    const section2 = document.createElement('div')
    section2.setAttribute('data-timeline-section', 'true')

    document.body.appendChild(section1)
    document.body.appendChild(section2)

    renderHook(() => useTimelineScroll({ threshold: 0.5 }))

    expect(IntersectionObserver).toHaveBeenCalled()

    const observerInstance = (IntersectionObserver as jest.Mock).mock.results[0].value
    expect(observerInstance.observe).toHaveBeenCalled()

    // Cleanup
    document.body.removeChild(section1)
    document.body.removeChild(section2)
  })

  it('should cleanup on unmount', () => {
    // Create mock DOM sections
    const section1 = document.createElement('div')
    section1.setAttribute('data-timeline-section', 'true')
    const section2 = document.createElement('div')
    section2.setAttribute('data-timeline-section', 'true')

    document.body.appendChild(section1)
    document.body.appendChild(section2)

    const { unmount } = renderHook(() => useTimelineScroll())

    const observerInstance = (IntersectionObserver as jest.Mock).mock.results[0].value

    unmount()

    expect(observerInstance.disconnect).toHaveBeenCalled()

    // Cleanup
    document.body.removeChild(section1)
    document.body.removeChild(section2)
  })
})

describe('useScrollDirection', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'scrollY', {
      writable: true,
      configurable: true,
      value: 0,
    })
  })

  it('should detect scroll down', () => {
    const { result } = renderHook(() => useScrollDirection())

    expect(result.current).toBeNull()

    Object.defineProperty(window, 'scrollY', { writable: true, value: 100 })
    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    expect(result.current).toBe('down')
  })

  it('should detect scroll up', () => {
    Object.defineProperty(window, 'scrollY', { writable: true, value: 100 })

    const { result } = renderHook(() => useScrollDirection())

    // First scroll event to set lastScrollY = 100
    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    // Now scroll up
    Object.defineProperty(window, 'scrollY', { writable: true, value: 50 })
    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    expect(result.current).toBe('up')
  })
})

describe('useScrollSnap', () => {
  beforeEach(() => {
    jest.useFakeTimers()
    window.scrollTo = jest.fn()
  })

  afterEach(() => {
    act(() => {
      jest.runOnlyPendingTimers()
    })
    jest.useRealTimers()
  })

  it('should set isSnapping to true during scroll', () => {
    const { result } = renderHook(() => useScrollSnap())

    expect(result.current.isSnapping).toBe(false)

    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    expect(result.current.isSnapping).toBe(true)
  })

  it('should reset isSnapping after timeout', () => {
    const { result } = renderHook(() => useScrollSnap())

    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    expect(result.current.isSnapping).toBe(true)

    act(() => {
      jest.advanceTimersByTime(150)
    })

    expect(result.current.isSnapping).toBe(false)
  })

  it('should find and snap to closest element', () => {
    const mockElements = [
      {
        getBoundingClientRect: jest.fn(() => ({ top: 50 })),
        scrollIntoView: jest.fn(),
      },
      {
        getBoundingClientRect: jest.fn(() => ({ top: 200 })),
        scrollIntoView: jest.fn(),
      },
    ]

    document.querySelectorAll = jest.fn(() => mockElements as any)

    renderHook(() => useScrollSnap(100))

    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })

    act(() => {
      jest.advanceTimersByTime(150)
    })

    // Should snap to closest element (first one with top: 50)
    expect(mockElements[0].scrollIntoView).toHaveBeenCalledWith({
      behavior: 'smooth',
      block: 'start',
    })
  })
})
