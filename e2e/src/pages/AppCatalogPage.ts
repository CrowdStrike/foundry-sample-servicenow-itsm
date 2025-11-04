/**
 * AppCatalogPage - App installation and management
 */

import { Page } from '@playwright/test';
import { BasePage } from './BasePage';
import { RetryHandler } from '../utils/SmartWaiter';
import { config } from '../config/TestConfig';

export class AppCatalogPage extends BasePage {
  constructor(page: Page) {
    super(page, 'AppCatalogPage');
  }

  protected getPagePath(): string {
    return '/foundry/app-catalog';
  }

  protected async verifyPageLoaded(): Promise<void> {
    // Use the heading which is unique
    await this.waiter.waitForVisible(
      this.page.locator('h1:has-text("App catalog")'),
      { description: 'App Catalog page' }
    );

    this.logger.success('App Catalog page loaded successfully');
  }

  /**
   * Search for app in catalog and navigate to its page
   */
  private async searchAndNavigateToApp(appName: string): Promise<void> {
    this.logger.info(`Searching for app '${appName}' in catalog`);

    await this.navigateToPath('/foundry/app-catalog', 'App catalog page');

    const searchBox = this.page.getByRole('searchbox', { name: 'Search' });
    await searchBox.fill(appName);
    await this.page.keyboard.press('Enter');
    await this.page.waitForLoadState('networkidle');

    const appLink = this.page.getByRole('link', { name: appName, exact: true });

    try {
      await this.waiter.waitForVisible(appLink, {
        description: `App '${appName}' link in catalog`,
        timeout: 10000
      });
      this.logger.success(`Found app '${appName}' in catalog`);
      await this.smartClick(appLink, `App '${appName}' link`);
      await this.page.waitForLoadState('networkidle');
    } catch (error) {
      throw new Error(`Could not find app '${appName}' in catalog. Make sure the app is deployed.`);
    }
  }

  /**
   * Check if app is installed
   */
  async isAppInstalled(appName: string): Promise<boolean> {
    this.logger.step(`Check if app '${appName}' is installed`);

    // Search for and navigate to the app's catalog page
    await this.searchAndNavigateToApp(appName);

    // Check for installation indicators on the app's page
    // Simple check: if "Install now" link exists, app is NOT installed
    const installLink = this.page.getByRole('link', { name: 'Install now' });
    const hasInstallLink = await this.elementExists(installLink, 3000);

    const isInstalled = !hasInstallLink;
    this.logger.info(`App '${appName}' installation status: ${isInstalled ? 'Installed' : 'Not installed'}`);

    return isInstalled;
  }

  /**
   * Install app if not already installed
   */
  async installApp(appName: string): Promise<boolean> {
    this.logger.step(`Install app '${appName}'`);

    const isInstalled = await this.isAppInstalled(appName);
    if (isInstalled) {
      this.logger.info(`App '${appName}' is already installed`);
      return false;
    }

    // Click Install now link
    this.logger.info('App not installed, looking for Install now link');
    const installLink = this.page.getByRole('link', { name: 'Install now' });

    await this.waiter.waitForVisible(installLink, { description: 'Install now link' });
    await this.smartClick(installLink, 'Install now link');
    this.logger.info('Clicked Install now, waiting for install page to load');

    // Wait for URL to change to install page and page to stabilize
    await this.page.waitForURL(/\/foundry\/app-catalog\/[^\/]+\/install$/, { timeout: 10000 });
    await this.page.waitForLoadState('networkidle');

    // Handle permissions dialog
    await this.handlePermissionsDialog();

    // Handle app configuration if present
    await this.handleAppConfiguration();

    // Click final install button
    await this.clickInstallAppButton();

    // Wait for installation to complete
    await this.waitForInstallation(appName);

    // Give the backend time to register the installation and update catalog status
    // Some apps take longer to fully register, especially with API integrations
    await this.waiter.delay(10000);

    // Verify the app is actually installed by checking catalog
    const verifyInstalled = await this.isAppInstalled(appName);
    if (!verifyInstalled) {
      this.logger.error(`App '${appName}' installation completed but app is not showing as installed in catalog`);
      return false;
    }

    this.logger.success(`App '${appName}' installed successfully`);
    return true;
  }

  /**
   * Handle permissions dialog if present
   */
  private async handlePermissionsDialog(): Promise<void> {
    const acceptButton = this.page.getByRole('button', { name: /accept.*continue/i });

    if (await this.elementExists(acceptButton, 3000)) {
      this.logger.info('Permissions dialog detected, accepting');
      await this.smartClick(acceptButton, 'Accept and continue button');
      await this.waiter.delay(2000);
    }
  }

  /**
   * Get field context by looking at nearby labels and text
   */
  private async getFieldContext(input: any): Promise<string> {
    try {
      // Try to find the label element
      const id = await input.getAttribute('id');
      if (id) {
        const label = this.page.locator(`label[for="${id}"]`);
        if (await label.isVisible({ timeout: 1000 }).catch(() => false)) {
          const labelText = await label.textContent();
          if (labelText) return labelText.toLowerCase();
        }
      }

      // Look at parent container for context
      const parent = input.locator('xpath=ancestor::div[contains(@class, "form") or contains(@class, "field") or contains(@class, "input")][1]');
      if (await parent.isVisible({ timeout: 1000 }).catch(() => false)) {
        const parentText = await parent.textContent();
        if (parentText) return parentText.toLowerCase();
      }
    } catch (error) {
      // Continue if we can't get context
    }
    return '';
  }

  /**
   * Get value for a field based on its context and name
   */
  private getFieldValue(context: string, name: string): string {
    const combined = `${context} ${name}`.toLowerCase();

    // Check for OAuth/API client IDs first - need realistic base64-like format
    if (/client.*id|clientid|oauth.*id|api.*id/i.test(combined)) {
      return 'MjkzZWY0NWEtZTNiNy00YzJkLWI5ZjYtOGE3YmMxZDIzNDU2';
    }

    // Field mapping based on workflow parameters and documentation
    const fieldMappings = [
      // ServiceNow API Integration patterns - check these first before workflow patterns
      { pattern: /\bname\b(?!.*column)(?!.*guid)/i, value: 'Test ServiceNow Integration' },
      { pattern: /host(?!.*guid)(?!.*column)|url|server/i, value: process.env.SERVICENOW_INSTANCE_URL || 'https://dev123456.service-now.com' },
      { pattern: /username|user.*name(?!.*guid)/i, value: 'foundry_test_user' },
      // Workflow configuration patterns
      { pattern: /table.*name/i, value: 'u_custom_company_access' },
      { pattern: /sysparam.*limit|limit/i, value: '10' },
      { pattern: /cmdb.*app.*name|app.*name.*column/i, value: 'u_cmdb_app_name' },
      { pattern: /host.*guid|hostguid/i, value: 'u_host_guid' },
      { pattern: /idp.*action|action.*column/i, value: 'u_idp_rule_action' },
      { pattern: /idp.*enabled|enabled.*column/i, value: 'u_idp_rule_enabled' },
      { pattern: /idp.*rule.*name.*prefix|prefix/i, value: 'Servicenow_' },
      { pattern: /idp.*simulation|simulation.*mode/i, value: 'u_idp_rule_simulation_mode' },
      { pattern: /idp.*trigger|trigger.*column/i, value: 'u_idp_rule_trigger' },
      { pattern: /sys.*updated.*on|updated.*column/i, value: 'sys_updated_on' },
      { pattern: /user.*guid|userguid/i, value: 'u_user_guid' },
    ];

    for (const { pattern, value } of fieldMappings) {
      if (pattern.test(combined)) {
        return value;
      }
    }

    return 'test-value';
  }

  /**
   * Handle app configuration settings during installation
   * Fills in dummy values for all configuration fields and clicks through settings
   */
  private async handleAppConfiguration(): Promise<void> {
    // Check for multi-instance API integration configuration (x-cs-multi-instance: true)
    // These show all configs in a left sidebar with sections/tabs
    const configSections = this.page.locator('[data-test="config-section-button"], [role="button"][class*="config"], button[class*="section"]');
    const configCount = await configSections.count();

    if (configCount > 1) {
      this.logger.info(`Multi-instance configuration detected with ${configCount} sections`);

      // Click through each configuration section in the sidebar
      for (let i = 0; i < configCount; i++) {
        this.logger.info(`Processing configuration section ${i + 1}/${configCount}`);

        // Click the section button to make it active
        const section = configSections.nth(i);
        if (await section.isVisible({ timeout: 1000 }).catch(() => false)) {
          await this.smartClick(section, `Configuration section ${i + 1}`);
          await this.page.waitForLoadState('networkidle');
          await this.waiter.delay(500);
        }

        // Fill fields for this section
        await this.fillConfigurationFields();
      }
    } else {
      // Single configuration or sequential "Next setting" flow
      // First check if there are any visible configuration fields
      const hasVisibleInputs = await this.page.locator('input[type="text"], input[type="url"], input:not([type="password"]):not([type]), input[type="password"]').count();

      if (hasVisibleInputs > 0) {
        this.logger.info('Configuration fields detected, filling them');
        await this.fillConfigurationFields();
      }

      // Now handle "Next setting" button loop for multi-screen configs
      const nextSettingButton = this.page.getByRole('button', { name: /next setting/i });

      while (await this.elementExists(nextSettingButton, 2000)) {
        this.logger.info('Processing additional configuration screen');
        await this.fillConfigurationFields();

        this.logger.info('Clicking Next setting');
        await this.smartClick(nextSettingButton, 'Next setting button');
        await this.page.waitForLoadState('networkidle');
      }
    }
  }

  /**
   * Fill all visible configuration fields on the current screen
   */
  private async fillConfigurationFields(): Promise<void> {
    // Handle authentication type dropdown if present (select Basic Authentication over OAuth)
    const authTypeButton = this.page.getByRole('button', { name: /OAuth 2\.0 Client Credentials/i });
    if (await this.elementExists(authTypeButton, 2000)) {
      this.logger.info('Found authentication type dropdown, selecting Basic Authentication');
      await this.smartClick(authTypeButton, 'Authentication Type dropdown');
      await this.waiter.delay(500);

      // Select Basic Authentication from dropdown
      const basicAuthOption = this.page.getByRole('option', { name: 'Basic Authentication' });
      await basicAuthOption.waitFor({ state: 'visible', timeout: 10000 });
      await basicAuthOption.click();

      // Wait for form to update with Basic Auth fields
      await this.page.waitForLoadState('networkidle');
      await this.waiter.delay(2000);

      this.logger.success('Selected Basic Authentication');
    }

    // Fill visible text inputs
    const inputs = this.page.locator('input[type="text"], input[type="url"], input:not([type="password"]):not([type])');
    const count = await inputs.count();

    for (let i = 0; i < count; i++) {
      const input = inputs.nth(i);
      if (await input.isVisible()) {
        const name = await input.getAttribute('name') || '';
        const context = await this.getFieldContext(input);

        const value = this.getFieldValue(context, name);
        await input.fill(value);

        this.logger.info(`Filled field (context: "${context.substring(0, 50)}...", name: "${name}") with: ${value}`);
      }
    }

    // Fill password inputs
    const passwordInputs = this.page.locator('input[type="password"]');
    const passwordCount = await passwordInputs.count();

    for (let i = 0; i < passwordCount; i++) {
      const input = passwordInputs.nth(i);
      if (await input.isVisible()) {
        const context = await this.getFieldContext(input);
        const name = await input.getAttribute('name') || '';
        const combined = `${context} ${name}`.toLowerCase();

        // Use realistic-looking credentials based on field type
        let value: string;
        if (/client.*secret|api.*secret|oauth|token/i.test(combined)) {
          // OAuth/API secrets need base64-like format
          value = 'NGY1ZDYyYzgtOTM0Yi00YWUzLWJhNzItMWQ4ZjdhNjhiOWNm';
        } else {
          // Basic auth passwords can use standard password format
          value = 'Test123!Password';
        }

        await input.fill(value);
        this.logger.info(`Filled password field (context: "${context.substring(0, 30)}...") with test credentials`);
      }
    }
  }

  /**
   * Click the final "Save and install" or "Install app" button
   */
  private async clickInstallAppButton(): Promise<void> {
    // Try both button texts - different apps use different wording
    const installButton = this.page.getByRole('button', { name: 'Save and install' })
      .or(this.page.getByRole('button', { name: 'Install app' }));

    await this.waiter.waitForVisible(installButton, { description: 'Install button' });

    // Wait for button to be enabled
    await installButton.waitFor({ state: 'visible', timeout: 10000 });
    await installButton.waitFor({ state: 'attached', timeout: 5000 });

    // Simple delay for form to enable button
    await this.waiter.delay(1000);

    await this.smartClick(installButton, 'Install button');
    this.logger.info('Clicked install button');
  }

  /**
   * Wait for installation to complete
   */
  private async waitForInstallation(appName: string): Promise<void> {
    this.logger.info('Waiting for installation to complete...');

    // Wait for the "installing" toast to appear
    const installingToast = this.page.getByText(/installing/i).first();
    try {
      await installingToast.waitFor({ state: 'visible', timeout: 10000 });
      this.logger.info('Installation started - "installing" toast visible');
    } catch (error) {
      this.logger.warn('Installing toast not visible, checking for installed toast');
    }

    // Wait for either success or failure
    const installedToast = this.page.getByText(/installed/i).first();
    const failedToast = this.page.getByText(/failed|error/i).first();

    try {
      await Promise.race([
        installedToast.waitFor({ state: 'visible', timeout: 20000 }),
        failedToast.waitFor({ state: 'visible', timeout: 20000 })
      ]);

      // Check which one appeared
      const installedVisible = await installedToast.isVisible().catch(() => false);
      const failedVisible = await failedToast.isVisible().catch(() => false);

      if (installedVisible) {
        this.logger.success('Installation completed - "installed" toast visible');
      } else if (failedVisible) {
        const failedText = await failedToast.textContent();
        this.logger.error(`Installation failed - error message: ${failedText}`);
        throw new Error(`App installation failed: ${failedText}`);
      }
    } catch (error) {
      if (error.message?.includes('failed')) {
        throw error;
      }
      this.logger.warn('Neither installed nor failed toast visible, will verify installation status in next step');
    }
  }

  /**
   * Navigate to app via Custom Apps menu
   */
  async navigateToAppViaCustomApps(appName: string): Promise<void> {
    this.logger.step(`Navigate to app '${appName}' via Custom Apps`);

    return RetryHandler.withPlaywrightRetry(
      async () => {
        // Navigate to Foundry home
        await this.navigateToPath('/foundry/home', 'Foundry home page');

        // Open hamburger menu
        const menuButton = this.page.getByRole('button', { name: 'Menu' });
        await this.smartClick(menuButton, 'Menu button');

        // Click Custom apps
        const customAppsButton = this.page.getByRole('button', { name: 'Custom apps' });
        await this.smartClick(customAppsButton, 'Custom apps button');

        // Find and click the app
        const appButton = this.page.getByRole('button', { name: appName, exact: false }).first();
        if (await this.elementExists(appButton, 3000)) {
          await this.smartClick(appButton, `App '${appName}' button`);
          await this.waiter.delay(1000);

          this.logger.success(`Navigated to app '${appName}' via Custom Apps`);
          return;
        }

        throw new Error(`App '${appName}' not found in Custom Apps menu`);
      },
      `Navigate to app via Custom Apps`
    );
  }

  /**
   * Uninstall app
   */
  async uninstallApp(appName: string): Promise<void> {
    this.logger.step(`Uninstall app '${appName}'`);

    try {
      // Search for and navigate to the app's catalog page
      await this.searchAndNavigateToApp(appName);

      // Check if app is actually installed by looking for "Install now" link
      // If "Install now" link exists, app is NOT installed
      const installLink = this.page.getByRole('link', { name: 'Install now' });
      const hasInstallLink = await this.elementExists(installLink, 3000);

      if (hasInstallLink) {
        this.logger.info(`App '${appName}' is already uninstalled`);
        return;
      }

      // Click the 3-dot menu button
      const openMenuButton = this.page.getByRole('button', { name: 'Open menu' });
      await this.waiter.waitForVisible(openMenuButton, { description: 'Open menu button' });
      await this.smartClick(openMenuButton, 'Open menu button');

      // Click "Uninstall app" menuitem
      const uninstallMenuItem = this.page.getByRole('menuitem', { name: 'Uninstall app' });
      await this.waiter.waitForVisible(uninstallMenuItem, { description: 'Uninstall app menuitem' });
      await this.smartClick(uninstallMenuItem, 'Uninstall app menuitem');

      // Confirm uninstallation in modal
      const uninstallButton = this.page.getByRole('button', { name: 'Uninstall' });
      await this.waiter.waitForVisible(uninstallButton, { description: 'Uninstall confirmation button' });
      await this.smartClick(uninstallButton, 'Uninstall button');

      // Wait for success message
      const successMessage = this.page.getByText(/has been uninstalled/i);
      await this.waiter.waitForVisible(successMessage, {
        description: 'Uninstall success message',
        timeout: 30000
      });

      // Give the backend time to register the uninstallation and update catalog status
      await this.waiter.delay(10000);

      this.logger.success(`App '${appName}' uninstalled successfully`);

    } catch (error) {
      this.logger.warn(`Failed to uninstall app '${appName}': ${error.message}`);
      throw error;
    }
  }
}