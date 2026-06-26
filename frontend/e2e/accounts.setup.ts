import { test as setup, expect } from '@playwright/test'
import path from 'path'

interface Account {
  email: string
  password: string
  file: string
}

function getAccounts(): Account[] {
  const raw = process.env.E2E_ACCOUNTS
  if (raw) {
    try {
      return JSON.parse(raw)
    } catch {
      console.warn('Invalid E2E_ACCOUNTS JSON, falling back to defaults')
    }
  }

  return [
    { email: process.env.E2E_ADMIN_EMAIL || 'admin@corekit.com', password: process.env.E2E_ADMIN_PASS || 'Admin123!', file: 'admin.json' },
    { email: process.env.E2E_VIEWER_EMAIL || 'viewer@test.corekit', password: process.env.E2E_VIEWER_PASS || 'ViewerPass1!', file: 'viewer.json' },
    { email: process.env.E2E_MANAGER_EMAIL || 'manager@test.corekit', password: process.env.E2E_MANAGER_PASS || 'ManagerPass1!', file: 'manager.json' },
  ]
}

const ACCOUNTS = getAccounts()

for (const acc of ACCOUNTS) {
  setup(`authenticate as ${acc.email}`, async ({ page, context }) => {
    const authFile = path.resolve(`e2e/.auth/${acc.file}`)

    await page.goto('/login')
    await page.waitForLoadState('networkidle')

    for (let attempt = 0; attempt < 20; attempt++) {
      await page.getByLabel(/email/i).fill(acc.email)
      await page.getByLabel(/password/i).fill(acc.password)
      await page.getByRole('button', { name: /sign in/i }).click()

      await page.waitForTimeout(1500)

      if (page.url().includes('/login') === false && page.url().includes('/register') === false) {
        break
      }

      await page.waitForTimeout(3000)
    }

    await page.context().storageState({ path: authFile })
  })
}
