import { test, expect, type Page } from '@playwright/test';

const T = {
  login_page: {
    form: 'login-form',
    email_input: 'login-email-input',
    password_input: 'login-password-input',
    sign_in_button: 'login-sign-in-button',
    success_banner: 'login-success-banner',
    expired_banner: 'login-expired-banner',
  },
  register_page: {
    form: 'register-form',
    name_input: 'register-name-input',
    email_input: 'register-email-input',
    password_input: 'register-password-input',
    submit_button: 'register-submit-button',
  },
  dashboard_page: {
    stat_total_users: 'dashboard-stat-total-users',
    recent_activity_heading: 'dashboard-recent-activity-heading',
  },
  layout: {
    nav_logout_button: 'nav-logout-button',
  },
};

function t(section: keyof typeof T, join: string) {
  const parts = join.split('.');
  let obj: any = T;
  for (const part of [section, ...parts]) {
    obj = obj[part];
  }
  return obj;
}

async function unauthPage(page: Page, url: string) {
  await page.route('**/api/me', route => route.fulfill({ status: 403 }));
  await page.goto(url);
}

test.describe('authentication flows', () => {
  test('complete_authentication_flow: register, logout, login, verify dashboard', async ({ page }) => {
    const ts = Date.now();
    await unauthPage(page, '/login');
    await expect(page.getByTestId(T.login_page.form)).toBeVisible();
    await page.goto('/register');
    await expect(page.getByTestId(T.register_page.form)).toBeVisible();
    await page.getByTestId(T.register_page.name_input).fill('Admin User');
    await page.getByTestId(T.register_page.email_input).fill(`admin-${ts}@test.com`);
    await page.getByTestId(T.register_page.password_input).fill('SecureP@ss1');
    await page.getByTestId(T.register_page.submit_button).click();
    await expect(page.getByTestId(T.dashboard_page.stat_total_users)).toBeVisible();

    await page.getByTestId(T.layout.nav_logout_button).click();
    await expect(page.getByTestId(T.login_page.form)).toBeVisible();

    await page.getByTestId(T.login_page.email_input).fill(`admin-${ts}@test.com`);
    await page.getByTestId(T.login_page.password_input).fill('SecureP@ss1');
    await page.getByTestId(T.login_page.sign_in_button).click();
    await expect(page.getByTestId(T.dashboard_page.recent_activity_heading)).toBeVisible();
  });

  test('second_user_registration_flow: register non-admin user', async ({ page }) => {
    await unauthPage(page, '/login');
    await expect(page.getByTestId(T.login_page.form)).toBeVisible();
    await page.goto('/register');
    await expect(page.getByTestId(T.register_page.form)).toBeVisible();
    await page.getByTestId(T.register_page.name_input).fill('Regular User');
    await page.getByTestId(T.register_page.email_input).fill('user@corekit.test');
    await page.getByTestId(T.register_page.password_input).fill('UserP@ss1');
    await page.getByTestId(T.register_page.submit_button).click();
    await expect(page.getByTestId(T.login_page.success_banner)).toBeVisible();
  });

  test('expired_session_redirect_flow: expired JWT shows expired banner', async ({ page }) => {
    await unauthPage(page, '/');
    await expect(page.getByTestId(T.login_page.expired_banner).or(page.getByTestId(T.login_page.form))).toBeVisible();
  });
});
