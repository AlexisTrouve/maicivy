/**
 * Optimized Image component
 * Wrapper around Next.js Image with additional optimizations
 */

import Image, { ImageProps } from 'next/image'
import { useState, useEffect } from 'react'

interface OptimizedImageProps extends Omit<ImageProps, 'placeholder'> {
  /**
   * Show blur placeholder while loading
   */
  showPlaceholder?: boolean

  /**
   * Fallback image if src fails to load
   */
  fallback?: string

  /**
   * Custom blur data URL
   */
  blurDataURL?: string
}

/**
 * OptimizedImage component with automatic lazy loading and placeholders
 */
export function OptimizedImage({
  src,
  alt,
  showPlaceholder = true,
  fallback = '/images/placeholder.png',
  blurDataURL,
  priority = false,
  ...props
}: OptimizedImageProps) {
  const [imgSrc, setImgSrc] = useState(src)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    setImgSrc(src)
  }, [src])

  // Generate blur placeholder if not provided
  const placeholder = showPlaceholder ? 'blur' : undefined
  const blurData = blurDataURL || generateBlurPlaceholder()

  return (
    <div className="relative overflow-hidden">
      <Image
        src={imgSrc}
        alt={alt}
        onLoad={() => setIsLoading(false)}
        onError={() => {
          setImgSrc(fallback)
          setIsLoading(false)
        }}
        placeholder={placeholder as any}
        blurDataURL={blurData}
        priority={priority}
        loading={priority ? undefined : 'lazy'}
        {...props}
        className={`transition-opacity duration-300 ${
          isLoading ? 'opacity-0' : 'opacity-100'
        } ${props.className || ''}`}
      />
    </div>
  )
}

/**
 * Generate a simple blur placeholder
 */
function generateBlurPlaceholder(): string {
  // Tiny 1x1 gray pixel as base64
  return 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mN8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=='
}

/**
 * Responsive Image with automatic srcset
 */
export function ResponsiveImage({
  src,
  alt,
  sizes,
  ...props
}: OptimizedImageProps) {
  return (
    <OptimizedImage
      src={src}
      alt={alt}
      sizes={sizes || '(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw'}
      {...props}
    />
  )
}

/**
 * Avatar image with circular styling
 */
export function AvatarImage({
  src,
  alt,
  size = 40,
  ...props
}: OptimizedImageProps & { size?: number }) {
  return (
    <div
      className="rounded-full overflow-hidden"
      style={{ width: size, height: size }}
    >
      <OptimizedImage
        src={src}
        alt={alt}
        width={size}
        height={size}
        {...props}
        className="object-cover"
      />
    </div>
  )
}

/**
 * Background Image with overlay
 */
export function BackgroundImage({
  src,
  alt,
  overlay = 'bg-black/50',
  children,
  ...props
}: OptimizedImageProps & {
  overlay?: string
  children?: React.ReactNode
}) {
  return (
    <div className="relative w-full h-full">
      <OptimizedImage
        src={src}
        alt={alt}
        fill
        {...props}
        className="object-cover"
      />
      {overlay && <div className={`absolute inset-0 ${overlay}`} />}
      {children && (
        <div className="absolute inset-0 z-10">{children}</div>
      )}
    </div>
  )
}

/**
 * Gallery Image with lightbox support
 */
export function GalleryImage({
  src,
  alt,
  onClick,
  ...props
}: OptimizedImageProps & {
  onClick?: () => void
}) {
  return (
    <div
      className="cursor-pointer hover:opacity-90 transition-opacity"
      onClick={onClick}
    >
      <OptimizedImage
        src={src}
        alt={alt}
        {...props}
        className="rounded-lg shadow-lg"
      />
    </div>
  )
}

/**
 * Logo image with automatic dark mode support
 */
export function LogoImage({
  srcLight,
  srcDark,
  alt,
  ...props
}: Omit<OptimizedImageProps, 'src'> & {
  srcLight: string
  srcDark: string
}) {
  return (
    <>
      {/* Light mode logo */}
      <OptimizedImage
        src={srcLight}
        alt={alt}
        {...props}
        className={`dark:hidden ${props.className || ''}`}
      />
      {/* Dark mode logo */}
      <OptimizedImage
        src={srcDark}
        alt={alt}
        {...props}
        className={`hidden dark:block ${props.className || ''}`}
      />
    </>
  )
}

/**
 * Product image with zoom on hover
 */
export function ZoomableImage({
  src,
  alt,
  ...props
}: OptimizedImageProps) {
  return (
    <div className="overflow-hidden rounded-lg">
      <OptimizedImage
        src={src}
        alt={alt}
        {...props}
        className={`transition-transform duration-300 hover:scale-110 ${
          props.className || ''
        }`}
      />
    </div>
  )
}

/**
 * Image with loading skeleton
 */
export function ImageWithSkeleton({
  src,
  alt,
  skeletonClassName = 'bg-gray-200 dark:bg-gray-700',
  ...props
}: OptimizedImageProps & {
  skeletonClassName?: string
}) {
  const [isLoading, setIsLoading] = useState(true)

  return (
    <div className="relative">
      {isLoading && (
        <div
          className={`absolute inset-0 animate-pulse ${skeletonClassName}`}
        />
      )}
      <OptimizedImage
        src={src}
        alt={alt}
        onLoad={() => setIsLoading(false)}
        {...props}
      />
    </div>
  )
}

/**
 * Lazy loaded image with Intersection Observer
 */
export function LazyImage({
  src,
  alt,
  threshold = 0.1,
  ...props
}: OptimizedImageProps & {
  threshold?: number
}) {
  const [shouldLoad, setShouldLoad] = useState(false)

  useEffect(() => {
    if (typeof window === 'undefined') return

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setShouldLoad(true)
          observer.disconnect()
        }
      },
      { threshold }
    )

    const element = document.getElementById(`lazy-image-${src}`)
    if (element) {
      observer.observe(element)
    }

    return () => observer.disconnect()
  }, [src, threshold])

  return (
    <div id={`lazy-image-${src}`}>
      {shouldLoad ? (
        <OptimizedImage src={src} alt={alt} {...props} />
      ) : (
        <div className="bg-gray-200 dark:bg-gray-700 animate-pulse w-full h-full" />
      )}
    </div>
  )
}
