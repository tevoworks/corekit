import { test, expect } from '@playwright/test'

test.describe('Sessions', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/sessions')
    await page.waitForLoadState('networkidle')
  })

  test('shows session page', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /all sessions/i })).toBeVisible()
  })

  test('revoke session shows confirm modal', async ({ page }) => {
    const revokeBtn = page.getByRole('button', { name: /revoke/i }).first()
    if (await revokeBtn.isVisible()) {
      await revokeBtn.click()
      await expect(page.getByText(/are you sure/i)).toBeVisible()
    }
  })
})
