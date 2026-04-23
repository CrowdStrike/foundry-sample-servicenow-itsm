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

      // Select Basic Authentication by name — falcon-select Ember component
      // only responds to keyboard navigation, not .click() on option elements
      const authTypeButton = page.getByRole('button', { name: /Authentication Type/i });
      await authTypeButton.click();
      const listbox = page.getByRole('listbox', { name: /Authentication Type/i });
      await listbox.waitFor({ state: 'visible', timeout: 5000 });
      const options = listbox.getByRole('option');
      const count = await options.count();
      await listbox.press('ArrowUp');
      for (let i = 0; i < count; i++) {
        if ((await options.nth(i).textContent())?.trim() === 'Basic Authentication') break;
        await listbox.press('ArrowDown');
      }
      await listbox.press('Enter');

      await page.waitForLoadState('domcontentloaded');

      await page.getByLabel('Username').fill(username);
      await page.getByLabel('Password').fill(password);
    },
  });
});
