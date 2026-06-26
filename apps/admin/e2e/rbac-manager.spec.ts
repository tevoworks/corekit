import { test, expect } from '@playwright/test'

test.describe('RBAC — Manager Permissions', () => {
  test('manager can add users', async ({ page }) => {
    await page.goto('/users')
    await page.waitForLoadState('networkidle')
    await expect(page.getByRole('button', { name: /add user/i })).toBeVisible()
  })

  test('manager can view roles page', async ({ page }) => {
    await page.goto('/roles')
    await page.waitForLoadState('networkidle')
    await expect(page.getByText('Roles & Permissions')).toBeVisible()
  })

  test('manager can view jobs page', async ({ page }) => {
    await page.goto('/jobs')
    await page.waitForLoadState('networkidle')
    await expect(page.getByRole('heading', { name: /background jobs/i })).toBeVisible()
  })
})
