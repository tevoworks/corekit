import { test, expect } from '@playwright/test'

test.describe('CRUD Operations', () => {
  test('create and revoke an API key', async ({ page }) => {
    const keyName = `e2e-key-${Date.now()}`

    await page.goto('/apikeys')
    await page.waitForLoadState('networkidle')

    await page.getByTestId('api-keys-name-input').fill(keyName)
    await page.getByRole('button', { name: /generate/i }).click()
    await expect(page.getByText(/copy this key now/i)).toBeVisible()
    await page.waitForLoadState('networkidle')

    // Revoke
    await page.getByRole('button', { name: /revoke/i }).first().click()
    await page.waitForTimeout(500)
    await page.locator('[role="dialog"] button').filter({ hasText: /revoke/i }).click()
    await page.waitForLoadState('networkidle')
  })

  test('profile update reverts on cancel', async ({ page }) => {
    await page.goto('/profile')
    await page.waitForLoadState('networkidle')

    const nameInput = page.getByLabel(/full name/i)
    const originalName = await nameInput.inputValue()

    await nameInput.fill('E2E Temporary Name')
    await page.getByRole('button', { name: /cancel/i }).click()
    await expect(nameInput).toHaveValue(originalName)
  })
})
