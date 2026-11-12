import { test, expect } from '@playwright/test'

test.describe('Homepage', () => {
  test('should load successfully', async ({ page }) => {
    await page.goto('/')

    // Wait for the page to load
    await page.waitForLoadState('networkidle')

    // Check if the page has a title
    await expect(page).toHaveTitle(/maicivy/i)
  })

  test('should have navigation links', async ({ page }) => {
    await page.goto('/')

    // Check for common navigation elements
    // Update these selectors based on your actual navigation structure
    const nav = page.locator('nav')
    await expect(nav).toBeVisible()
  })
})

test.describe('CV Page', () => {
  test('should navigate to CV page', async ({ page }) => {
    await page.goto('/')

    // Navigate to CV page (update selector as needed)
    // await page.click('a[href="/cv"]')

    // Verify we're on the CV page
    // await expect(page).toHaveURL(/.*cv/)
  })

  test('should display CV content', async ({ page }) => {
    await page.goto('/cv')

    // Check for CV elements (update selectors based on your actual structure)
    // await expect(page.locator('h1')).toBeVisible()
  })
})
