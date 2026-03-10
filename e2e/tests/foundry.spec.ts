import { test, expect } from '../src/fixtures';

test.describe.configure({ mode: 'serial' });

test.describe('ServiceNow ITSM - E2E Tests', () => {
  test('should verify ServiceNow ITSM Helper workflow actions are available in workflow builder', async ({ workflowsPage }) => {
    // This app provides helper functions for ServiceNow ITSM integration
    // We verify all 5 ITSM Helper actions are available in the workflow builder
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

      // Wait for search results to load
      await loadingMessages.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});
      await workflowsPage.page.waitForLoadState('networkidle');

      // Expand "Other (Custom, Foundry, etc.)" section if it exists
      const otherSection = workflowsPage.page.getByText('Other (Custom, Foundry, etc.)');
      if (await otherSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await otherSection.click();

        // Wait for section's internal loading to complete
        await loadingMessages.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});
        await workflowsPage.page.waitForLoadState('networkidle');
      }

      // Use the node-tile-heading test selector for precise matching
      // Each action card has a heading span with data-test-selector="node-tile-heading"
      const actionHeadings = workflowsPage.page.locator('[data-test-selector="node-tile-heading"]').filter({ hasText: actionName });
      await actionHeadings.first().waitFor({ state: 'visible', timeout: 30000 });

      // Filter to exact matches only (avoid "Create Incident" matching "Create SIR Incident")
      const allHeadings = await actionHeadings.all();
      const exactMatches = [];
      for (const heading of allHeadings) {
        const text = await heading.textContent();
        if (text?.trim() === actionName) {
          exactMatches.push(heading);
        }
      }

      const candidates = exactMatches.length > 0 ? exactMatches : allHeadings;
      console.log(`Found ${candidates.length} instance(s) of '${actionName}' - trying each until one works...`);

      if (candidates.length === 0) {
        throw new Error(`Action '${actionName}' not found in search results`);
      }

      let actionVerified = false;

      // Try each instance until we find one that's not stale
      for (let i = 0; i < candidates.length; i++) {
        console.log(`  Trying instance ${i + 1}/${candidates.length}...`);

        try {
          await candidates[i].click({ timeout: 10000 });
          await workflowsPage.page.waitForLoadState('networkidle');

          try {
            const configureTab = workflowsPage.page.getByRole('tab', { name: 'Configure' });
            await configureTab.waitFor({ state: 'visible', timeout: 15000 });
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
              await loadingMessages.first().waitFor({ state: 'hidden', timeout: 60000 }).catch(() => {});
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
        throw new Error(`Failed to verify action '${actionName}' - all ${candidates.length} instance(s) appear to be stale or invalid`);
      }
    }

    console.log('All ServiceNow ITSM Helper actions verified successfully');
  });
});
