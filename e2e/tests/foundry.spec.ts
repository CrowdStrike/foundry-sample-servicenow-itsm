import { test, expect } from '../src/fixtures';

test.describe.configure({ mode: 'serial' });

test.describe('ServiceNow ITSM - E2E Tests', () => {
  test('should verify ServiceNow ITSM Helper workflow actions are available in workflow builder', async ({ workflowsPage }) => {
    // This app provides helper functions for ServiceNow ITSM integration
    // We verify all 5 ITSM Helper actions are available in the workflow builder
    await workflowsPage.navigateToWorkflows();
    await workflowsPage.createNewWorkflow();

    // Select "On demand" trigger
    const onDemandTrigger = workflowsPage.page.getByText('On demand').first();
    await onDemandTrigger.click();

    const nextButton = workflowsPage.page.getByRole('button', { name: 'Next' });
    await nextButton.click();

    await workflowsPage.page.waitForLoadState('networkidle');
    await workflowsPage.page.getByText('Add next').waitFor({ state: 'visible', timeout: 10000 });

    // Click "Add action" button
    const addNextMenu = workflowsPage.page.getByTestId('add-next-menu-container');
    const addActionButton = addNextMenu.getByTestId('context-menu-seq-action-button');
    await addActionButton.click();

    await workflowsPage.page.waitForLoadState('networkidle');

    // Search for ITSM Helper actions
    const searchBox = workflowsPage.page.getByRole('searchbox').or(workflowsPage.page.getByPlaceholder(/search/i));
    await searchBox.fill('ITSM Helper');

    await workflowsPage.page.getByText('This may take a few moments').waitFor({ state: 'hidden', timeout: 30000 });
    await workflowsPage.page.waitForLoadState('networkidle');

    // Verify all 5 ITSM Helper actions are visible
    const expectedActions = [
      'ITSM Helper - Entities - Check if external entity exists',
      'ITSM Helper - Entities - Establish mapping',
      'ITSM Helper - Create Incident',
      'ITSM Helper - Create SIR Incident',
      'ITSM Helper - Throttle'
    ];

    for (const actionName of expectedActions) {
      const actionElement = workflowsPage.page.getByText(actionName, { exact: false });
      await expect(actionElement).toBeVisible({ timeout: 10000 });
      console.log(`✓ Workflow action available: ${actionName}`);
    }

    console.log('All ServiceNow ITSM Helper actions verified successfully');
  });
});
