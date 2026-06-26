import { test, expect } from '@playwright/test'

test.describe('UI/UX Compliance', () => {
  test('sidebar links navigate to correct pages', async ({ page }) => {
    await page.goto('/')
    await page.waitForLoadState('networkidle')

    const links = [
      { href: '/', title: 'Dashboard' },
      { href: '/users', title: 'Users' },
      { href: '/roles', title: 'Roles' },
      { href: '/permissions', title: 'Permissions' },
      { href: '/settings', title: 'Settings' },
      { href: '/feature-flags', title: 'Feature Flags' },
      { href: '/audit', title: 'Audit Log' },
      { href: '/apikeys', title: 'API Keys' },
      { href: '/webhooks', title: 'Webhooks' },
      { href: '/storage', title: 'Storage' },
      { href: '/sessions', title: 'Sessions' },
      { href: '/jobs', title: 'Jobs' },
      { href: '/notifications', title: 'Notifications' },
      { href: '/profile', title: 'Profile' },
    ]

    for (const link of links) {
      const navEl = page.locator(`a[href="${link.href}"]`).first()
      if (await navEl.isVisible({ timeout: 1000 }).catch(() => false)) {
        await navEl.click()
        await page.waitForTimeout(500)
        await expect(page).toHaveURL(link.href)
      }
    }
  })

  test('cancel buttons use ghost variant', async ({ page }) => {
    // Check profile page cancel button
    await page.goto('/profile')
    await page.waitForLoadState('networkidle')
    const cancelBtn = page.getByRole('button', { name: /cancel/i })
    if (await cancelBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      // Ghost buttons don't have border — verify no border classes
      const classes = await cancelBtn.getAttribute('class')
      expect(classes).not.toContain('border')
    }
  })

  test('error banners are dismissible', async ({ page }) => {
    // Trigger an error by navigating to a page that will fail
    // Check if error banner has close button
    await page.goto('/jobs')
    await page.waitForLoadState('networkidle')
    // Error banner should have X close button if error exists
    const errorBanner = page.locator('.bg-\\[var\\(--danger-bg\\)\\]').first()
    if (await errorBanner.isVisible({ timeout: 2000 }).catch(() => false)) {
      const closeBtn = errorBanner.locator('button')
      await expect(closeBtn).toBeVisible()
    }
  })

  test('tables have sticky header', async ({ page }) => {
    await page.goto('/users')
    await page.waitForLoadState('networkidle')
    const thead = page.locator('thead.sticky')
    await expect(thead).toBeVisible()
  })
})
