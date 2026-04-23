import { test as setup } from '@playwright/test';
import { AppCatalogPage, config } from '@crowdstrike/foundry-playwright';

setup('install app', async ({ page }) => {
  const instanceUrl = process.env.SERVICENOW_INSTANCE_URL;
  const username = process.env.SERVICENOW_USERNAME;
  const password = process.env.SERVICENOW_PASSWORD;
  if (!instanceUrl || !username || !password) {
    throw new Error('Missing required ServiceNow env vars: SERVICENOW_INSTANCE_URL, SERVICENOW_USERNAME, SERVICENOW_PASSWORD');
  }

  const catalog = new AppCatalogPage(page);
  await catalog.installApp(config.appName, {
    configureSettings: async (page) => {
      await page.getByLabel('Configuration name').fill('ServiceNow Test Instance');
      await page.getByLabel('ServiceNow Instance URL').fill(instanceUrl);

      // Change auth type from OAuth 2.0 to Basic Authentication
      const authTypeButton = page.getByRole('button', { name: /Authentication Type/i });
      await authTypeButton.click();
      const basicAuthOption = page.getByRole('option', { name: 'Basic Authentication' });
      await basicAuthOption.waitFor({ state: 'visible', timeout: 5000 });
      await basicAuthOption.click();

      await page.waitForLoadState('networkidle');

      await page.getByLabel('Username').fill(username);
      await page.getByLabel('Password').fill(password);
    },
  });
});
