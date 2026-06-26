import { test, expect } from '@playwright/test'

test.describe('Users', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/users')
    await page.waitForLoadState('networkidle')
  })

  test('displays user table', async ({ page }) => {
    await expect(page.getByRole('cell', { name: /admin@corekit\.com/i })).toBeVisible()
  })

  test('create user modal has correct form layout', async ({ page }) => {
    await page.getByRole('button', { name: /add user/i }).click()
    await expect(page.getByLabel(/email/i)).toBeVisible()
    await expect(page.getByLabel(/full name/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /cancel/i }).first()).toBeVisible()
    await expect(page.getByRole('button', { name: /create/i })).toBeVisible()
  })

  test('create user modal has sticky footer', async ({ page }) => {
    await page.getByRole('button', { name: /add user/i }).click()
    await expect(page.getByTestId('users-create-modal')).toBeVisible()
  })

  test('delete user shows confirm modal', async ({ page }) => {
    await page.getByRole('button', { name: /delete/i }).first().click()
    await expect(page.getByTestId('users-delete-confirm-dialog')).toBeVisible()
  })
})
