import { test, expect } from '@playwright/test'

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/settings')
    await page.waitForLoadState('networkidle')
  })

  test('add setting modal has correct layout', async ({ page }) => {
    await page.getByRole('button', { name: /add setting/i }).click()
    await expect(page.getByLabel(/key/i)).toBeVisible()
    await expect(page.getByLabel(/value/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /cancel/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /save/i })).toBeVisible()
  })
})

test.describe('Feature Flags', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/feature-flags')
    await page.waitForLoadState('networkidle')
  })

  test('add flag modal opens', async ({ page }) => {
    await page.getByRole('button', { name: /add flag/i }).click()
    await expect(page.getByLabel(/name/i)).toBeVisible()
    await expect(page.getByLabel(/key/i)).toBeVisible()
  })

  test('toggle flag enable/disable', async ({ page }) => {
    const toggleBtn = page.getByRole('button', { name: /enable|disable/i }).first()
    if (await toggleBtn.isVisible()) {
      const currentText = await toggleBtn.textContent()
      await toggleBtn.click()
      await page.waitForLoadState('networkidle')
    }
  })
})
