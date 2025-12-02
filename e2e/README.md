# E2E Tests

End-to-end tests for the ServiceNow ITSM Foundry app using Playwright.

## Tests Included

- **Workflow Verification**: Verifies that the app's workflows are deployed and accessible in Fusion SOAR

## About This App

This app synchronizes ServiceNow Configuration Management Database (CMDB) configuration rules with Identity Protection policies. It includes two workflows for synchronizing policy rules and managing checkpoints. The E2E tests verify that both workflows are properly deployed and discoverable.

## Setup

```bash
npm ci
npx playwright install chromium
cp .env.sample .env
# Edit .env with your credentials
```

## Run Tests

```bash
npm test              # All tests
npm run test:debug    # Debug mode
npm run test:ui       # Interactive UI
```

## Environment Variables

```env
# CrowdStrike Falcon Configuration
APP_NAME=foundry-sample-servicenow-itsm
FALCON_BASE_URL=https://falcon.us-2.crowdstrike.com
FALCON_USERNAME=your-username
FALCON_PASSWORD=your-password
FALCON_AUTH_SECRET=your-mfa-secret

# ServiceNow Instance Configuration (Required for App Installation)
SERVICENOW_INSTANCE_URL=https://dev123456.service-now.com
SERVICENOW_USERNAME=your-servicenow-username
SERVICENOW_PASSWORD=your-servicenow-password
```

**Important:**
- The `APP_NAME` must exactly match the app name as deployed in Falcon.
- ServiceNow credentials are required for app installation as the app needs to configure API integration with a ServiceNow instance.
- You'll need a valid ServiceNow developer instance. Get one free at [developer.servicenow.com](https://developer.servicenow.com).

## Test Flow

1. **Setup**: Authenticates and installs the app
2. **Workflow Verification**:
   - Searches for both workflows
   - Verifies both workflows are discoverable in Fusion SOAR
3. **Teardown**: Uninstalls the app

## CI/CD

Tests run automatically in GitHub Actions on push/PR to main. The workflow deploys the app, runs tests, and cleans up.
