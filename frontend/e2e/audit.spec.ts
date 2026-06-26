import { test, expect } from '@playwright/test'

test.describe('Audit Log', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/audit')
    await page.waitForLoadState('networkidle')
  })

  test('shows audit log entries', async ({ page }) => {
    await expect(page.getByText(/login|register|create|update|delete|session/i).first()).toBeVisible({ timeout: 10000 })
  })

  test('filter inputs are visible', async ({ page }) => {
    await expect(page.getByText(/actor id/i)).toBeVisible()
    await expect(page.getByText('From', { exact: true })).toBeVisible()
    await expect(page.getByText('To', { exact: true })).toBeVisible()
  })
})
