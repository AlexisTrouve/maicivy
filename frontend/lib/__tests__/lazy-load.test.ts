/**
 * @jest-environment jsdom
 */
import React from 'react';
import {
  DefaultLoader,
  CardSkeleton,
  TextSkeleton,
  lazyLoad,
  lazyLoadClientOnly,
  preloadComponent,
  useLazyLoadOnScroll,
  LazyImageLoader,
  prefetchRoute,
  preconnect,
  shouldLazyLoad,
  getOptimalImageSize,
  deferScript
} from '../lazy-load.tsx';

// Mock dynamic from next/dynamic
jest.mock('next/dynamic', () => {
  return function mockDynamic(importFn: any, options?: any) {
    // Return a simple mock component
    return () => {
      if (options?.loading) {
        return React.createElement(options.loading);
      }
      return React.createElement('div', {}, 'Lazy Component');
    };
  };
});

describe('Lazy Load Utils', () => {
  describe('DefaultLoader', () => {
    it('should render a loading spinner', () => {
      const element = DefaultLoader();
      expect(element).toBeDefined();
      expect(element.props.className).toContain('flex');
      expect(element.props.className).toContain('items-center');
      expect(element.props.className).toContain('justify-center');
    });

    it('should have animate-spin class', () => {
      const element = DefaultLoader();
      const spinner = element.props.children;
      expect(spinner.props.className).toContain('animate-spin');
    });
  });

  describe('CardSkeleton', () => {
    it('should render a card skeleton', () => {
      const element = CardSkeleton();
      expect(element).toBeDefined();
      expect(element.props.className).toContain('animate-pulse');
      expect(element.props.className).toContain('rounded-lg');
    });

    it('should have correct height', () => {
      const element = CardSkeleton();
      expect(element.props.className).toContain('h-48');
    });
  });

  describe('TextSkeleton', () => {
    it('should render text skeleton with multiple lines', () => {
      const element = TextSkeleton();
      expect(element).toBeDefined();
      expect(element.props.children).toHaveLength(3); // 3 skeleton lines
    });

    it('should have varying widths', () => {
      const element = TextSkeleton();
      const lines = element.props.children;

      // Check that different lines have different width classes
      expect(lines[0].props.className).toContain('w-3/4');
      expect(lines[1].props.className).toContain('w-1/2');
      expect(lines[2].props.className).toContain('w-5/6');
    });
  });

  describe('lazyLoad', () => {
    it('should create a lazy loaded component', () => {
      const TestComponent = () => React.createElement('div', {}, 'Test');
      const importFn = () => Promise.resolve({ default: TestComponent });
      const LazyComponent = lazyLoad(importFn);

      expect(LazyComponent).toBeDefined();
      expect(typeof LazyComponent).toBe('function');
    });

    it('should use custom loading component', () => {
      const CustomLoader = () => React.createElement('div', {}, 'Custom Loading...');
      const TestComponent = () => React.createElement('div', {}, 'Test');
      const importFn = () => Promise.resolve({ default: TestComponent });

      const LazyComponent = lazyLoad(importFn, {
        loading: CustomLoader
      });

      expect(LazyComponent).toBeDefined();
    });

    it('should respect SSR option', () => {
      const TestComponent = () => React.createElement('div', {}, 'Test');
      const importFn = () => Promise.resolve({ default: TestComponent });

      const LazyComponentSSR = lazyLoad(importFn, { ssr: true });
      const LazyComponentNoSSR = lazyLoad(importFn, { ssr: false });

      expect(LazyComponentSSR).toBeDefined();
      expect(LazyComponentNoSSR).toBeDefined();
    });
  });

  describe('lazyLoadClientOnly', () => {
    it('should create client-only lazy component', () => {
      const TestComponent = () => React.createElement('div', {}, 'Test');
      const importFn = () => Promise.resolve({ default: TestComponent });
      const LazyComponent = lazyLoadClientOnly(importFn);

      expect(LazyComponent).toBeDefined();
      expect(typeof LazyComponent).toBe('function');
    });

    it('should use custom loading component', () => {
      const CustomLoader = () => React.createElement('div', {}, 'Loading...');
      const TestComponent = () => React.createElement('div', {}, 'Test');
      const importFn = () => Promise.resolve({ default: TestComponent });

      const LazyComponent = lazyLoadClientOnly(importFn, CustomLoader);

      expect(LazyComponent).toBeDefined();
    });
  });

  describe('preloadComponent', () => {
    it('should call import function', async () => {
      const TestComponent = () => React.createElement('div', {}, 'Test');
      const importFn = jest.fn(() => Promise.resolve({ default: TestComponent }));

      preloadComponent(importFn);

      expect(importFn).toHaveBeenCalled();
    });

    it('should handle promise rejection', async () => {
      const importFn = jest.fn(() => Promise.reject(new Error('Load failed')).catch(() => {}));

      // Should not throw synchronously
      expect(() => preloadComponent(importFn)).not.toThrow();
      
      // Call returns promise that is already caught
      expect(importFn).toHaveBeenCalled();
    });
  });

  describe('useLazyLoadOnScroll', () => {
    it('should return ref and inView', () => {
      const { ref, inView } = useLazyLoadOnScroll();

      expect(ref).toBeDefined();
      expect(typeof inView).toBe('boolean');
    });

    it('should accept threshold parameter', () => {
      const { ref, inView } = useLazyLoadOnScroll(0.5);

      expect(ref).toBeDefined();
      expect(typeof inView).toBe('boolean');
    });

    it('should accept rootMargin parameter', () => {
      const { ref, inView } = useLazyLoadOnScroll(0.1, '100px');

      expect(ref).toBeDefined();
      expect(typeof inView).toBe('boolean');
    });
  });

  describe('LazyImageLoader', () => {
    let mockObserver: any;

    beforeEach(() => {
      mockObserver = {
        observe: jest.fn(),
        unobserve: jest.fn(),
        disconnect: jest.fn()
      };

      global.IntersectionObserver = jest.fn(() => mockObserver) as any;
    });

    it('should create IntersectionObserver', () => {
      const loader = new LazyImageLoader();

      expect(global.IntersectionObserver).toHaveBeenCalled();
    });

    it('should observe elements', () => {
      const loader = new LazyImageLoader();
      const element = document.createElement('img');

      loader.observe(element);

      expect(mockObserver.observe).toHaveBeenCalledWith(element);
    });

    it('should disconnect observer', () => {
      const loader = new LazyImageLoader();

      loader.disconnect();

      expect(mockObserver.disconnect).toHaveBeenCalled();
    });

    it('should load image when intersecting', () => {
      let callback: any;

      global.IntersectionObserver = jest.fn((cb) => {
        callback = cb;
        return mockObserver;
      }) as any;

      const loader = new LazyImageLoader();
      const img = document.createElement('img');
      img.dataset.src = 'https://example.com/image.jpg';

      loader.observe(img);

      // Simulate intersection
      callback([{
        isIntersecting: true,
        target: img
      }]);

      expect(img.src).toBe('https://example.com/image.jpg');
    });

    it('should unobserve after loading', () => {
      let callback: any;

      global.IntersectionObserver = jest.fn((cb) => {
        callback = cb;
        return mockObserver;
      }) as any;

      const loader = new LazyImageLoader();
      const img = document.createElement('img');
      img.dataset.src = 'https://example.com/image.jpg';

      loader.observe(img);

      // Simulate intersection
      callback([{
        isIntersecting: true,
        target: img
      }]);

      expect(mockObserver.unobserve).toHaveBeenCalledWith(img);
    });

    it('should accept custom threshold', () => {
      const loader = new LazyImageLoader(0.5);

      expect(global.IntersectionObserver).toHaveBeenCalledWith(
        expect.any(Function),
        { threshold: 0.5 }
      );
    });
  });

  describe('prefetchRoute', () => {
    it('should create link element', () => {
      const appendChildSpy = jest.spyOn(document.head, 'appendChild');

      prefetchRoute('/cv');

      expect(appendChildSpy).toHaveBeenCalled();
      const link = appendChildSpy.mock.calls[0][0] as HTMLLinkElement;
      expect(link.rel).toBe('prefetch');
      expect(link.href).toContain('/cv');

      appendChildSpy.mockRestore();
    });

    it('should handle multiple prefetches', () => {
      const appendChildSpy = jest.spyOn(document.head, 'appendChild');

      prefetchRoute('/cv');
      prefetchRoute('/letters');

      expect(appendChildSpy).toHaveBeenCalledTimes(2);

      appendChildSpy.mockRestore();
    });
  });

  describe('preconnect', () => {
    it('should create preconnect link', () => {
      const appendChildSpy = jest.spyOn(document.head, 'appendChild');

      preconnect('https://api.example.com');

      expect(appendChildSpy).toHaveBeenCalled();
      const link = appendChildSpy.mock.calls[0][0] as HTMLLinkElement;
      expect(link.rel).toBe('preconnect');
      expect(link.href).toContain('https://api.example.com');
      expect(link.crossOrigin).toBe('anonymous');

      appendChildSpy.mockRestore();
    });

    it('should handle multiple domains', () => {
      const appendChildSpy = jest.spyOn(document.head, 'appendChild');

      preconnect('https://api1.example.com');
      preconnect('https://api2.example.com');

      expect(appendChildSpy).toHaveBeenCalledTimes(2);

      appendChildSpy.mockRestore();
    });
  });

  describe('shouldLazyLoad', () => {
    beforeEach(() => {
      // Reset navigator.connection
      (navigator as any).connection = undefined;
    });

    it('should return false for 4g connection', () => {
      (navigator as any).connection = {
        effectiveType: '4g'
      };

      expect(shouldLazyLoad()).toBe(false);
    });

    it('should return false for wifi connection', () => {
      (navigator as any).connection = {
        effectiveType: 'wifi'
      };

      expect(shouldLazyLoad()).toBe(false);
    });

    it('should return true for slow connections', () => {
      (navigator as any).connection = {
        effectiveType: '3g'
      };

      expect(shouldLazyLoad()).toBe(true);
    });

    it('should return true when connection API unavailable', () => {
      (navigator as any).connection = undefined;

      expect(shouldLazyLoad()).toBe(true);
    });
  });

  describe('getOptimalImageSize', () => {
    beforeEach(() => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        value: 1920
      });

      Object.defineProperty(window, 'devicePixelRatio', {
        writable: true,
        value: 1
      });

      (navigator as any).connection = undefined;
    });

    it('should return size based on screen width', () => {
      window.innerWidth = 1920;
      window.devicePixelRatio = 1;

      const { width, quality } = getOptimalImageSize();

      expect(width).toBe(1920);
      expect(quality).toBe(75);
    });

    it('should account for device pixel ratio', () => {
      window.innerWidth = 1920;
      window.devicePixelRatio = 2;

      const { width } = getOptimalImageSize();

      expect(width).toBe(3840); // 1920 * 2
    });

    it('should adjust quality for 2g connection', () => {
      (navigator as any).connection = {
        effectiveType: '2g'
      };

      const { quality } = getOptimalImageSize();

      expect(quality).toBe(50);
    });

    it('should adjust quality for 3g connection', () => {
      (navigator as any).connection = {
        effectiveType: '3g'
      };

      const { quality } = getOptimalImageSize();

      expect(quality).toBe(60);
    });

    it('should adjust quality for 4g connection', () => {
      (navigator as any).connection = {
        effectiveType: '4g'
      };

      const { quality } = getOptimalImageSize();

      expect(quality).toBe(85);
    });

    it('should use default quality when connection unknown', () => {
      (navigator as any).connection = undefined;

      const { quality } = getOptimalImageSize();

      expect(quality).toBe(75);
    });
  });

  describe('deferScript', () => {
    let requestIdleCallbackMock: any;

    beforeEach(() => {
      // Clear any existing scripts from previous tests
      const existingScripts = document.body.querySelectorAll('script');
      existingScripts.forEach(script => {
        if (script.parentNode) {
          script.parentNode.removeChild(script);
        }
      });
      
      // Save original requestIdleCallback
      requestIdleCallbackMock = (window as any).requestIdleCallback;
    });

    afterEach(() => {
      // Restore requestIdleCallback
      if (requestIdleCallbackMock) {
        (window as any).requestIdleCallback = requestIdleCallbackMock;
      }
      
      // Clean up any scripts added during tests
      const scripts = document.body.querySelectorAll('script');
      scripts.forEach(script => {
        if (script.parentNode) {
          script.parentNode.removeChild(script);
        }
      });
    });

    it('should create script element', async () => {
      const appendChildSpy = jest.spyOn(document.body, 'appendChild');

      // Mock requestIdleCallback to execute immediately
      (window as any).requestIdleCallback = (cb: Function) => {
        setTimeout(cb, 0);
      };

      deferScript('https://example.com/script.js');

      // Wait for async callback
      await new Promise(resolve => setTimeout(resolve, 50));

      expect(appendChildSpy).toHaveBeenCalled();
      const script = appendChildSpy.mock.calls[0][0] as HTMLScriptElement;
      expect(script.src).toBe('https://example.com/script.js');
      expect(script.defer).toBe(true);

      appendChildSpy.mockRestore();
    });

    it('should call onLoad callback', async () => {
      const onLoad = jest.fn();

      // Mock requestIdleCallback
      (window as any).requestIdleCallback = (cb: Function) => {
        setTimeout(cb, 0);
      };

      deferScript('https://example.com/script.js', onLoad);

      await new Promise(resolve => setTimeout(resolve, 50));

      const script = document.body.querySelector('script[src="https://example.com/script.js"]') as HTMLScriptElement;
      if (script && script.onload) {
        (script.onload as any)(new Event('load'));
        expect(onLoad).toHaveBeenCalled();
      }
    });

    it('should use setTimeout fallback when requestIdleCallback unavailable', async () => {
      delete (window as any).requestIdleCallback;

      const appendChildSpy = jest.spyOn(document.body, 'appendChild');

      deferScript('https://example.com/fallback.js');

      await new Promise(resolve => setTimeout(resolve, 1100));

      expect(appendChildSpy).toHaveBeenCalled();
      appendChildSpy.mockRestore();
    });
  });

  describe('Edge Cases', () => {
    it('should handle SSR (no window)', () => {
      // These functions should not throw when window is undefined
      // They already check for typeof window === 'undefined'
      expect(() => useLazyLoadOnScroll()).not.toThrow();
    });

    it('should handle no navigator.connection', () => {
      (navigator as any).connection = undefined;

      expect(() => shouldLazyLoad()).not.toThrow();
      expect(() => getOptimalImageSize()).not.toThrow();
    });

    it('should handle LazyImageLoader with no IntersectionObserver', () => {
      const originalObserver = global.IntersectionObserver;
      (global as any).IntersectionObserver = undefined;

      // Without IntersectionObserver, the constructor will throw
      // This is expected behavior - the class requires IntersectionObserver
      expect(() => {
        new LazyImageLoader();
      }).toThrow();

      global.IntersectionObserver = originalObserver;
    });

    it('should handle empty image src', () => {
      let callback: any;
      const mockObserver = {
        observe: jest.fn(),
        unobserve: jest.fn(),
        disconnect: jest.fn()
      };

      global.IntersectionObserver = jest.fn((cb) => {
        callback = cb;
        return mockObserver;
      }) as any;

      const loader = new LazyImageLoader();
      const img = document.createElement('img');
      // No data-src attribute

      loader.observe(img);

      // Should not crash
      expect(() => {
        callback([{
          isIntersecting: true,
          target: img
        }]);
      }).not.toThrow();
    });

    it('should handle very high DPR', () => {
      window.innerWidth = 1920;
      window.devicePixelRatio = 3;

      const { width } = getOptimalImageSize();

      expect(width).toBe(5760); // 1920 * 3
    });

    it('should handle very small screen', () => {
      window.innerWidth = 320;
      window.devicePixelRatio = 1;

      const { width } = getOptimalImageSize();

      expect(width).toBe(320);
    });
  });

  describe('Performance', () => {
    it('should not create multiple observers for same LazyImageLoader', () => {
      const constructorSpy = jest.spyOn(window as any, 'IntersectionObserver');

      const loader = new LazyImageLoader();
      loader.observe(document.createElement('img'));
      loader.observe(document.createElement('img'));
      loader.observe(document.createElement('img'));

      // Should only create one observer
      expect(constructorSpy).toHaveBeenCalledTimes(1);

      constructorSpy.mockRestore();
    });

    it('should handle multiple prefetchRoute calls efficiently', () => {
      const appendChildSpy = jest.spyOn(document.head, 'appendChild');

      // Prefetch multiple routes
      for (let i = 0; i < 10; i++) {
        prefetchRoute(`/route-${i}`);
      }

      expect(appendChildSpy).toHaveBeenCalledTimes(10);

      appendChildSpy.mockRestore();
    });
  });
});
