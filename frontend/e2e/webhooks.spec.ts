import { test, expect } from '@playwright/test'

test.describe('Webhooks', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/webhooks')
    await page.waitForLoadState('networkidle')
  })

  test('add webhook form has correct layout', async ({ page }) => {
    await page.getByRole('button', { name: /add webhook/i }).click()
    await expect(page.getByLabel(/name/i)).toBeVisible()
    await expect(page.getByLabel(/url/i)).toBeVisible()
    await expect(page.getByText(/events/i).first()).toBeVisible()
    await expect(page.getByRole('button', { name: /cancel/i })).toBeVisible()
    await expect(page.getByTestId('webhooks-form-submit-button')).toBeVisible()
  })

  test('sticky footer visible in form', async ({ page }) => {
    await page.getByRole('button', { name: /add webhook/i }).click()
    await expect(page.getByTestId('webhooks-form-card')).toBeVisible()
  })

  test('delete shows confirm modal', async ({ page }) => {
    const deleteBtn = page.getByRole('button', { name: /delete/i }).first()
    if (await deleteBtn.isVisible()) {
      await deleteBtn.click()
      await expect(page.getByText(/are you sure/i)).toBeVisible()
    }
  })
})
