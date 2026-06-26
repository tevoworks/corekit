import { test, expect } from '@playwright/test'

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForLoadState('networkidle')
  })

  test('shows stat cards with values', async ({ page }) => {
    await expect(page.getByText(/total users/i)).toBeVisible()
    await expect(page.getByText(/your role/i)).toBeVisible()
    await expect(page.getByText(/welcome back/i)).toBeVisible()
  })

  test('shows recent activity section', async ({ page }) => {
    await expect(page.getByText(/recent activity/i)).toBeVisible()
  })
})
