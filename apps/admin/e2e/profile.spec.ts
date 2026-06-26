import { test, expect } from '@playwright/test'

test.describe('Profile', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/profile')
    await page.waitForLoadState('networkidle')
  })

  test('shows account details form', async ({ page }) => {
    await expect(page.getByLabel(/full name/i)).toBeVisible()
    await expect(page.getByLabel(/email/i)).toBeVisible()
  })

  test('save button is disabled when no changes', async ({ page }) => {
    await expect(page.getByRole('button', { name: /save changes/i })).toBeDisabled()
  })

  test('save button enables on changes', async ({ page }) => {
    await page.getByLabel(/full name/i).fill('Changed Name')
    await expect(page.getByRole('button', { name: /save changes/i })).toBeEnabled()
  })

  test('cancel resets form fields', async ({ page }) => {
    const nameInput = page.getByLabel(/full name/i)
    const originalValue = await nameInput.inputValue()
    await nameInput.fill('Temporary')
    await page.getByRole('button', { name: /cancel/i }).click()
    await expect(nameInput).toHaveValue(originalValue)
  })

  test('sticky footer is visible', async ({ page }) => {
    // Scroll to ensure footer is in viewport
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await page.waitForTimeout(300)
    await expect(page.locator('.sticky.bottom-0').first()).toBeVisible()
  })

  test('account info section shows user data', async ({ page }) => {
    await expect(page.getByText(/super admin/i)).toBeVisible()
    await expect(page.getByText(/status/i).first()).toBeVisible()
  })
})
