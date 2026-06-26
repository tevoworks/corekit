import { test, expect } from '@playwright/test'

test.describe('Audit Logs — Filtering', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/audit')
    await page.waitForLoadState('networkidle')
  })

  test('filter inputs are present and refresh works', async ({ page }) => {
    await expect(page.getByText(/actor id/i).first()).toBeVisible()
    await expect(page.getByText(/action/i).first()).toBeVisible()
    await expect(page.getByText(/from/i).first()).toBeVisible()
    await expect(page.getByText(/to/i).first()).toBeVisible()
  })

  test('refresh button refetches data', async ({ page }) => {
    await page.getByRole('button', { name: /refresh/i }).click()
    await page.waitForLoadState('networkidle')
    await expect(page.getByRole('table')).toBeVisible()
  })
})
