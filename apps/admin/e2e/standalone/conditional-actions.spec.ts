import { test, expect } from '@playwright/test'

test.describe('Conditional Actions', () => {
  test('notifications: read single notification (if any exist)', async ({ page }) => {
    // Check if notifications exist by looking at the unread count badge
    await page.goto('/notifications')
    await page.waitForLoadState('networkidle')

    const readBtns = page.getByRole('button', { name: /read/i })
    if (await readBtns.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await readBtns.first().click()
      await page.waitForLoadState('networkidle')
      // After marking as read, the button should disappear
      await expect(page.getByRole('button', { name: /read/i }).first()).not.toBeVisible({ timeout: 5000 })
    }
  })

  test('jobs: retry button (if failed jobs exist)', async ({ page }) => {
    await page.goto('/jobs')
    await page.waitForLoadState('networkidle')

    const retryBtn = page.getByRole('button', { name: /retry/i }).first()
    if (await retryBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await retryBtn.click()
      await page.waitForLoadState('networkidle')
      // Retry should succeed without error
      const errorBanner = page.locator('.bg-\\[var\\(--danger-bg\\)\\]').first()
      await expect(errorBanner).not.toBeVisible({ timeout: 3000 })
    }
  })

  test('jobs: cancel button (if pending jobs exist)', async ({ page }) => {
    await page.goto('/jobs')
    await page.waitForLoadState('networkidle')

    const cancelBtn = page.getByRole('button', { name: /cancel/i }).first()
    if (await cancelBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await cancelBtn.click()
      await expect(page.getByText(/are you sure/i)).toBeVisible()
      await page.locator('[role="dialog"]').getByRole('button', { name: /delete/i }).click({ force: true })
      await page.waitForLoadState('networkidle')
    }
  })
})
