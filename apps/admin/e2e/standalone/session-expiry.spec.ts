import { test, expect } from '@playwright/test'

test.describe('Session Expiry', () => {
  test('revoked session returns 401 and refresh fails', async ({ page }) => {
    // Login via API (retry once if rate limited)
    let loginResp = await page.request.post('/api/auth/login', {
      data: { email: 'admin@corekit.com', password: 'Admin123!' },
      headers: { 'Origin': 'http://localhost:5173' },
    })
    if (loginResp.status() === 429) {
      await page.waitForTimeout(2000)
      loginResp = await page.request.post('/api/auth/login', {
        data: { email: 'admin@corekit.com', password: 'Admin123!' },
        headers: { 'Origin': 'http://localhost:5173' },
      })
    }
    expect(loginResp.status()).toBe(200)
    const body = await loginResp.json()
    const token = body?.data?.token
    expect(token).toBeDefined()

    // 1. Token works for authenticated requests
    const meResp = await page.request.get('/api/me', {
      headers: { 'Authorization': `Bearer ${token}`, 'Origin': 'http://localhost:5173' },
    })
    expect(meResp.status()).toBe(200)

    // Parse token to get token_id
    const payload = JSON.parse(Buffer.from(token.split('.')[1], 'base64url').toString())
    const tokenId = payload.token_id

    // 2. Revoke the session
    const revokeResp = await page.request.delete(`/api/sessions/all/${tokenId}`, {
      headers: { 'Authorization': `Bearer ${token}`, 'Origin': 'http://localhost:5173' },
    })
    expect(revokeResp.status()).toBe(200)

    // 3. Same token now returns 401
    const failedResp = await page.request.get('/api/me', {
      headers: { 'Authorization': `Bearer ${token}`, 'Origin': 'http://localhost:5173' },
    })
    expect(failedResp.status()).toBe(401)

    // 4. Refresh also fails with revoked token
    const refreshResp = await page.request.post('/api/auth/refresh', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Cookie': `token=${token}`,
        'Origin': 'http://localhost:5173',
      },
    })
    expect(refreshResp.status()).toBe(401)
  })
})
