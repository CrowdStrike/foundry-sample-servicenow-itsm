#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Deploying app with Foundry CLI...${NC}"
foundry apps deploy --change-type=major --change-log="E2E test deployment"
echo -e "${GREEN}✓ App deployment initiated${NC}"

echo -e "${YELLOW}Waiting for deployment to complete...${NC}"
timeout=300  # 5 minute timeout
elapsed=0

while [ $elapsed -lt $timeout ]; do
  if foundry apps list-deployments | grep -i "successful"; then
    echo -e "${GREEN}✓ Deployment successful${NC}"
    echo -e "${YELLOW}Releasing app...${NC}"
    foundry apps release --change-type=major --notes="E2E test release"
    echo -e "${GREEN}✓ App released${NC}"

    # Brief wait for release to complete
    echo -e "${YELLOW}Allowing brief time for release to complete...${NC}"
    sleep 15
    break
  fi

  if foundry apps list-deployments | grep -i "failed"; then
    echo -e "\033[0;31m✗ Deployment failed${NC}"
    exit 1
  fi

  sleep 5
  elapsed=$((elapsed + 5))
done

if [ $elapsed -ge $timeout ]; then
  echo -e "\033[0;31m✗ Deployment timeout after ${timeout} seconds${NC}"
  exit 1
fi

echo -e "${YELLOW}Running E2E tests...${NC}"
cd e2e && npm test
TEST_EXIT_CODE=$?

echo -e "${YELLOW}Restoring workflow files...${NC}"
cd ..
git checkout workflows/*.yml
echo -e "${GREEN}✓ Workflow files restored${NC}"

if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo -e "${GREEN}✓ E2E tests completed successfully!${NC}"
else
  echo -e "\033[0;31m✗ E2E tests failed${NC}"
  exit $TEST_EXIT_CODE
fi
