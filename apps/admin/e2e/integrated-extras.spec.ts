import { test, expect } from '@playwright/test'

test.describe('Integrated Extras', () => {
  test('permissions: tab switching', async ({ page }) => {
    await page.goto('/permissions')
    await page.waitForLoadState('networkidle')

    // Click Templates tab
    await page.getByRole('button', { name: /templates/i }).click()
    await page.waitForTimeout(500)
    // Templates are seeded — table should have data
    await expect(page.getByRole('table')).toBeVisible()

    // Click Registry tab
    await page.getByRole('button', { name: /registry/i }).click()
    await page.waitForTimeout(500)
    await expect(page.getByRole('table').or(page.getByText(/permissions registered/i))).toBeVisible()
  })

  test('permissions: sync from YAML', async ({ page }) => {
    await page.goto('/permissions')
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: /sync/i }).click()
    await page.waitForLoadState('networkidle')
    // Sync should succeed — page stays on registry tab
    await expect(page.getByText(/no permissions registered/i).or(page.getByRole('table'))).toBeVisible({ timeout: 10000 })
  })

  test('permissions: delete registry entry', async ({ page }) => {
    await page.goto('/permissions')
    await page.waitForLoadState('networkidle')

    const deleteBtn = page.getByRole('button', { name: /delete/i }).first()
    if (!(await deleteBtn.isVisible({ timeout: 2000 }).catch(() => false))) return

    await deleteBtn.click()
    await expect(page.locator('[role="dialog"]')).toBeVisible()
    await expect(page.getByText(/are you sure/i)).toBeVisible()
    await page.locator('[role="dialog"]').getByRole('button', { name: /cancel/i }).click()
    await expect(page.locator('[role="dialog"]')).not.toBeVisible()
  })

  test('roles: permission editor save flow', async ({ page }) => {
    await page.goto('/roles')
    await page.waitForLoadState('networkidle')

    // Open permission editor for first role
    await page.getByRole('button', { name: /lock permissions/i }).first().click()
    await page.waitForTimeout(500)
    await expect(page.getByTestId('roles-permissions-modal')).toBeVisible()

    // Toggle first unchecked permission
    const unchecked = page.locator('[role="dialog"] input[type="checkbox"]:not(:checked)').first()
    if (await unchecked.isVisible({ timeout: 3000 }).catch(() => false)) {
      await unchecked.check()
    }

    // Click Save
    await page.getByRole('button', { name: /save/i }).click()
    await page.waitForLoadState('networkidle')
    // Modal should close on success
    await expect(page.locator('[role="dialog"]')).not.toBeVisible()
  })

  test('webhooks: deliveries panel open and close', async ({ page }) => {
    await page.goto('/webhooks')
    await page.waitForLoadState('networkidle')

    const deliveriesBtn = page.getByRole('button', { name: /deliveries/i }).first()
    if (!(await deliveriesBtn.isVisible({ timeout: 2000 }).catch(() => false))) return

    await deliveriesBtn.click()
    await expect(page.getByText(/deliveries:/i)).toBeVisible()

    // Close panel
    await page.getByRole('button', { name: /close/i }).click()
    await expect(page.getByText(/deliveries:/i)).not.toBeVisible()
  })

  test('feature flags: create and verify in table', async ({ page }) => {
    const ts = Date.now()
    const key = `e2e_flag_${ts}`

    await page.goto('/feature-flags')
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: /add flag/i }).click()
    await page.waitForTimeout(300)
    await page.getByLabel(/name/i).fill(`Flag ${ts}`)
    await page.getByLabel(/key/i).fill(key)
    await page.getByLabel(/description/i).fill(`Description ${ts}`)
    await page.getByRole('button', { name: /create/i }).click()
    await page.waitForLoadState('networkidle')

    await expect(page.getByText(key).first()).toBeVisible({ timeout: 30000 })
  })

  test('feature flags: toggle status changes', async ({ page }) => {
    // First ensure there's a flag to toggle (create one with unique key)
    const ts = Date.now()
    const key = `e2e_toggle_${ts}`

    await page.goto('/feature-flags')
    await page.waitForLoadState('networkidle')

    // Create
    await page.getByRole('button', { name: /add flag/i }).click()
    await page.waitForTimeout(300)
    await page.getByLabel(/name/i).fill(`Toggle ${ts}`)
    await page.getByLabel(/key/i).fill(key)
    await page.getByLabel(/description/i).fill(`Description ${ts}`)
    await page.getByRole('button', { name: /create/i }).click()
    await page.waitForLoadState('networkidle')
    await expect(page.getByText(key).first()).toBeVisible()

    // Toggle disable
    const disableBtn = page.getByRole('button', { name: /disable/i }).first()
    if (await disableBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await disableBtn.click()
      await page.waitForLoadState('networkidle')
      // Now should show "Enable"
      await expect(page.getByRole('button', { name: /enable/i }).first()).toBeVisible({ timeout: 5000 })
    }
  })
})
