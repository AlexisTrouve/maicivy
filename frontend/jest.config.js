const nextJest = require('next/jest')

const createJestConfig = nextJest({
  // Provide the path to your Next.js app to load next.config.js and .env files in your test environment
  dir: './',
})

// Add any custom config to be passed to Jest
const customJestConfig = {
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  testEnvironment: 'jest-environment-jsdom',
  moduleNameMapper: {
    // Handle module aliases (this will be automatically configured for you soon)
    '^@/(.*)$': '<rootDir>/$1',
    // Handle CSS imports (with CSS modules)
    '^.+\.module\.(css|sass|scss)$': 'identity-obj-proxy',
    // Handle CSS imports (without CSS modules)
    '^.+\.(css|sass|scss)$': '<rootDir>/__mocks__/styleMocks.js',
    // Handle image imports
    '^.+\.(png|jpg|jpeg|gif|webp|avif|ico|bmp|svg)$/i': '<rootDir>/__mocks__/fileMocks.js',
    // Mock three.js
    '^three$': '<rootDir>/__mocks__/three.ts',
    // Mock @radix-ui/react-slot
    '^@radix-ui/react-slot$': '<rootDir>/__mocks__/@radix-ui/react-slot.js',
    // Mock @radix-ui/react-select
    '^@radix-ui/react-select$': '<rootDir>/__mocks__/@radix-ui/react-select.tsx',
    // Mock lucide-react icons
    '^lucide-react$': '<rootDir>/__mocks__/lucide-react.tsx',
  },
  testPathIgnorePatterns: [
    '<rootDir>/.next/',
    '<rootDir>/node_modules/',
    '<rootDir>/tests/e2e/',
  ],
  transformIgnorePatterns: [
    'node_modules/(?!(@bundled-es-modules)/)',
    '^.+\.module\.(css|sass|scss)$',
    '/dist/',
    '/coverage/',
    '/build/',
  ],
  collectCoverageFrom: [
    'app/**/*.{js,jsx,ts,tsx}',
    'components/**/*.{js,jsx,ts,tsx}',
    'lib/**/*.{js,jsx,ts,tsx}',
    '!**/*.d.ts',
    '!**/node_modules/**',
    '!**/.next/**',
    '!**/coverage/**',
    '!**/jest.config.js',
    '!**/__tests__/**', // Exclude test files
    '!**/__mocks__/**', // Exclude mock files
  ],
  coverageThreshold: {
    global: {
      branches: 70,
      functions: 70,
      lines: 70,
      statements: 70,
    },
  },
  // Memory and performance optimizations
  // Run tests serially to prevent worker memory issues
  maxWorkers: 1,
  workerIdleMemoryLimit: '4096MB', // Massive limit for high-memory tests
  testTimeout: 120000, // 120s timeout per test (allow time for GC)

  // Bail early on first test failure to save memory
  // bail: 1, // DISABLED for full test run

  // Clear mocks between tests to prevent accumulation
  clearMocks: true,
  restoreMocks: true,
}

// createJestConfig is exported this way to ensure that next/jest can load the Next.js config which is async
module.exports = createJestConfig(customJestConfig)
