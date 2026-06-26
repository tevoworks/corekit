import { test, expect } from '@playwright/test'

test.describe('RBAC — Viewer Permissions', () => {
  test('viewer can view users page', async ({ page }) => {
    await page.goto('/users')
    await page.waitForLoadState('networkidle')
    // Viewer has read:users permission — heading should be visible
    await expect(page.getByRole('heading', { name: /users/i })).toBeVisible()
  })

  test('viewer can view roles page', async ({ page }) => {
    await page.goto('/roles')
    await page.waitForLoadState('networkidle')
    await expect(page.getByRole('heading', { name: /roles/i })).toBeVisible()
  })

  test('viewer can view audit logs page', async ({ page }) => {
    await page.goto('/audit')
    await page.waitForLoadState('networkidle')
    await expect(page.getByText(/actor id/i)).toBeVisible()
  })

  test('viewer can view jobs page without error', async ({ page }) => {
    await page.goto('/jobs')
    await page.waitForLoadState('networkidle')
    // Jobs page loads (API 403 is handled gracefully by TanStack Query)
    await expect(page.getByRole('heading', { name: /background jobs/i })).toBeVisible()
  })

  test('viewer sees access denied on sessions page', async ({ page }) => {
    await page.goto('/sessions')
    await page.waitForLoadState('networkidle')
    await expect(page.getByText(/access denied/i)).toBeVisible()
  })
})
