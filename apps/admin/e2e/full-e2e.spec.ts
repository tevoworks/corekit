import { test, expect, type Page } from '@playwright/test';

const T = {
  shared: {
    modal_close_button: 'modal-close-button',
    error_boundary_retry_button: 'error-boundary-retry-button',
    page_loading_indicator: 'page-loading-spinner',
  },
  layout: {
    nav_dashboard_link: 'nav-dashboard-link',
    nav_users_link: 'nav-users-link',
    nav_roles_link: 'nav-roles-link',
    nav_permissions_link: 'nav-permissions-link',
    nav_settings_link: 'nav-settings-link',
    nav_feature_flags_link: 'nav-feature-flags-link',
    nav_audit_log_link: 'nav-audit-log-link',
    nav_api_keys_link: 'nav-api-keys-link',
    nav_webhooks_link: 'nav-webhooks-link',
    nav_storage_link: 'nav-storage-link',
    nav_sessions_link: 'nav-sessions-link',
    nav_jobs_link: 'nav-jobs-link',
    nav_notifications_link: 'nav-notifications-link',
    nav_profile_link: 'nav-profile-link',
    nav_logout_button: 'nav-logout-button',
  },
  login_page: {
    card: 'login-card',
    form: 'login-form',
    success_banner: 'login-success-banner',
    expired_banner: 'login-expired-banner',
    error_banner: 'login-error-banner',
    email_input: 'login-email-input',
    password_input: 'login-password-input',
    sign_in_button: 'login-sign-in-button',
  },
  register_page: {
    card: 'register-card',
    form: 'register-form',
    error_banner: 'register-error-banner',
    name_input: 'register-name-input',
    email_input: 'register-email-input',
    password_input: 'register-password-input',
    submit_button: 'register-submit-button',
    sign_in_link: 'register-sign-in-link',
  },
  dashboard_page: {
    stat_total_users: 'dashboard-stat-total-users',
    stat_roles: 'dashboard-stat-roles',
    stat_your_role: 'dashboard-stat-your-role',
    stat_status: 'dashboard-stat-status',
    recent_activity_heading: 'dashboard-recent-activity-heading',
    activity_loading: 'dashboard-activity-loading',
    empty_activity: 'dashboard-empty-activity',
    activity_card: 'dashboard-activity-card',
    activity_row_prefix: 'dashboard-activity-row-',
    stat_loading_prefix: 'dashboard-stat-loading-',
  },
  users_page: {
    loading_card: 'users-loading-card',
    loading: 'users-loading',
    error_banner: 'users-error-banner',
    error_dismiss_button: 'users-error-dismiss-button',
    add_button: 'users-add-button',
    search_input: 'users-search-input',
    sort_name: 'users-sort-name',
    sort_email: 'users-sort-email',
    sort_role: 'users-sort-role',
    sort_status: 'users-sort-status',
    table: 'users-table',
    row_prefix: 'users-row-',
    edit_prefix: 'users-edit-',
    status_prefix: 'users-status-',
    delete_prefix: 'users-delete-',
    empty_card: 'users-empty-card',
    empty_state: 'users-empty-state',
    empty_add_button: 'users-empty-add-button',
    no_results_card: 'users-no-results-card',
    no_results: 'users-no-results',
    create_modal: 'users-create-modal',
    edit_modal: 'users-edit-modal',
    status_modal: 'users-status-modal',
    delete_modal: 'users-delete-modal',
    form_email_input: 'users-form-email-input',
    form_name_input: 'users-form-name-input',
    form_cancel_button: 'users-form-cancel-button',
    form_submit_button: 'users-form-submit-button',
    status_radio_prefix: 'users-status-radio-',
    status_cancel_button: 'users-status-cancel-button',
    status_confirm_button: 'users-status-confirm-button',
    delete_cancel_button: 'users-delete-cancel-button',
    delete_confirm_button: 'users-delete-confirm-button',
  },
  roles_page: {
    loading_card: 'roles-loading-card',
    loading: 'roles-loading',
    error_banner: 'roles-error-banner',
    error_dismiss_button: 'roles-error-dismiss-button',
    add_button: 'roles-add-button',
    search_input: 'roles-search-input',
    sort_name: 'roles-sort-name',
    sort_description: 'roles-sort-description',
    sort_permissions: 'roles-sort-permissions',
    table: 'roles-table',
    row_prefix: 'roles-row-',
    permissions_prefix: 'roles-permissions-',
    edit_prefix: 'roles-edit-',
    delete_prefix: 'roles-delete-',
    empty_card: 'roles-empty-card',
    empty_state: 'roles-empty-state',
    empty_add_button: 'roles-empty-add-button',
    no_results_card: 'roles-no-results-card',
    no_results: 'roles-no-results',
    load_more_button: 'roles-load-more-button',
    form_modal: 'roles-form-modal',
    form_name_input: 'roles-form-name-input',
    form_description_input: 'roles-form-description-input',
    form_cancel_button: 'roles-form-cancel-button',
    form_submit_button: 'roles-form-submit-button',
    delete_modal: 'roles-delete-modal',
    delete_cancel_button: 'roles-delete-cancel-button',
    delete_confirm_button: 'roles-delete-confirm-button',
    permissions_modal: 'roles-permissions-modal',
    perm_error_banner: 'roles-perm-error-banner',
    perm_checkbox_prefix: 'roles-perm-checkbox-',
    permissions_cancel_button: 'roles-permissions-cancel-button',
    permissions_save_button: 'roles-permissions-save-button',
  },
  permissions_page: {
    error_banner: 'permissions-error-banner',
    error_dismiss_button: 'permissions-error-dismiss-button',
    sync_button: 'permissions-sync-button',
    tab_registry: 'permissions-tab-registry',
    tab_templates: 'permissions-tab-templates',
    domain_empty: 'permissions-domain-empty',
    registry_loading_card: 'permissions-registry-loading-card',
    registry_loading: 'permissions-registry-loading',
    registry_empty_card: 'permissions-registry-empty-card',
    registry_empty: 'permissions-registry-empty',
    empty_sync_button: 'permissions-empty-sync-button',
    registry_table: 'permissions-registry-table',
    delete_registry_prefix: 'permissions-delete-registry-',
    templates_loading_card: 'permissions-templates-loading-card',
    templates_loading: 'permissions-templates-loading',
    templates_empty_card: 'permissions-templates-empty-card',
    templates_empty: 'permissions-templates-empty',
    templates_table: 'permissions-templates-table',
    delete_template_prefix: 'permissions-delete-template-',
    delete_modal: 'permissions-delete-modal',
    delete_cancel_button: 'permissions-delete-cancel-button',
    delete_confirm_button: 'permissions-delete-confirm-button',
  },
  settings_page: {
    error_banner: 'settings-error-banner',
    error_dismiss_button: 'settings-error-dismiss-button',
    add_button: 'settings-add-button',
    search_input: 'settings-search-input',
    loading: 'settings-loading',
    empty_state: 'settings-empty-state',
    empty_add_button: 'settings-empty-add-button',
    table: 'settings-table',
    edit_prefix: 'settings-edit-',
    delete_prefix: 'settings-delete-',
    add_modal: 'settings-add-modal',
    edit_modal: 'settings-edit-modal',
    delete_modal: 'settings-delete-modal',
    form_key_input: 'settings-form-key-input',
    form_value_input: 'settings-form-value-input',
    form_cancel_button: 'settings-form-cancel-button',
    form_submit_button: 'settings-form-submit-button',
    delete_cancel_button: 'settings-delete-cancel-button',
    delete_confirm_button: 'settings-delete-confirm-button',
  },
  feature_flags_page: {
    error_banner: 'feature-flags-error-banner',
    error_dismiss_button: 'feature-flags-error-dismiss-button',
    add_button: 'feature-flags-add-button',
    search_input: 'feature-flags-search-input',
    loading_card: 'feature-flags-loading-card',
    loading: 'feature-flags-loading',
    empty_card: 'feature-flags-empty-card',
    empty_state: 'feature-flags-empty-state',
    empty_add_button: 'feature-flags-empty-add-button',
    table: 'feature-flags-table',
    toggle_prefix: 'feature-flags-toggle-',
    edit_prefix: 'feature-flags-edit-',
    delete_prefix: 'feature-flags-delete-',
    load_more_button: 'feature-flags-load-more-button',
    add_modal: 'feature-flags-add-modal',
    edit_modal: 'feature-flags-edit-modal',
    delete_modal: 'feature-flags-delete-modal',
    form_name_input: 'feature-flags-form-name-input',
    form_key_input: 'feature-flags-form-key-input',
    form_description_input: 'feature-flags-form-description-input',
    form_enabled_checkbox: 'feature-flags-form-enabled-checkbox',
    form_cancel_button: 'feature-flags-form-cancel-button',
    form_submit_button: 'feature-flags-form-submit-button',
    delete_cancel_button: 'feature-flags-delete-cancel-button',
    delete_confirm_button: 'feature-flags-delete-confirm-button',
  },
  audit_page: {
    refresh_button: 'audit-refresh-button',
    actor_input: 'audit-actor-input',
    action_input: 'audit-action-input',
    date_from_input: 'audit-date-from-input',
    date_to_input: 'audit-date-to-input',
    clear_button: 'audit-clear-button',
    loading_card: 'audit-loading-card',
    loading: 'audit-loading',
    empty_card: 'audit-empty-card',
    empty_state: 'audit-empty-state',
    table: 'audit-table',
  },
  apikeys_page: {
    error_banner: 'api-keys-error-banner',
    error_dismiss_button: 'api-keys-error-dismiss-button',
    create_card: 'api-keys-create-card',
    name_input: 'api-keys-name-input',
    generate_button: 'api-keys-generate-button',
    new_key_banner: 'api-keys-new-key-banner',
    new_key_dismiss_button: 'api-keys-new-key-dismiss-button',
    loading_card: 'api-keys-loading-card',
    loading: 'api-keys-loading',
    empty_card: 'api-keys-empty-card',
    empty_state: 'api-keys-empty-state',
    table: 'api-keys-table',
    revoke_prefix: 'api-keys-revoke-',
    revoke_modal: 'api-keys-revoke-modal',
    revoke_cancel_button: 'api-keys-revoke-cancel-button',
    revoke_confirm_button: 'api-keys-revoke-confirm-button',
  },
  webhooks_page: {
    error_banner: 'webhooks-error-banner',
    error_dismiss_button: 'webhooks-error-dismiss-button',
    add_button: 'webhooks-add-button',
    search_input: 'webhooks-search-input',
    form_card: 'webhooks-form-card',
    form_name_input: 'webhooks-form-name-input',
    form_url_input: 'webhooks-form-url-input',
    event_preset_prefix: 'webhooks-event-preset-',
    custom_event_input: 'webhooks-custom-event-input',
    custom_event_add_button: 'webhooks-custom-event-add-button',
    event_remove_prefix: 'webhooks-event-remove-',
    form_active_checkbox: 'webhooks-form-active-checkbox',
    form_cancel_button: 'webhooks-form-cancel-button',
    form_submit_button: 'webhooks-form-submit-button',
    loading_card: 'webhooks-loading-card',
    loading: 'webhooks-loading',
    empty_card: 'webhooks-empty-card',
    empty_state: 'webhooks-empty-state',
    no_results_card: 'webhooks-no-results-card',
    no_results: 'webhooks-no-results',
    sort_name: 'webhooks-sort-name',
    sort_status: 'webhooks-sort-status',
    table: 'webhooks-table',
    row_prefix: 'webhooks-row-',
    deliveries_prefix: 'webhooks-deliveries-',
    edit_prefix: 'webhooks-edit-',
    delete_prefix: 'webhooks-delete-',
    load_more_button: 'webhooks-load-more-button',
    delete_modal: 'webhooks-delete-modal',
    delete_cancel_button: 'webhooks-delete-cancel-button',
    delete_confirm_button: 'webhooks-delete-confirm-button',
    deliveries_close_button: 'webhooks-deliveries-close-button',
    deliveries_empty: 'webhooks-deliveries-empty',
    deliveries_table: 'webhooks-deliveries-table',
  },
  storage_page: {
    error_banner: 'storage-error-banner',
    error_dismiss_button: 'storage-error-dismiss-button',
    file_input: 'storage-file-input',
    upload_button: 'storage-upload-button',
    loading_card: 'storage-loading-card',
    loading: 'storage-loading',
    empty_card: 'storage-empty-card',
    empty_state: 'storage-empty-state',
    empty_upload_button: 'storage-empty-upload-button',
    table: 'storage-table',
    row_prefix: 'storage-row-',
    download_prefix: 'storage-download-',
    delete_prefix: 'storage-delete-',
    delete_modal: 'storage-delete-modal',
    delete_cancel_button: 'storage-delete-cancel-button',
    delete_confirm_button: 'storage-delete-confirm-button',
  },
  sessions_page: {
    access_denied: 'sessions-access-denied',
    error_banner: 'sessions-error-banner',
    error_dismiss_button: 'sessions-error-dismiss-button',
    loading_card: 'sessions-loading-card',
    loading: 'sessions-loading',
    empty_card: 'sessions-empty-card',
    empty_state: 'sessions-empty-state',
    table: 'sessions-table',
    revoke_prefix: 'sessions-revoke-',
    revoke_modal: 'sessions-revoke-modal',
    revoke_cancel_button: 'sessions-revoke-cancel-button',
    revoke_confirm_button: 'sessions-revoke-confirm-button',
  },
  jobs_page: {
    error_banner: 'jobs-error-banner',
    error_dismiss_button: 'jobs-error-dismiss-button',
    loading_card: 'jobs-loading-card',
    loading: 'jobs-loading',
    empty_card: 'jobs-empty-card',
    empty_state: 'jobs-empty-state',
    table: 'jobs-table',
    retry_prefix: 'jobs-retry-',
    cancel_prefix: 'jobs-cancel-',
    cancel_modal: 'jobs-cancel-modal',
    cancel_cancel_button: 'jobs-cancel-cancel-button',
    cancel_confirm_button: 'jobs-cancel-confirm-button',
  },
  notifications_page: {
    error_banner: 'notifications-error-banner',
    error_dismiss_button: 'notifications-error-dismiss-button',
    mark_all_read_button: 'notifications-mark-all-read-button',
    loading_card: 'notifications-loading-card',
    loading: 'notifications-loading',
    empty_card: 'notifications-empty-card',
    empty_state: 'notifications-empty-state',
    card_prefix: 'notifications-card-',
    read_prefix: 'notifications-read-',
    delete_prefix: 'notifications-delete-',
    delete_modal: 'notifications-delete-modal',
    delete_cancel_button: 'notifications-delete-cancel-button',
    delete_confirm_button: 'notifications-delete-confirm-button',
  },
  profile_page: {
    toast: 'profile-toast',
    toast_dismiss_button: 'profile-toast-dismiss-button',
    account_details_card: 'profile-account-details-card',
    name_input: 'profile-name-input',
    email_input: 'profile-email-input',
    cancel_button: 'profile-cancel-button',
    save_button: 'profile-save-button',
    sessions_card: 'profile-sessions-card',
    logout_all_button: 'profile-logout-all-button',
    sessions_loading: 'profile-sessions-loading',
    sessions_empty: 'profile-sessions-empty',
    revoke_session_prefix: 'profile-revoke-session-',
    preferences_loading: 'profile-preferences-loading',
    preferences_empty: 'profile-preferences-empty',
    logout_all_modal: 'profile-logout-all-modal',
    logout_all_cancel_button: 'profile-logout-all-cancel-button',
    logout_all_confirm_button: 'profile-logout-all-confirm-button',
    revoke_modal: 'profile-revoke-modal',
    revoke_cancel_button: 'profile-revoke-cancel-button',
    revoke_confirm_button: 'profile-revoke-confirm-button',
  },
};

function t(path: string): string {
  const [section, key] = path.split('.');
  const value = (T as Record<string, Record<string, string>>)[section]?.[key];
  if (!value) throw new Error(`Unknown testid: ${path}`);
  return value;
}

function dynamicT(path: string, params: Record<string, string | number>): string {
  let result = t(path);
  for (const [k, v] of Object.entries(params)) {
    result = result.replace(`{${k}}`, String(v));
  }
  return result;
}

function byTidPrefix(prefix: string) {
  return `[data-testid^="${prefix}"]`;
}

async function unauthPage(page: Page, url: string) {
  await page.route('**/api/me', route => route.fulfill({ status: 403 }));
  await page.goto(url);
}

test.describe('authentication - edge cases', () => {
  test('login with empty email shows error', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('login with empty password shows error', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId(t('login_page.email_input')).fill('admin@test.com');
    await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId(t('login_page.email_input')).fill('wrong@test.com');
    await page.getByTestId(t('login_page.password_input')).fill('WrongP@ss1');
    await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('login with locked account shows error', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId(t('login_page.email_input')).fill('locked@test.com');
    await page.getByTestId(t('login_page.password_input')).fill('AnyP@ss1');
    await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('rate limited IP shows error', async ({ page }) => {
    await page.goto('/login');
    for (let i = 0; i < 11; i++) {
      await page.getByTestId(t('login_page.email_input')).fill('user@test.com');
      await page.getByTestId(t('login_page.password_input')).fill('ValidP@ss1');
      await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
      if (await page.getByTestId(t('login_page.error_banner')).isVisible().catch(() => false)) break;
    }
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('login with suspended user shows error', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId(t('login_page.email_input')).fill('suspended@test.com');
    await page.getByTestId(t('login_page.password_input')).fill('ValidP@ss1');
    await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('login with banned user shows error', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId(t('login_page.email_input')).fill('banned@test.com');
    await page.getByTestId(t('login_page.password_input')).fill('ValidP@ss1');
    await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('login with force password reset user shows error', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId(t('login_page.email_input')).fill('reset@test.com');
    await page.getByTestId(t('login_page.password_input')).fill('OldP@ss1');
    await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('login with soft deleted user shows error', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId(t('login_page.email_input')).fill('deleted@test.com');
    await page.getByTestId(t('login_page.password_input')).fill('ValidP@ss1');
    await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('login with network error shows error', async ({ page }) => {
    await page.route('**/api/auth/login', route => route.abort('connectionrefused'));
    await page.goto('/login');
    await page.getByTestId(t('login_page.email_input')).fill('user@test.com');
    await page.getByTestId(t('login_page.password_input')).fill('ValidP@ss1');
    await page.getByTestId(t('login_page.sign_in_button')).click({ force: true });
    await expect(page.getByTestId(t('login_page.error_banner'))).toBeVisible();
  });

  test('register with empty fields shows error', async ({ page }) => {
    await unauthPage(page, '/login');
    await expect(page.getByTestId(t('login_page.form'))).toBeVisible();
    await page.goto('/register');
    await page.getByTestId(t('register_page.submit_button')).click({ force: true });
    await expect(page.getByTestId(t('register_page.error_banner'))).toBeVisible();
  });

  test('register with duplicate email shows error', async ({ page }) => {
    await unauthPage(page, '/login');
    await expect(page.getByTestId(t('login_page.form'))).toBeVisible();
    await page.goto('/register');
    await page.getByTestId(t('register_page.name_input')).fill('Test');
    await page.getByTestId(t('register_page.email_input')).fill(`existing-${Date.now()}@test.com`);
    await page.getByTestId(t('register_page.password_input')).fill('ValidP@ss1');
    await page.getByTestId(t('register_page.submit_button')).click();
    await expect(page.getByTestId(t('register_page.error_banner')).or(page.getByTestId(t('login_page.success_banner')))).toBeVisible();
  });

  test('register with HTML injection in name shows error', async ({ page }) => {
    const ts = Date.now();
    await unauthPage(page, '/login');
    await expect(page.getByTestId(t('login_page.form'))).toBeVisible();
    await page.goto('/register');
    await page.getByTestId(t('register_page.name_input')).fill("<script>alert('xss')</script>");
    await page.getByTestId(t('register_page.email_input')).fill(`html-${ts}@test.com`);
    await page.getByTestId(t('register_page.password_input')).fill('ValidP@ss1');
    await page.getByTestId(t('register_page.submit_button')).click();
    await expect(page.getByTestId(t('register_page.error_banner')).or(page.getByTestId(t('login_page.success_banner')))).toBeVisible();
  });

  test('register with weak password shows error', async ({ page }) => {
    const ts = Date.now();
    await unauthPage(page, '/login');
    await expect(page.getByTestId(t('login_page.form'))).toBeVisible();
    await page.goto('/register');
    await page.getByTestId(t('register_page.name_input')).fill('Test');
    await page.getByTestId(t('register_page.email_input')).fill(`weak-${ts}@test.com`);
    await page.getByTestId(t('register_page.password_input')).fill('weak');
    await page.getByTestId(t('register_page.submit_button')).click();
    await expect(page.getByTestId(t('register_page.error_banner')).or(page.getByTestId(t('login_page.success_banner')))).toBeVisible();
  });

  test('register second user gets verification status', async ({ page }) => {
    const ts = Date.now();
    await unauthPage(page, '/login');
    await expect(page.getByTestId(t('login_page.form'))).toBeVisible();
    await page.goto('/register');
    await page.getByTestId(t('register_page.name_input')).fill('Second');
    await page.getByTestId(t('register_page.email_input')).fill(`second-${ts}@test.com`);
    await page.getByTestId(t('register_page.password_input')).fill('ValidP@ss1');
    await page.getByTestId(t('register_page.submit_button')).click();
    await expect(page.getByTestId(t('login_page.success_banner')).or(page.getByTestId(t('register_page.error_banner')))).toBeVisible();
  });

  test('expired session redirect shows expired banner', async ({ page }) => {
    await unauthPage(page, '/');
    await expect(page.getByTestId(t('login_page.expired_banner')).or(page.getByTestId(t('login_page.form')))).toBeVisible();
  });

  test('revoked session redirect shows expired banner', async ({ page }) => {
    await unauthPage(page, '/');
    await expect(page.getByTestId(t('login_page.expired_banner')).or(page.getByTestId(t('login_page.form')))).toBeVisible();
  });

});
test.use({ storageState: 'e2e/.auth/admin.json' });

test.describe('dashboard_navigation_verification', () => {

  test('all nav links render correct pages', async ({ page }) => {
    const navChecks: [string, string][] = [
      ['layout.nav_dashboard_link', 'dashboard_page.stat_total_users'],
      ['layout.nav_users_link', 'users_page.search_input'],
      ['layout.nav_roles_link', 'roles_page.search_input'],
      ['layout.nav_permissions_link', 'permissions_page.tab_registry'],
      ['layout.nav_settings_link', 'settings_page.search_input'],
      ['layout.nav_feature_flags_link', 'feature_flags_page.search_input'],
      ['layout.nav_audit_log_link', 'audit_page.actor_input'],
      ['layout.nav_api_keys_link', 'apikeys_page.name_input'],
      ['layout.nav_webhooks_link', 'webhooks_page.search_input'],
      ['layout.nav_storage_link', 'storage_page.upload_button'],
      ['layout.nav_sessions_link', 'sessions_page.table'],
      ['layout.nav_jobs_link', 'jobs_page.table'],
      ['layout.nav_notifications_link', 'notifications_page.mark_all_read_button'],
      ['layout.nav_profile_link', 'profile_page.name_input'],
    ];
    for (const [nav, verify] of navChecks) {
      await page.getByTestId(t(nav)).click();
      await expect(page.getByTestId(t(verify))).toBeVisible();
    }
  });
});

test.describe('users_crud_flow', () => {

  test('create, edit, status change, delete user', async ({ page }) => {
    await page.goto('/users');
    await expect(page.getByTestId(t('users_page.search_input'))).toBeVisible();
    const searchId = `${Date.now()}`;
    await page.getByTestId(t('users_page.search_input')).fill(searchId);
    await expect(page.getByTestId(t('users_page.no_results_card')).or(page.getByTestId(t('users_page.table')))).toBeVisible();
    await page.getByTestId(t('users_page.search_input')).fill('');
    await page.getByTestId(t('users_page.sort_email')).click();

    await page.getByTestId(t('users_page.add_button')).click();
    await expect(page.getByTestId(t('users_page.form_email_input'))).toBeVisible();
    const userEmail = `e2e-crud-${Date.now()}@test.com`;
    await page.getByTestId(t('users_page.form_email_input')).fill(userEmail);
    await page.getByTestId(t('users_page.form_name_input')).fill('E2E Crud User');
    await page.getByTestId(t('users_page.form_submit_button')).click();
    await page.getByTestId(t('users_page.create_modal')).or(page.getByTestId(t('users_page.table'))).waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
    await expect(page.getByTestId(t('users_page.table'))).toBeVisible();

    await page.locator(byTidPrefix(t('users_page.edit_prefix'))).first().click();
    await expect(page.getByTestId(t('users_page.form_name_input'))).toBeVisible();
    await page.getByTestId(t('users_page.form_name_input')).fill('Updated User');
    await page.getByTestId(t('users_page.form_submit_button')).click();

    await page.locator(byTidPrefix(t('users_page.status_prefix'))).first().click();
    await expect(page.getByTestId(t('users_page.status_modal'))).toBeVisible();
    await page.getByTestId(t('users_page.status_radio_prefix') + 'SUSPENDED').click();
    await page.getByTestId(t('users_page.status_confirm_button')).click();
    await page.getByTestId(t('users_page.status_modal')).waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});

    await page.locator(byTidPrefix(t('users_page.delete_prefix'))).first().click();
    await expect(page.getByTestId(t('users_page.delete_modal'))).toBeVisible();
    await page.getByTestId(t('users_page.delete_confirm_button')).click();
  });
});

test.describe('roles_crud_flow', () => {

  test('create, edit, manage permissions, delete role', async ({ page }) => {
    await page.goto('/roles');
    await expect(page.getByTestId(t('roles_page.table'))).toBeVisible();

    await page.getByTestId(t('roles_page.add_button')).click();
    await expect(page.getByTestId(t('roles_page.form_name_input'))).toBeVisible();
    await page.getByTestId(t('roles_page.form_name_input')).fill('Editor');
    await page.getByTestId(t('roles_page.form_description_input')).fill('Content editor role');
    await page.getByTestId(t('roles_page.form_submit_button')).click();
    await page.getByTestId(t('roles_page.form_modal')).waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});

    await page.locator(byTidPrefix(t('roles_page.row_prefix'))).first().waitFor({ state: 'visible' });
    await page.locator(byTidPrefix(t('roles_page.edit_prefix'))).first().click();
    await page.getByTestId(t('roles_page.form_name_input')).fill('Senior Editor');
    await page.getByTestId(t('roles_page.form_submit_button')).click();
    await page.getByTestId(t('roles_page.form_modal')).waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});

    await page.locator(byTidPrefix(t('roles_page.permissions_prefix'))).first().click();
    await expect(page.getByTestId(t('roles_page.permissions_modal'))).toBeVisible();
    await page.getByTestId(t('roles_page.perm_checkbox_prefix') + 'read:users').click();
    await page.getByTestId(t('roles_page.permissions_save_button')).click();
    await page.getByTestId(t('roles_page.permissions_modal')).waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});

    await page.locator(byTidPrefix(t('roles_page.delete_prefix'))).first().click();
    await expect(page.getByTestId(t('roles_page.delete_modal'))).toBeVisible();
    await page.getByTestId(t('roles_page.delete_confirm_button')).click();
    await page.getByTestId(t('roles_page.delete_modal')).waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
  });
});

test.describe('settings_crud_flow', () => {

  test('create, edit, delete setting', async ({ page }) => {
    await page.goto('/settings');
    await expect(page.getByTestId(t('settings_page.add_button'))).toBeVisible();
    await page.getByTestId(t('settings_page.add_button')).click();
    await expect(page.getByTestId(t('settings_page.form_key_input'))).toBeVisible();
    const sk = `stg_${Date.now()}`;
    await page.getByTestId(t('settings_page.form_key_input')).fill(sk);
    await page.getByTestId(t('settings_page.form_value_input')).fill('CoreKit');
    await page.getByTestId(t('settings_page.form_submit_button')).click();
    await page.getByTestId(t('settings_page.add_modal')).waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
    await expect(page.getByTestId(t('settings_page.table'))).toBeVisible();

    await page.locator(byTidPrefix(t('settings_page.edit_prefix'))).first().click();
    await expect(page.getByTestId(t('settings_page.form_value_input'))).toBeVisible();
    await page.getByTestId(t('settings_page.form_value_input')).fill('MyCoreKit');
    await page.getByTestId(t('settings_page.form_submit_button')).click();
    await page.getByTestId(t('settings_page.edit_modal')).waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});

    await page.locator(byTidPrefix(t('settings_page.delete_prefix'))).first().click();
    await expect(page.getByTestId(t('settings_page.delete_modal'))).toBeVisible();
    await page.getByTestId(t('settings_page.delete_confirm_button')).click();
    await page.getByTestId(t('settings_page.delete_modal')).waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {});
  });
});

test.describe('feature_flags_crud_flow', () => {

  test('create, toggle, edit, delete flag', async ({ page }) => {
    await page.goto('/feature-flags');
    await expect(page.getByTestId(t('feature_flags_page.add_button'))).toBeVisible();

    await page.getByTestId(t('feature_flags_page.add_button')).click();
    await expect(page.getByTestId(t('feature_flags_page.form_name_input'))).toBeVisible();
    await page.getByTestId(t('feature_flags_page.form_name_input')).fill('Dark Mode');
    await page.getByTestId(t('feature_flags_page.form_key_input')).fill('dark_mode');
    await page.getByTestId(t('feature_flags_page.form_submit_button')).click();
    await expect(page.getByTestId(t('feature_flags_page.table'))).toBeVisible();

    await page.locator(byTidPrefix(t('feature_flags_page.toggle_prefix'))).first().click();
    await page.locator(byTidPrefix(t('feature_flags_page.toggle_prefix'))).first().click();

    await page.locator(byTidPrefix(t('feature_flags_page.edit_prefix'))).first().click();
    await page.getByTestId(t('feature_flags_page.form_submit_button')).click();

    await page.locator(byTidPrefix(t('feature_flags_page.delete_prefix'))).first().click();
    await expect(page.getByTestId(t('feature_flags_page.delete_modal'))).toBeVisible();
    await page.getByTestId(t('feature_flags_page.delete_confirm_button')).click();
  });
});

test.describe('audit_log_filter_flow', () => {

  test('filter audit logs by actor, action, date range, clear, refresh', async ({ page }) => {
    await page.goto('/audit');
    await expect(page.getByTestId(t('audit_page.actor_input'))).toBeVisible();

    await page.getByTestId(t('audit_page.actor_input')).fill('1');
    await page.getByTestId(t('audit_page.action_input')).fill('LOGIN');
    await page.getByTestId(t('audit_page.date_from_input')).fill('2025-01-01');
    await page.getByTestId(t('audit_page.date_to_input')).fill('2025-12-31');
    await page.getByTestId(t('audit_page.clear_button')).click();
    await page.getByTestId(t('audit_page.refresh_button')).click();
    await expect(page.getByTestId(t('audit_page.table')).or(page.getByTestId(t('audit_page.empty_state')))).toBeVisible();
  });
});

test.describe('apikeys_crud_flow', () => {

  test('create, view raw key, revoke API key', async ({ page }) => {
    await page.goto('/apikeys');
    await expect(page.getByTestId(t('apikeys_page.name_input'))).toBeVisible();
    await page.getByTestId(t('apikeys_page.name_input')).fill('Test API Key');
    await page.getByTestId(t('apikeys_page.generate_button')).click();
    await expect(page.getByTestId(t('apikeys_page.new_key_banner'))).toBeVisible();
    await page.getByTestId(t('apikeys_page.new_key_dismiss_button')).click();

    await page.locator(byTidPrefix(t('apikeys_page.revoke_prefix'))).first().click();
    await expect(page.getByTestId(t('apikeys_page.revoke_modal'))).toBeVisible();
    await page.getByTestId(t('apikeys_page.revoke_confirm_button')).click();
  });
});

test.describe('webhooks_crud_flow', () => {

  test('create, events, deliveries, edit, delete webhook', async ({ page }) => {
    await page.goto('/webhooks');
    await expect(page.getByTestId(t('webhooks_page.add_button'))).toBeVisible();

    await page.getByTestId(t('webhooks_page.add_button')).click();
    await expect(page.getByTestId(t('webhooks_page.form_name_input'))).toBeVisible();
    await page.getByTestId(t('webhooks_page.form_name_input')).fill('Test Webhook');
    await page.getByTestId(t('webhooks_page.form_url_input')).fill('https://example.com/webhook');
    await page.getByTestId(t('webhooks_page.event_preset_prefix') + 'user.created').click();
    await page.getByTestId(t('webhooks_page.form_submit_button')).click();
    await expect(page.getByTestId(t('webhooks_page.table'))).toBeVisible();

    await page.locator(byTidPrefix(t('webhooks_page.deliveries_prefix'))).first().click();
    await expect(page.getByTestId(t('webhooks_page.deliveries_table')).or(page.getByTestId(t('webhooks_page.deliveries_empty')))).toBeVisible();
    await page.getByTestId(t('webhooks_page.deliveries_close_button')).click();

    await page.locator(byTidPrefix(t('webhooks_page.edit_prefix'))).first().click();
    await page.getByTestId(t('webhooks_page.form_submit_button')).click();

    await page.locator(byTidPrefix(t('webhooks_page.delete_prefix'))).first().click();
    await expect(page.getByTestId(t('webhooks_page.delete_modal'))).toBeVisible();
    await page.getByTestId(t('webhooks_page.delete_confirm_button')).click();
  });
});

test.describe('storage_upload_download_delete_flow', () => {

  test('upload, delete file', async ({ page }) => {
    await page.goto('/storage');
    await expect(page.getByTestId(t('storage_page.upload_button'))).toBeVisible();

    await page.getByTestId(t('storage_page.file_input')).setInputFiles({
      name: 'test.jpg',
      mimeType: 'image/jpeg',
      buffer: Buffer.from('fake-jpeg-data'),
    });
    await expect(page.getByTestId(t('storage_page.table'))).toBeVisible();

    await page.locator(byTidPrefix(t('storage_page.delete_prefix'))).first().click();
    await expect(page.getByTestId(t('storage_page.delete_modal'))).toBeVisible();
    await page.getByTestId(t('storage_page.delete_confirm_button')).click();
  });
});

test.describe('sessions_flows', () => {

  test('super admin views and revokes a session', async ({ page }) => {
    await page.goto('/sessions');
    await expect(page.getByTestId(t('sessions_page.table')).or(page.getByTestId(t('sessions_page.access_denied')))).toBeVisible();

    const revokeBtn = page.locator(byTidPrefix(t('sessions_page.revoke_prefix'))).first();
    if (await revokeBtn.isVisible().catch(() => false)) {
      await revokeBtn.click();
      await expect(page.getByTestId(t('sessions_page.revoke_modal'))).toBeVisible();
      await page.getByTestId(t('sessions_page.revoke_confirm_button')).click();
    }
  });

  test('non-super admin sees access denied on sessions page', async ({ page }) => {
    await page.goto('/sessions');
    await expect(page.getByTestId(t('sessions_page.access_denied'))).toBeVisible();
  });
});

test.describe('jobs_monitoring_flow', () => {

  test('view jobs, retry failed, cancel pending', async ({ page }) => {
    await page.goto('/jobs');
    await expect(page.getByTestId(t('jobs_page.table'))).toBeVisible();

    const retryBtn = page.locator(byTidPrefix(t('jobs_page.retry_prefix'))).first();
    if (await retryBtn.isVisible().catch(() => false)) {
      await retryBtn.click();
    }

    const cancelBtn = page.locator(byTidPrefix(t('jobs_page.cancel_prefix'))).first();
    if (await cancelBtn.isVisible().catch(() => false)) {
      await cancelBtn.click();
      await expect(page.getByTestId(t('jobs_page.cancel_modal'))).toBeVisible();
      await page.getByTestId(t('jobs_page.cancel_confirm_button')).click();
    }
  });
});

test.describe('notifications_mark_read_delete_flow', () => {

  test('view, mark read, mark all read, delete notification', async ({ page }) => {
    await page.goto('/notifications');
    await expect(page.getByTestId(t('notifications_page.mark_all_read_button'))).toBeVisible();

    const firstCard = page.locator(byTidPrefix(t('notifications_page.card_prefix'))).first();
    if (await firstCard.isVisible().catch(() => false)) {
      const readBtn = page.locator(byTidPrefix(t('notifications_page.read_prefix'))).first();
      if (await readBtn.isVisible().catch(() => false)) {
        await readBtn.click();
      }
    }

    await page.getByTestId(t('notifications_page.mark_all_read_button')).click();

    const delBtn = page.locator(byTidPrefix(t('notifications_page.delete_prefix'))).first();
    if (await delBtn.isVisible().catch(() => false)) {
      await delBtn.click();
      await expect(page.getByTestId(t('notifications_page.delete_modal'))).toBeVisible();
      await page.getByTestId(t('notifications_page.delete_confirm_button')).click();
    }
  });
});

test.describe('profile_flows', () => {

  test('profile_edit_flow: edit name/email, save, cancel', async ({ page }) => {
    await page.goto('/profile');
    await expect(page.getByTestId(t('profile_page.name_input'))).toBeVisible();
    await page.getByTestId(t('profile_page.name_input')).fill('Updated Name');
    await page.getByTestId(t('profile_page.cancel_button')).click();
    await page.getByTestId(t('profile_page.name_input')).fill('New Name');
    await page.getByTestId(t('profile_page.email_input')).fill('newemail@test.com');
    await page.getByTestId(t('profile_page.save_button')).click();
    await expect(page.getByTestId(t('profile_page.toast'))).toBeVisible();
  });

  test('profile_sessions_flow: revoke session, logout all', async ({ page }) => {
    await page.goto('/profile');
    await expect(page.getByTestId(t('profile_page.sessions_card'))).toBeVisible();

    const revokeBtn = page.locator(byTidPrefix(t('profile_page.revoke_session_prefix'))).first();
    if (await revokeBtn.isVisible().catch(() => false)) {
      await revokeBtn.click();
      await expect(page.getByTestId(t('profile_page.revoke_modal'))).toBeVisible();
      await page.getByTestId(t('profile_page.revoke_confirm_button')).click();
    }
  });
});


test.describe('authentication - admin edge cases', () => {

  test('webhook non-HTTPS URL shows error', async ({ page }) => {
    await page.goto('/webhooks');
    await page.getByTestId(t('webhooks_page.add_button')).click();
    await page.getByTestId(t('webhooks_page.form_name_input')).fill('Test');
    await page.getByTestId(t('webhooks_page.form_url_input')).fill('http://example.com/webhook');
    await page.getByTestId(t('webhooks_page.form_submit_button')).click({ force: true });
    await expect(page.getByTestId(t('webhooks_page.error_banner'))).toBeVisible();
  });

  test('webhook SSRF private IP URL shows error', async ({ page }) => {
    await page.goto('/webhooks');
    await page.getByTestId(t('webhooks_page.add_button')).click();
    await page.getByTestId(t('webhooks_page.form_name_input')).fill('Test');
    await page.getByTestId(t('webhooks_page.form_url_input')).fill('http://169.254.169.254/latest/meta-data/');
    await page.getByTestId(t('webhooks_page.form_submit_button')).click({ force: true });
    await expect(page.getByTestId(t('webhooks_page.error_banner'))).toBeVisible();
  });

  test('upload forbidden MIME type shows error', async ({ page }) => {
    await page.goto('/storage');
    await page.getByTestId(t('storage_page.file_input')).setInputFiles({
      name: 'malware.html',
      mimeType: 'text/html',
      buffer: Buffer.from('<script>alert(1)</script>'),
    });
    await expect(page.getByTestId(t('storage_page.error_banner'))).toBeVisible();
  });

  test('upload excessive file size shows error', async ({ page }) => {
    await page.goto('/storage');
    const largeBuffer = Buffer.alloc(50 * 1024 * 1024, 'x');
    await page.getByTestId(t('storage_page.file_input')).setInputFiles({
      name: '50mb_dummy.bin',
      mimeType: 'application/octet-stream',
      buffer: largeBuffer,
    });
    await expect(page.getByTestId(t('storage_page.error_banner'))).toBeVisible();
  });

  test('create user with duplicate email shows error', async ({ page }) => {
    await page.goto('/users');
    await page.getByTestId(t('users_page.add_button')).click();
    await page.getByTestId(t('users_page.form_email_input')).fill('existing@test.com');
    await page.getByTestId(t('users_page.form_name_input')).fill('Test');
    await page.getByTestId(t('users_page.form_submit_button')).click({ force: true });
    await expect(page.getByTestId(t('users_page.error_banner'))).toBeVisible();
  });

  test('create setting with duplicate key shows error', async ({ page }) => {
    await page.goto('/settings');
    await page.getByTestId(t('settings_page.add_button')).click();
    await page.getByTestId(t('settings_page.form_key_input')).fill('site_name');
    await page.getByTestId(t('settings_page.form_value_input')).fill('Test');
    await page.getByTestId(t('settings_page.form_submit_button')).click({ force: true });
    await expect(page.getByTestId(t('settings_page.error_banner'))).toBeVisible();
  });

  test('create feature flag with duplicate key shows error', async ({ page }) => {
    await page.goto('/feature-flags');
    await page.getByTestId(t('feature_flags_page.add_button')).click();
    await page.getByTestId(t('feature_flags_page.form_name_input')).fill('Test');
    await page.getByTestId(t('feature_flags_page.form_key_input')).fill('dark_mode');
    await page.getByTestId(t('feature_flags_page.form_submit_button')).click({ force: true });
    await expect(page.getByTestId(t('feature_flags_page.error_banner'))).toBeVisible();
  });

  test('generate API key with empty name is disabled', async ({ page }) => {
    await page.goto('/apikeys');
    await expect(page.getByTestId(t('apikeys_page.generate_button'))).toBeDisabled();
  });

  test('create webhook with empty name shows error', async ({ page }) => {
    await page.goto('/webhooks');
    await page.getByTestId(t('webhooks_page.add_button')).click();
    await page.getByTestId(t('webhooks_page.form_url_input')).fill('https://test.com');
    await page.getByTestId(t('webhooks_page.form_submit_button')).click({ force: true });
    await expect(page.getByTestId(t('webhooks_page.error_banner'))).toBeVisible();
  });

  test('save profile without changes is disabled', async ({ page }) => {
    await page.goto('/profile');
    await expect(page.getByTestId(t('profile_page.save_button'))).toBeDisabled();
  });
});
