import { test, expect } from '@playwright/test'

test.describe('Auth', () => {
  test('shows error for invalid credentials', async ({ page }) => {
    await page.goto('/login')
    await page.waitForLoadState('networkidle')
    await page.getByLabel(/email/i).fill('wrong@test.com')
    await page.getByLabel(/password/i).fill('WrongPass1!')
    await page.getByRole('button', { name: /sign in/i }).click()
    await expect(page.getByText(/invalid email or password/i).first()).toBeVisible({ timeout: 10000 })
  })

  test('login form renders correctly', async ({ page }) => {
    await page.goto('/login')
    await page.waitForLoadState('networkidle')
    await expect(page.getByLabel(/email/i)).toBeVisible()
    await expect(page.getByLabel(/password/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible()
  })
})
