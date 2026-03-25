import { test, expect } from '../src/fixtures';

test.describe.configure({ mode: 'serial' });

test.describe('ServiceNow ITSM - E2E Tests', () => {
  test('should verify ServiceNow ITSM Helper workflow actions are available in workflow builder', async ({ workflowsPage }) => {
    test.setTimeout(180000); // 3 minutes
    await workflowsPage.navigateToWorkflows();
    await workflowsPage.createNewWorkflow();

    // Select "On demand" trigger
    const onDemandTrigger = workflowsPage.page.getByText('On demand').first();
    await onDemandTrigger.click();

    const nextButton = workflowsPage.page.getByRole('button', { name: 'Next' });
    await nextButton.click();

    await workflowsPage.page.waitForLoadState('networkidle');
    await workflowsPage.page.getByText('Add next').waitFor({ state: 'visible', timeout: 10000 });

    // Click "Add action" button to open the action selection dialog
    const addNextMenu = workflowsPage.page.getByTestId('add-next-menu-container');
    const addActionButton = addNextMenu.getByTestId('context-menu-seq-action-button');
    await addActionButton.click();
    await workflowsPage.page.waitForLoadState('networkidle');

    // Wait for search box to be visible
    const searchBox = workflowsPage.page.getByRole('searchbox').or(workflowsPage.page.getByPlaceholder(/search/i));
    await searchBox.waitFor({ state: 'visible', timeout: 10000 });

    // Wait for initial action list loading to complete
    const loadingMessages = workflowsPage.page.getByText('This may take a few moments');
    await loadingMessages.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});
    await workflowsPage.page.waitForLoadState('networkidle');

    // All 5 ITSM Helper actions to verify
    const expectedActions = [
      'ITSM Helper - Entities - Check if external entity exists',
      'ITSM Helper - Entities - Establish mapping',
      'ITSM Helper - Create Incident',
      'ITSM Helper - Create SIR Incident',
      'ITSM Helper - Throttle'
    ];

    for (const actionName of expectedActions) {
      // Search for the action
      await expect(searchBox).toBeEnabled({ timeout: 10000 });
      await searchBox.fill(actionName);
      await loadingMessages.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});
      await workflowsPage.page.waitForLoadState('networkidle');

      // Click the tile label, not the heading span (the heading is inside a <label>
      // that intercepts pointer events, causing unreliable clicks on the span)
      const tile = workflowsPage.page.locator('label[data-test-selector="content-item"]').filter({
        has: workflowsPage.page.locator('[data-test-selector="node-tile-heading"]', { hasText: actionName })
      });
      await tile.first().waitFor({ state: 'visible', timeout: 30000 });

      // Retry click up to 3 times - after multiple back-navigations the DOM can
      // need a moment to stabilize before clicks register
      let tabsVisible = false;
      for (let clickAttempt = 0; clickAttempt < 3 && !tabsVisible; clickAttempt++) {
        if (clickAttempt > 0) {
          await workflowsPage.page.waitForTimeout(500);
        }
        await tile.first().click({ timeout: 10000 });
        await workflowsPage.page.waitForLoadState('networkidle');

        // Verify the action detail panel opens (all ITSM Helper actions show tabs)
        const detailTabs = workflowsPage.page.getByRole('tab');
        tabsVisible = await detailTabs.first().waitFor({ state: 'visible', timeout: 5000 })
          .then(() => true)
          .catch(() => false);
      }
      if (!tabsVisible) {
        throw new Error(`Action '${actionName}' detail panel did not open after clicking`);
      }
      console.log(`✓ Action verified: ${actionName}`);

      // Go back to action list for next action
      const backButton = workflowsPage.page.getByRole('button', { name: 'Back' }).or(
        workflowsPage.page.getByLabel('Back')
      );
      if (await backButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await backButton.click();
        await workflowsPage.page.waitForLoadState('networkidle');
        await loadingMessages.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});
        await workflowsPage.page.waitForLoadState('networkidle');
      }
    }

    console.log('All ServiceNow ITSM Helper actions verified successfully');
  });
});
