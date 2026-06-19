import { defineConfig, devices } from '@playwright/test';

// Config Playwright pour tester la PROD (pas de webServer local — le `next dev` turbopack est cassé).
// Usage : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com npx playwright test --config=playwright.prod.config.ts
export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  reporter: 'list',
  use: {
    baseURL: process.env.PLAYWRIGHT_TEST_BASE_URL || 'https://maicivy.etheryale.com',
    screenshot: 'only-on-failure',
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  // PAS de webServer : on tape la prod directement.
});
