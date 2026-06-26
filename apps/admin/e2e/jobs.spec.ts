import { test, expect } from '@playwright/test'

test.describe('Jobs', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/jobs')
    await page.waitForLoadState('networkidle')
  })

  test('shows job page sections', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /background jobs/i })).toBeVisible()
  })
})
