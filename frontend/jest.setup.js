// Polyfills for Node.js environment
if (typeof global.TextEncoder === 'undefined') {
  const { TextEncoder, TextDecoder } = require('util')
  global.TextEncoder = TextEncoder
  global.TextDecoder = TextDecoder
}

// Polyfill for ReadableStream
if (typeof global.ReadableStream === 'undefined') {
  const { ReadableStream, WritableStream, TransformStream } = require('stream/web')
  global.ReadableStream = ReadableStream
  global.WritableStream = WritableStream
  global.TransformStream = TransformStream
}

// Polyfill for MessageChannel/MessagePort
if (typeof global.MessageChannel === 'undefined') {
  const { MessageChannel, MessagePort } = require('worker_threads')
  global.MessageChannel = MessageChannel
  global.MessagePort = MessagePort
}

// Learn more: https://github.com/testing-library/jest-dom
require('@testing-library/jest-dom')

// Setup environment variables for tests
process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080'
process.env.NEXT_PUBLIC_WS_URL = 'ws://localhost:8080'

// Polyfill fetch using undici for jsdom environment
// Even though Node.js 18+ has native fetch, jsdom doesn't have it
const { fetch, Request, Response, Headers, FormData } = require('undici')
global.fetch = fetch
global.Request = Request
global.Response = Response
global.Headers = Headers
global.FormData = FormData

// Polyfill for clearImmediate (needed by undici)
if (typeof global.clearImmediate === 'undefined') {
  global.clearImmediate = (id) => clearTimeout(id)
}
if (typeof window !== 'undefined' && typeof window.clearImmediate === 'undefined') {
  window.clearImmediate = (id) => clearTimeout(id)
}

// Polyfill for setImmediate (needed by undici)
if (typeof global.setImmediate === 'undefined') {
  global.setImmediate = (fn, ...args) => setTimeout(fn, 0, ...args)
}
if (typeof window !== 'undefined' && typeof window.setImmediate === 'undefined') {
  window.setImmediate = (fn, ...args) => setTimeout(fn, 0, ...args)
}

// Mock performance.markResourceTiming for undici (JSDOM doesn't have it)
if (typeof global.performance !== 'undefined' && typeof global.performance.markResourceTiming === 'undefined') {
  global.performance.markResourceTiming = () => {}
}
if (typeof window !== 'undefined' && typeof window.performance !== 'undefined' && typeof window.performance.markResourceTiming === 'undefined') {
  window.performance.markResourceTiming = () => {}
}

// Mock Service Worker setup
// NOTE: MSW v2 has compatibility issues with Jest in some environments
// If you need API mocking, import and setup server manually in your test files
// Example:
//   import { server } from '@/__mocks__/server'
//   beforeAll(() => server.listen())
//   afterEach(() => server.resetHandlers())
//   afterAll(() => server.close())

// Patch JSDOM timers to support .unref() method (needed by undici)
// JSDOM's setTimeout/setInterval don't have .unref() which is Node.js specific
const originalSetTimeout = global.setTimeout
const originalSetInterval = global.setInterval

global.setTimeout = function(...args) {
  const timer = originalSetTimeout.apply(this, args)
  if (timer && typeof timer === 'object') {
    timer.unref = timer.unref || function() { return this }
    timer.ref = timer.ref || function() { return this }
  }
  return timer
}

global.setInterval = function(...args) {
  const timer = originalSetInterval.apply(this, args)
  if (timer && typeof timer === 'object') {
    timer.unref = timer.unref || function() { return this }
    timer.ref = timer.ref || function() { return this }
  }
  return timer
}

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(), // deprecated
    removeListener: jest.fn(), // deprecated
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
})

// Import WebAPIs mocks (WebSocket, IntersectionObserver, WebGL, etc.)
require('./__mocks__/webAPIs')

// Mock ResizeObserver
global.ResizeObserver = class ResizeObserver {
  constructor() {}
  disconnect() {}
  observe() {}
  unobserve() {}
}

// Suppress console errors in tests (optional)
// global.console = {
//   ...console,
//   error: jest.fn(),
//   warn: jest.fn(),
// }

// Memory optimization: Aggressive garbage collection hints
if (global.gc) {
  // Run GC after each test
  afterEach(() => {
    global.gc()
  })
}

// Increase default timeout for async tests
jest.setTimeout(30000)

// Prevent memory leaks from unresolved promises
process.on('unhandledRejection', (reason) => {
  console.error('Unhandled Rejection at:', reason)
})
