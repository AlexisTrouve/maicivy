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

// Polyfill for fetch using undici (for Node < 18 or when not available)
if (typeof global.fetch === 'undefined') {
  const { fetch, Request, Response, Headers, FormData } = require('undici')
  global.fetch = fetch
  global.Request = Request
  global.Response = Response
  global.Headers = Headers
  global.FormData = FormData
}

// Mock Service Worker setup
// NOTE: MSW v2 has compatibility issues with Jest in some environments
// If you need API mocking, import and setup server manually in your test files
// Example:
//   import { server } from '@/__mocks__/server'
//   beforeAll(() => server.listen())
//   afterEach(() => server.resetHandlers())
//   afterAll(() => server.close())

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

// Mock IntersectionObserver
global.IntersectionObserver = class IntersectionObserver {
  constructor() {}
  disconnect() {}
  observe() {}
  takeRecords() {
    return []
  }
  unobserve() {}
}

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
