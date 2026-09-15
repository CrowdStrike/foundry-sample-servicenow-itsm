import { test, expect } from '../src/fixtures';

test.describe.configure({ mode: 'serial' });

test.describe('ServiceNow ITSM - E2E Tests', () => {
  test('should verify ServiceNow ITSM Helper workflow actions are available in workflow builder', async ({ page, workflowsPage }) => {
    test.setTimeout(180000);
    await workflowsPage.navigateToWorkflows();

    // Create new workflow - click the card then Next
    const createButton = page.getByRole('link', { name: 'Create workflow' })
      .or(page.getByRole('link', { name: 'Create a workflow' }));
    await createButton.click();

    // Select "Create workflow from scratch" card in the modal
    await page.getByText('Create workflow from scratch').click();
    // Click Next to proceed to trigger selection
    await page.getByRole('button', { name: 'Next' }).click();

    // Select "On demand" trigger
    const onDemandTrigger = page.getByText('On demand').first();
    await onDemandTrigger.waitFor({ state: 'visible', timeout: 15000 });
    await onDemandTrigger.click();

    const nextButton = page.getByRole('button', { name: 'Next' });
    await nextButton.click();

    await page.getByText('Add next').waitFor({ state: 'visible', timeout: 15000 });

    // Click "Add action" button to open the action selection dialog
    const addNextMenu = page.getByTestId('add-next-menu-container');
    const addActionButton = addNextMenu.getByTestId('context-menu-seq-action-button');
    await addActionButton.click();

    // Wait for search box to be visible
    const searchBox = page.getByRole('searchbox').or(page.getByPlaceholder(/search/i));
    await searchBox.waitFor({ state: 'visible', timeout: 10000 });

    // Wait for initial action list loading to complete
    const loadingMessages = page.getByText('This may take a few moments');
    await loadingMessages.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});

    // All 5 ITSM Helper actions to verify
    const expectedActions = [
      'ITSM Helper - Entities - Check if external entity exists',
      'ITSM Helper - Entities - Establish mapping',
      'ITSM Helper - Create Incident',
      'ITSM Helper - Create SIR Incident',
      'ITSM Helper - Throttle'
    ];

    for (const actionName of expectedActions) {
      await expect(searchBox).toBeEnabled({ timeout: 10000 });
      await searchBox.fill(actionName);
      await loadingMessages.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});

      const tile = page.locator('label[data-test-selector="content-item"]').filter({
        has: page.locator('[data-test-selector="node-tile-heading"]', { hasText: actionName })
      });
      await tile.first().waitFor({ state: 'visible', timeout: 30000 });

      let tabsVisible = false;
      for (let clickAttempt = 0; clickAttempt < 3 && !tabsVisible; clickAttempt++) {
        await tile.first().click({ timeout: 10000 });

        const detailTabs = page.getByRole('tab');
        tabsVisible = await detailTabs.first().waitFor({ state: 'visible', timeout: 5000 })
          .then(() => true)
          .catch(() => false);
      }
      if (!tabsVisible) {
        throw new Error(`Action '${actionName}' detail panel did not open after clicking`);
      }
      console.log(`✓ Action verified: ${actionName}`);

      const backButton = page.getByRole('button', { name: 'Back' }).or(
        page.getByLabel('Back')
      );
      if (await backButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await backButton.click();
        await loadingMessages.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});
      }
    }

    console.log('All ServiceNow ITSM Helper actions verified successfully');
  });
});
