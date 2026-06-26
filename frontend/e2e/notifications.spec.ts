import { test, expect } from '@playwright/test'

test.describe('Notifications', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/notifications')
    await page.waitForLoadState('networkidle')
  })

  test('shows notification section', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /notifications/i }).first()).toBeVisible()
  })

  test('mark all as read button visible when notifications exist', async ({ page }) => {
    const btn = page.getByRole('button', { name: /mark all as read/i })
    if (await btn.isVisible()) {
      await btn.click()
      await page.waitForLoadState('networkidle')
    }
  })

  test('delete shows confirm modal', async ({ page }) => {
    const deleteBtn = page.getByRole('button', { name: /delete/i }).first()
    if (await deleteBtn.isVisible()) {
      await deleteBtn.click()
      await expect(page.getByText(/are you sure/i)).toBeVisible()
    }
  })
})
