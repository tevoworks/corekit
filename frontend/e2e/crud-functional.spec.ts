import { test, expect } from '@playwright/test'

test.describe('CRUD Functional', () => {
  test('users: create user modal works', async ({ page }) => {
    const ts = Date.now()
    const email = `e2e-crud-${ts}@test.corekit`

    await page.goto('/users')
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: /add user/i }).click()
    await page.waitForTimeout(300)
    await page.getByLabel(/email/i).fill(email)
    await page.getByLabel(/full name/i).fill(`E2E User ${ts}`)
    await page.getByRole('button', { name: /create/i }).click()

    // Verify modal closes (API returns 201)
    await expect(page.locator('[role="dialog"]')).not.toBeVisible({ timeout: 15000 })
    // No error banner
    await expect(page.locator('.bg-\\[var\\(--danger-bg\\)\\]').first()).not.toBeVisible({ timeout: 3000 })
  })

  test('roles: create a new role via modal', async ({ page }) => {
    const ts = Date.now()
    const roleName = `e2e-role-${ts}`

    await page.goto('/roles')
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: /add role/i }).click()
    await page.waitForTimeout(300)
    await page.getByLabel(/role name/i).fill(roleName)
    await page.getByRole('button', { name: /create/i }).click()
    await expect(page.getByRole('cell', { name: roleName })).toBeVisible({ timeout: 30000 })
  })

  test('settings: create a setting via modal', async ({ page }) => {
    const ts = Date.now()
    const key = `e2e_setting_${ts}`

    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: /add setting/i }).click()
    await page.waitForTimeout(300)
    await page.getByLabel(/key/i).fill(key)
    await page.getByLabel(/value/i).fill(`value_${ts}`)
    await page.getByRole('button', { name: /save/i }).click()
    await expect(page.getByRole('cell', { name: key })).toBeVisible({ timeout: 30000 })
  })

  test('feature flags: create flag modal works', async ({ page }) => {
    await page.goto('/feature-flags')
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: /add flag/i }).click()
    await page.waitForTimeout(300)
    await expect(page.getByLabel(/name/i)).toBeVisible()
    await expect(page.getByLabel(/key/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /create/i })).toBeVisible()
  })
})
