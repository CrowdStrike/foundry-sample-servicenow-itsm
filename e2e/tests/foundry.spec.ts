import { test, expect } from '../src/fixtures';

test.describe.configure({ mode: 'serial' });

test.describe('ServiceNow ITSM - E2E Tests', () => {
  test('should verify ServiceNow ITSM Helper workflow actions are available in workflow builder', async ({ workflowsPage }) => {
    // This app provides helper functions for ServiceNow ITSM integration
    // We verify all 5 ITSM Helper actions are available in the workflow builder
    test.setTimeout(120000); // 2 minutes - buffer for stale action handling
    await workflowsPage.navigateToWorkflows();
    await workflowsPage.createNewWorkflow();

    // Select "On demand" trigger
    const onDemandTrigger = workflowsPage.page.getByText('On demand').first();
    await onDemandTrigger.click();

    const nextButton = workflowsPage.page.getByRole('button', { name: 'Next' });
    await nextButton.click();

    await workflowsPage.page.waitForLoadState('networkidle');
    await workflowsPage.page.getByText('Add next').waitFor({ state: 'visible', timeout: 10000 });

    // Click "Add action" button once to open the action selection dialog
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

    // Verify all 5 ITSM Helper actions by checking their Configure sections
    const expectedActions = [
      'ITSM Helper - Entities - Check if external entity exists',
      'ITSM Helper - Entities - Establish mapping',
      'ITSM Helper - Create Incident',
      'ITSM Helper - Create SIR Incident',
      'ITSM Helper - Throttle'
    ];

    for (const actionName of expectedActions) {
      // Search for the specific action
      await expect(searchBox).toBeEnabled({ timeout: 10000 });
      await searchBox.fill(actionName);

      // Wait for search results to load by waiting for network idle first
      await workflowsPage.page.waitForLoadState('networkidle');

      // Wait for ALL loading spinners to disappear completely
      await workflowsPage.page.waitForSelector('text="This may take a few moments"', { state: 'hidden', timeout: 90000 });

      // Wait for network to be idle again after search results load
      await workflowsPage.page.waitForLoadState('networkidle');

      // Look for "Top results" or similar text indicating search completed successfully
      const topResults = workflowsPage.page.getByText('Top results').or(
        workflowsPage.page.getByText(/\d+ actions/)
      );
      await expect(topResults.first()).toBeVisible({ timeout: 30000 });

      // ALWAYS expand "Other (Custom, Foundry, etc.)" section since ITSM actions are there
      const otherSection = workflowsPage.page.getByText('Other (Custom, Foundry, etc.)');
      await expect(otherSection).toBeVisible({ timeout: 10000 });
      await otherSection.click();

      // Wait for the section to expand and load its contents
      await workflowsPage.page.waitForLoadState('networkidle');

      // Wait for any loading in the expanded section to complete
      await workflowsPage.page.waitForSelector('text="This may take a few moments"', {
        state: 'hidden',
        timeout: 30000
      }).catch(() => {}); // Don't fail if no loading messages appear

      // Find all instances of this action (may include stale ones from previous installs)
      // Wait for at least one instance to appear before getting all instances
      const actionLocator = workflowsPage.page.getByText(actionName, { exact: false });
      await actionLocator.first().waitFor({ state: 'attached', timeout: 30000 });

      const actionElements = await actionLocator.all();

      if (actionElements.length === 0) {
        throw new Error(`Action '${actionName}' not found in search results`);
      }

      console.log(`Found ${actionElements.length} instance(s) of '${actionName}' - trying each until one works...`);

      let actionVerified = false;

      // Try each instance until we find one that's not stale
      for (let i = 0; i < actionElements.length; i++) {
        console.log(`  Trying instance ${i + 1}/${actionElements.length}...`);

        try {
          // Click on the action
          await actionElements[i].click();
          await workflowsPage.page.waitForLoadState('networkidle');

          // Wait for the details panel to load and check if configuration is present
          // Stale actions won't show the "Configure" heading
          try {
            // Try multiple ways to detect a valid action configuration
            const configureHeading = workflowsPage.page.getByRole('heading', { name: 'Configure' });
            const configureTab = workflowsPage.page.getByRole('tab', { name: 'Configure' });
            const configIndicator = configureHeading.or(configureTab);

            await configIndicator.waitFor({ state: 'visible', timeout: 20000 });
            console.log(`✓ Action verified: ${actionName} - Configure section is present`);
            actionVerified = true;

            // Close the dialog to prepare for next action
            const backButton = workflowsPage.page.getByRole('button', { name: 'Back' }).or(
              workflowsPage.page.getByLabel('Back')
            );
            if (await backButton.isVisible({ timeout: 1000 }).catch(() => false)) {
              await backButton.click();
              await workflowsPage.page.waitForLoadState('networkidle');

              // Wait for action list to reload after going back
              const allLoadingMessages = workflowsPage.page.locator('text="This may take a few moments"');
              await expect(allLoadingMessages).toHaveCount(0, { timeout: 60000 });
              await workflowsPage.page.waitForLoadState('networkidle');
            }

            break;
          } catch (error) {
            const errorMsg = error.message || 'Unknown error';
            console.log(`  Instance ${i + 1} failed: ${errorMsg}`);

            // Go back to try next instance
            const backButton = workflowsPage.page.getByRole('button', { name: 'Back' }).or(
              workflowsPage.page.getByLabel('Back')
            );
            if (await backButton.isVisible({ timeout: 1000 }).catch(() => false)) {
              await backButton.click();
              await workflowsPage.page.waitForLoadState('networkidle');
            }
          }
        } catch (error) {
          console.log(`  Instance ${i + 1} failed: ${error.message}, trying next...`);
        }
      }

      if (!actionVerified) {
        throw new Error(`Failed to verify action '${actionName}' - all ${actionElements.length} instance(s) appear to be stale or invalid`);
      }
    }

    console.log('All ServiceNow ITSM Helper actions verified successfully');
  });
});
