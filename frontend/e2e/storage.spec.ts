import { test, expect } from '@playwright/test'

test.describe('Storage', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/storage')
    await page.waitForLoadState('networkidle')
  })

  test('upload button is visible', async ({ page }) => {
    await expect(page.getByRole('button', { name: /upload file/i }).first()).toBeVisible()
  })

  test('delete file shows confirm modal', async ({ page }) => {
    const deleteBtn = page.getByRole('button', { name: /delete/i }).first()
    if (await deleteBtn.isVisible()) {
      await deleteBtn.click()
      await expect(page.getByText(/are you sure/i)).toBeVisible()
    }
  })
})
