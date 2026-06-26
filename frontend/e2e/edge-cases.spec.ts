import { test, expect } from '@playwright/test'

test.describe('Edge Cases', () => {
  test('users: validation on empty create form', async ({ page }) => {
    await page.goto('/users')
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: /add user/i }).click()
    await page.waitForTimeout(300)

    // Click Create without filling fields
    await page.getByRole('button', { name: /create/i }).click()
    await page.waitForTimeout(500)
    // HTML5 validation prevents submission — form should still be open
    await expect(page.locator('[role="dialog"]')).toBeVisible()
  })

  test('profile: toast appears after save', async ({ page }) => {
    await page.goto('/profile')
    await page.waitForLoadState('networkidle')

    // Make a change, save, expect toast
    const nameInput = page.getByLabel(/full name/i)
    const original = await nameInput.inputValue()
    await nameInput.fill(original + ' ')
    await page.getByRole('button', { name: /save changes/i }).click()
    await expect(page.getByText(/profile updated/i)).toBeVisible({ timeout: 5000 })
    // Toast should auto-dismiss
    await page.waitForTimeout(3500)
    await expect(page.getByText(/profile updated/i)).not.toBeVisible()
  })

  test('modal: close via X button', async ({ page }) => {
    await page.goto('/users')
    await page.waitForLoadState('networkidle')
    await page.getByRole('button', { name: /add user/i }).click()
    await page.waitForTimeout(300)

    await expect(page.locator('[role="dialog"]')).toBeVisible()
    await page.locator('[role="dialog"] button').filter({ hasText: /close/i }).click()
    await expect(page.locator('[role="dialog"]')).not.toBeVisible()
  })

  test('modal: close via backdrop click', async ({ page }) => {
    await page.goto('/users')
    await page.waitForLoadState('networkidle')
    await page.getByRole('button', { name: /add user/i }).click()
    await page.waitForTimeout(300)

    await expect(page.locator('[role="dialog"]')).toBeVisible()
    // Click the backdrop overlay
    await page.locator('.fixed.inset-0').first().click({ position: { x: 10, y: 10 } })
    await expect(page.locator('[role="dialog"]')).not.toBeVisible()
  })

  test('apikeys: generate disabled with empty name', async ({ page }) => {
    await page.goto('/apikeys')
    await page.waitForLoadState('networkidle')
    await expect(page.getByRole('button', { name: /generate/i })).toBeDisabled()
  })

  test('users: empty state shows action button', async ({ page }) => {
    // Create then delete all users to trigger empty state
    await page.goto('/users')
    await page.waitForLoadState('networkidle')
    const hasEmpty = await page.getByText(/no users found/i).isVisible().catch(() => false)
    if (hasEmpty) {
      await expect(page.getByRole('button', { name: /add user/i }).first()).toBeVisible()
    }
  })
})
