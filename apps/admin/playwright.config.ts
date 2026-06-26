import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 2 : 1,
  reporter: [['html', { outputFolder: 'playwright-report' }]],

  use: {
    baseURL: 'http://localhost:5173',
    viewport: { width: 1440, height: 900 },
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on-first-retry',
  },

  projects: [
    {
      name: 'standalone',
      testMatch: ['**/standalone/**', '**/registration.spec.ts'],
    },
    {
      name: 'setup',
      testMatch: '**/*.setup.ts',
    },
    {
      name: 'auth',
      testMatch: ['**/auth.spec.ts', '**/full-e2e.spec.ts'],
      dependencies: ['setup'],
    },
    {
      name: 'admin',
      testIgnore: [
        '**/auth.spec.ts',
        '**/full-e2e.spec.ts',
        '**/rbac.spec.ts',
        '**/rbac-manager.spec.ts',
        '**/standalone/register.spec.ts',
        '**/standalone/session-expiry.spec.ts',
        '**/registration.spec.ts',
      ],
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
        storageState: 'e2e/.auth/admin.json',
      },
      dependencies: ['setup'],
    },
    {
      name: 'viewer',
      testMatch: '**/rbac.spec.ts',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
        storageState: 'e2e/.auth/viewer.json',
      },
      dependencies: ['setup'],
    },
    {
      name: 'manager',
      testMatch: '**/rbac-manager.spec.ts',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
        storageState: 'e2e/.auth/manager.json',
      },
      dependencies: ['setup'],
    },
  ],
})
