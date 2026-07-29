/**
 * Lazy loading utilities for React components
 * Reduces initial bundle size by code splitting
 */

import dynamic from 'next/dynamic'
import { ComponentType } from 'react'

/**
 * Default loading component shown while lazy component loads
 */
export const DefaultLoader = () => (
  <div className="flex items-center justify-center p-8">
    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-gray-100"></div>
  </div>
)

/**
 * Skeleton loader for cards
 */
export const CardSkeleton = () => (
  <div className="bg-gray-200 dark:bg-gray-700 rounded-lg h-48 animate-pulse"></div>
)

/**
 * Skeleton loader for text content
 */
export const TextSkeleton = () => (
  <div className="space-y-3">
    <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded animate-pulse w-3/4"></div>
    <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded animate-pulse w-1/2"></div>
    <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded animate-pulse w-5/6"></div>
  </div>
)

/**
 * Creates a lazy-loaded component with custom loader
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function lazyLoad<T extends ComponentType<any>>(
  importFn: () => Promise<{ default: T }>,
  options?: {
    loading?: () => JSX.Element | null
    ssr?: boolean
  }
) {
  return dynamic(importFn, {
    loading: options?.loading ?? (() => <DefaultLoader />),
    ssr: options?.ssr ?? true, // SSR enabled by default
  })
}

/**
 * Lazy load component with no SSR (client-side only)
 * Useful for components that use browser-only APIs
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function lazyLoadClientOnly<T extends ComponentType<any>>(
  importFn: () => Promise<{ default: T }>,
  LoadingComponent?: () => JSX.Element | null
) {
  return dynamic(importFn, {
    loading: LoadingComponent ?? (() => <DefaultLoader />),
    ssr: false,
  })
}

/**
 * Preload a component before it's needed
 * Call this on hover or when user is likely to need the component
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function preloadComponent(importFn: () => Promise<any>) {
  // Webpack magic comment for prefetching
  importFn()
}

/**
 * Intersection Observer hook for lazy loading on scroll
 */
export function useLazyLoadOnScroll(
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _threshold: number = 0.1,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _rootMargin: string = '50px'
) {
  if (typeof window === 'undefined') {
    return { ref: null, inView: false }
  }

  // In production, use react-intersection-observer or implement full hook
  // This is a simplified version
  const ref = null
  const inView = false

  return { ref, inView }
}

/**
 * Lazy load images with Intersection Observer
 */
export class LazyImageLoader {
  private observer: IntersectionObserver | null = null

  constructor(threshold: number = 0.1) {
    if (typeof window !== 'undefined') {
      this.observer = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) {
              const img = entry.target as HTMLImageElement
              const src = img.dataset.src
              if (src) {
                img.src = src
                img.removeAttribute('data-src')
              }
              this.observer?.unobserve(img)
            }
          })
        },
        { threshold }
      )
    }
  }

  observe(element: HTMLElement) {
    this.observer?.observe(element)
  }

  disconnect() {
    this.observer?.disconnect()
  }
}

/**
 * Prefetch data for a route
 */
export function prefetchRoute(href: string) {
  if (typeof window !== 'undefined') {
    const link = document.createElement('link')
    link.rel = 'prefetch'
    link.href = href
    document.head.appendChild(link)
  }
}

/**
 * Preconnect to external domains
 */
export function preconnect(url: string) {
  if (typeof window !== 'undefined') {
    const link = document.createElement('link')
    link.rel = 'preconnect'
    link.href = url
    link.crossOrigin = 'anonymous'
    document.head.appendChild(link)
  }
}

/**
 * Check if component should be lazy loaded based on viewport
 */
export function shouldLazyLoad(): boolean {
  if (typeof window === 'undefined') return false

  // Don't lazy load on fast connections
  const connection = (navigator as any).connection
  if (connection) {
    const effectiveType = connection.effectiveType
    if (effectiveType === '4g' || effectiveType === 'wifi') {
      return false // Load immediately on fast connections
    }
  }

  // Lazy load on slow connections or mobile
  return true
}

/**
 * Get optimal image size based on device
 */
export function getOptimalImageSize(): { width: number; quality: number } {
  if (typeof window === 'undefined') {
    return { width: 1920, quality: 75 }
  }

  const screenWidth = window.innerWidth
  const dpr = window.devicePixelRatio || 1

  // Adjust quality based on connection
  const connection = (navigator as any).connection
  let quality = 75

  if (connection) {
    const effectiveType = connection.effectiveType
    if (effectiveType === '2g' || effectiveType === 'slow-2g') {
      quality = 50
    } else if (effectiveType === '3g') {
      quality = 60
    } else if (effectiveType === '4g') {
      quality = 85
    }
  }

  return {
    width: Math.round(screenWidth * dpr),
    quality,
  }
}

/**
 * Defer script loading until page is idle
 */
export function deferScript(src: string, onLoad?: () => void) {
  if (typeof window === 'undefined') return

  const script = document.createElement('script')
  script.src = src
  script.defer = true
  if (onLoad) {
    script.onload = onLoad
  }

  // Load when browser is idle
  if ('requestIdleCallback' in window) {
    requestIdleCallback(() => {
      document.body.appendChild(script)
    })
  } else {
    // Fallback for browsers without requestIdleCallback
    setTimeout(() => {
      document.body.appendChild(script)
    }, 1000)
  }
}
