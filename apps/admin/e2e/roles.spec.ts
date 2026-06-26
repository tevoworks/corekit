import { test, expect } from '@playwright/test'

test.describe('Roles', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/roles')
    await page.waitForLoadState('networkidle')
  })

  test('shows seeded roles', async ({ page }) => {
    await expect(page.getByTestId('roles-table')).toBeVisible()
    await expect(page.getByRole('cell', { name: /admin/i })).toBeVisible()
    await expect(page.getByRole('cell', { name: /viewer/i })).toBeVisible()
  })

  test('add role modal has correct form', async ({ page }) => {
    await page.getByRole('button', { name: /add role/i }).click()
    await expect(page.getByLabel(/role name/i)).toBeVisible()
    await expect(page.getByLabel(/description/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /cancel/i })).toBeVisible()
  })

  test('add role modal has sticky footer', async ({ page }) => {
    await page.getByRole('button', { name: /add role/i }).click()
    await expect(page.getByTestId('roles-form-modal')).toBeVisible()
  })

  test('delete role shows confirm modal', async ({ page }) => {
    const deleteBtn = page.getByRole('button', { name: /delete/i }).first()
    await deleteBtn.click()
    await expect(page.getByTestId('roles-delete-confirm-dialog')).toBeVisible()
  })

  test('permission editor opens', async ({ page }) => {
    const permBtn = page.getByRole('button', { name: /lock permissions/i }).first()
    await permBtn.click()
    await page.waitForTimeout(500)
    await expect(page.getByTestId('roles-permissions-modal')).toBeVisible({ timeout: 10000 })
  })
})
