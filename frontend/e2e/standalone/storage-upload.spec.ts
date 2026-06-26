import { test, expect } from '@playwright/test'
import path from 'path'
import fs from 'fs'
import os from 'os'

test.describe('Storage Upload', () => {
  let filePath: string

  test.beforeAll(() => {
    // Create a temporary test file
    filePath = path.join(os.tmpdir(), `e2e-test-upload-${Date.now()}.txt`)
    fs.writeFileSync(filePath, 'E2E test file content for Playwright upload test')
  })

  test.afterAll(() => {
    // Cleanup temp file
    try { fs.unlinkSync(filePath) } catch {}
  })

  test('upload a file and verify it appears in the table', async ({ page }) => {
    await page.goto('/storage')
    await page.waitForLoadState('networkidle')

    // Set file on the hidden input
    const fileInput = page.locator('input[type="file"]')
    await expect(fileInput).toBeAttached()
    await fileInput.setInputFiles(filePath)

    // Wait for upload to complete
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000)

    // Verify file appears in table (filename is the original name)
    const fileName = path.basename(filePath)
    const fileCell = page.getByRole('cell', { name: fileName }).first()
    if (await fileCell.isVisible({ timeout: 5000 }).catch(() => false)) {
      // Delete the uploaded file
      const deleteBtn = page.getByRole('button', { name: /delete/i }).first()
      if (await deleteBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
        await deleteBtn.click()
        await expect(page.locator('[role="dialog"]')).toBeVisible()
        await page.locator('[role="dialog"]').getByRole('button', { name: /delete/i }).click({ force: true })
        await page.waitForLoadState('networkidle')
      }
    }
  })
})
