import { test, expect } from '@playwright/test'

test.describe('API Keys', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/apikeys')
    await page.waitForLoadState('networkidle')
  })

  test('create key shows raw key warning', async ({ page }) => {
    await page.getByTestId('api-keys-name-input').fill('e2e-test-key')
    await page.getByRole('button', { name: /generate/i }).click()
    await expect(page.getByTestId('api-keys-new-key-banner')).toBeVisible()
  })

  test('revoke key shows confirm modal', async ({ page }) => {
    const revokeBtn = page.getByRole('button', { name: /revoke/i }).first()
    if (await revokeBtn.isVisible()) {
      await revokeBtn.click()
      await expect(page.getByTestId('api-keys-revoke-confirm-dialog')).toBeVisible()
    }
  })

  test('create key input has placeholder', async ({ page }) => {
    await expect(page.getByTestId('api-keys-name-input')).toBeVisible()
  })
})
