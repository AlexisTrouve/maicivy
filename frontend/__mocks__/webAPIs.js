/**
 * Global WebAPIs mocks for Jest tests
 * These mocks provide basic functionality for WebAPIs not available in Node.js environment
 */

// Mock WebSocket
global.WebSocket = class WebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  constructor(url) {
    this.url = url
    this.readyState = WebSocket.CONNECTING
    this.onopen = null
    this.onclose = null
    this.onerror = null
    this.onmessage = null
    this.protocol = ''
    this.bufferedAmount = 0
    this.extensions = ''
    this.binaryType = 'blob'

    // Simulate async connection
    setTimeout(() => {
      this.readyState = WebSocket.OPEN
      if (this.onopen) {
        this.onopen({ type: 'open' })
      }
    }, 0)
  }

  send(data) {
    if (this.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket is not open')
    }
    // Mock send - do nothing
  }

  close(code = 1000, reason = '') {
    if (this.readyState === WebSocket.CLOSED || this.readyState === WebSocket.CLOSING) {
      return
    }

    this.readyState = WebSocket.CLOSING
    setTimeout(() => {
      this.readyState = WebSocket.CLOSED
      if (this.onclose) {
        this.onclose({ type: 'close', code, reason, wasClean: true })
      }
    }, 0)
  }

  addEventListener(event, listener) {
    this[`on${event}`] = listener
  }

  removeEventListener(event, listener) {
    if (this[`on${event}`] === listener) {
      this[`on${event}`] = null
    }
  }
}

// Enhance IntersectionObserver mock (already exists in jest.setup.js, this provides more functionality)
const IntersectionObserverMock = class IntersectionObserver {
  constructor(callback, options = {}) {
    this.callback = callback
    this.options = options
    this.elements = new Map()
  }

  observe(element) {
    // Store element
    this.elements.set(element, true)

    // Immediately trigger callback with element visible by default
    // Tests can override this behavior if needed
    setTimeout(() => {
      this.callback([{
        target: element,
        isIntersecting: true,
        intersectionRatio: 1,
        boundingClientRect: element.getBoundingClientRect ? element.getBoundingClientRect() : {},
        intersectionRect: {},
        rootBounds: {},
        time: Date.now(),
      }], this)
    }, 0)
  }

  unobserve(element) {
    this.elements.delete(element)
  }

  disconnect() {
    this.elements.clear()
  }

  takeRecords() {
    return []
  }
}

// Only override if not already defined (allow tests to use their own mocks)
if (!global.IntersectionObserver || global.IntersectionObserver.toString().includes('[native code]')) {
  global.IntersectionObserver = IntersectionObserverMock
}

// Mock WebGL context
const webGLContextMock = {
  canvas: null,
  drawingBufferWidth: 300,
  drawingBufferHeight: 150,

  // WebGL methods
  getParameter: jest.fn((pname) => {
    // Return reasonable defaults
    if (pname === 37446) return 'WebGL Mock Renderer' // UNMASKED_RENDERER_WEBGL
    if (pname === 37445) return 'WebKit' // UNMASKED_VENDOR_WEBGL
    if (pname === 3379) return 16384 // MAX_TEXTURE_SIZE
    if (pname === 34921) return 16 // MAX_COMBINED_TEXTURE_IMAGE_UNITS
    if (pname === 3386) return [4096, 4096] // MAX_VIEWPORT_DIMS
    return null
  }),

  getExtension: jest.fn((name) => {
    // Mock common extensions
    if (name === 'WEBGL_debug_renderer_info') {
      return {
        UNMASKED_VENDOR_WEBGL: 37445,
        UNMASKED_RENDERER_WEBGL: 37446
      }
    }
    if (name === 'WEBGL_lose_context') {
      return {
        loseContext: jest.fn(),
        restoreContext: jest.fn()
      }
    }
    return null
  }),

  getSupportedExtensions: jest.fn(() => [
    'WEBGL_debug_renderer_info',
    'WEBGL_lose_context',
    'OES_texture_float',
    'OES_standard_derivatives'
  ]),

  // Context creation methods
  createBuffer: jest.fn(() => ({})),
  createTexture: jest.fn(() => ({})),
  createProgram: jest.fn(() => ({})),
  createShader: jest.fn(() => ({})),
  createFramebuffer: jest.fn(() => ({})),
  createRenderbuffer: jest.fn(() => ({})),

  // Shader methods
  shaderSource: jest.fn(),
  compileShader: jest.fn(),
  getShaderParameter: jest.fn(() => true),
  getShaderInfoLog: jest.fn(() => ''),

  // Program methods
  attachShader: jest.fn(),
  linkProgram: jest.fn(),
  getProgramParameter: jest.fn(() => true),
  getProgramInfoLog: jest.fn(() => ''),
  useProgram: jest.fn(),

  // Attribute/Uniform methods
  getAttribLocation: jest.fn(() => 0),
  getUniformLocation: jest.fn(() => ({})),
  enableVertexAttribArray: jest.fn(),
  vertexAttribPointer: jest.fn(),

  // Drawing methods
  clear: jest.fn(),
  clearColor: jest.fn(),
  drawArrays: jest.fn(),
  drawElements: jest.fn(),

  // State methods
  enable: jest.fn(),
  disable: jest.fn(),
  blendFunc: jest.fn(),
  depthFunc: jest.fn(),

  // Buffer methods
  bindBuffer: jest.fn(),
  bufferData: jest.fn(),

  // Texture methods
  bindTexture: jest.fn(),
  texImage2D: jest.fn(),
  texParameteri: jest.fn(),

  // Viewport
  viewport: jest.fn(),

  // Error handling
  getError: jest.fn(() => 0), // NO_ERROR
}

// Mock Canvas.getContext to support WebGL
const originalGetContext = HTMLCanvasElement.prototype.getContext
if (!originalGetContext || originalGetContext.toString().includes('[native code]')) {
  HTMLCanvasElement.prototype.getContext = function(contextType, contextAttributes) {
    if (contextType === 'webgl' || contextType === 'experimental-webgl' ||
        contextType === 'webgl2' || contextType === 'experimental-webgl2') {
      const ctx = { ...webGLContextMock }
      ctx.canvas = this
      return ctx
    }

    if (contextType === '2d') {
      return {
        canvas: this,
        fillRect: jest.fn(),
        clearRect: jest.fn(),
        getImageData: jest.fn(() => ({ data: [] })),
        putImageData: jest.fn(),
        createImageData: jest.fn(() => ({ data: [] })),
        setTransform: jest.fn(),
        drawImage: jest.fn(),
        save: jest.fn(),
        fillText: jest.fn(),
        restore: jest.fn(),
        beginPath: jest.fn(),
        moveTo: jest.fn(),
        lineTo: jest.fn(),
        closePath: jest.fn(),
        stroke: jest.fn(),
        translate: jest.fn(),
        scale: jest.fn(),
        rotate: jest.fn(),
        arc: jest.fn(),
        fill: jest.fn(),
        measureText: jest.fn(() => ({ width: 0 })),
        transform: jest.fn(),
        rect: jest.fn(),
        clip: jest.fn(),
      }
    }

    return null
  }
}

// Mock navigator.connection (Network Information API)
if (!navigator.connection) {
  Object.defineProperty(navigator, 'connection', {
    writable: true,
    configurable: true,
    value: {
      effectiveType: '4g',
      downlink: 10,
      rtt: 50,
      saveData: false,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
    }
  })
}

// Mock navigator.deviceMemory
if (typeof navigator.deviceMemory === 'undefined') {
  Object.defineProperty(navigator, 'deviceMemory', {
    writable: true,
    configurable: true,
    value: 8, // 8GB RAM by default
  })
}

// Mock requestIdleCallback / cancelIdleCallback
if (typeof window.requestIdleCallback === 'undefined') {
  window.requestIdleCallback = function(callback, options) {
    const start = Date.now()
    return setTimeout(() => {
      callback({
        didTimeout: false,
        timeRemaining: () => Math.max(0, 50 - (Date.now() - start))
      })
    }, 1)
  }
}

if (typeof window.cancelIdleCallback === 'undefined') {
  window.cancelIdleCallback = function(id) {
    clearTimeout(id)
  }
}

// Mock requestAnimationFrame / cancelAnimationFrame (usually provided by jsdom but ensure they exist)
if (typeof window.requestAnimationFrame === 'undefined') {
  window.requestAnimationFrame = function(callback) {
    return setTimeout(() => callback(Date.now()), 16)
  }
}

if (typeof window.cancelAnimationFrame === 'undefined') {
  window.cancelAnimationFrame = function(id) {
    clearTimeout(id)
  }
}

module.exports = {
  WebSocket: global.WebSocket,
  IntersectionObserver: global.IntersectionObserver,
}
