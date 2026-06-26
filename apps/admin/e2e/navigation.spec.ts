import { test, expect } from '@playwright/test'

test.describe('Navigation', () => {
  test('sidebar shows expected links', async ({ page }) => {
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    await expect(page.getByRole('link', { name: /dashboard/i })).toBeVisible()
    await expect(page.getByRole('link', { name: /users/i })).toBeVisible()
    await expect(page.getByRole('link', { name: /settings/i })).toBeVisible()
  })

  test('logout button is visible', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('button', { name: /logout/i })).toBeVisible()
  })

  test('admin user email appears in sidebar', async ({ page }) => {
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    await expect(page.getByText(/admin@corekit\.com/i)).toBeVisible()
  })
})
