package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"itsmhelper/internal/storage"

	fdk "github.com/CrowdStrike/foundry-fn-go"
	"github.com/CrowdStrike/foundry-fn-go/fdktest"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/crowdstrike/gofalcon/falcon/client/api_integrations"
	"github.com/crowdstrike/gofalcon/falcon/client/custom_storage"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/suite"
)

// HandlerTestSuite defines the test suite for handler functionality
type HandlerTestSuite struct {
	suite.Suite
	mockStorage         *storage.MockStorageService
	mockAPIIntegrations *MockAPIIntegrationsService
	logger              *slog.Logger
}

// SetupTest runs before each test in the suite
func (s *HandlerTestSuite) SetupTest() {
	s.mockStorage = &storage.MockStorageService{}
	s.mockAPIIntegrations = &MockAPIIntegrationsService{}
	s.logger = fdktest.NewLogger(s.T())
}

// TestHandleCheckIfExtEntityExists tests the Handler.HandleCheckIfExtEntityExists method
func (s *HandlerTestSuite) TestHandleCheckIfExtEntityExists() {
	// Define test cases
	tests := []struct {
		name            string
		request         fdk.RequestOf[CheckIfExtExistsReq]
		setupMockStore  func(mockStorage *storage.MockStorageService)
		setupMockClient func() (*client.CrowdStrikeAPISpecification, string, error)
		wantCode        int
		wantBody        map[string]interface{}
		wantErrors      []fdk.APIError
	}{
		{
			name: "Entity doesn't exist",
			request: fdk.RequestOf[CheckIfExtExistsReq]{
				Body: CheckIfExtExistsReq{
					InternalEntityID: "entity123",
					ExternalSystemID: "servicenow",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					// Verify that the ObjectKey is correctly formed using the external system ID and internal entity ID
					expectedKey, err := storage.CreateTrackedEntityKey("servicenow", "entity123")
					if err != nil {
						s.T().Errorf("Unexpected error creating tracked entity key: %v", err)
						return nil, err
					}
					s.Equal(expectedKey, params.ObjectKey, "ObjectKey should match expected value")
					return nil, fmt.Errorf("status 404")
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 200,
			wantBody: map[string]interface{}{
				"exists": false,
			},
		},
		{
			name: "Entity exists with ServiceNow Incident external system ID",
			request: fdk.RequestOf[CheckIfExtExistsReq]{
				Body: CheckIfExtExistsReq{
					InternalEntityID: "entity123",
					ExternalSystemID: ExternalSystemIDServiceNowIncident,
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					record := storage.ExternalEntityRecord{
						InternalEntityID: "entity123",
						ExternalEntityID: "ext123",
						ExternalSystemID: ExternalSystemIDServiceNowIncident,
					}
					json.NewEncoder(writer).Encode(record)
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 200,
			wantBody: map[string]interface{}{
				"exists":        true,
				"ext_id":        "ext123",
				"ext_system_id": ExternalSystemIDServiceNowIncident,
			},
		},
		{
			name: "Entity exists with ServiceNow SIR Incident external system ID",
			request: fdk.RequestOf[CheckIfExtExistsReq]{
				Body: CheckIfExtExistsReq{
					InternalEntityID: "entity123",
					ExternalSystemID: ExternalSystemIDServiceNowSIRIncident,
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					record := storage.ExternalEntityRecord{
						InternalEntityID: "entity123",
						ExternalEntityID: "ext123",
						ExternalSystemID: ExternalSystemIDServiceNowSIRIncident,
					}
					json.NewEncoder(writer).Encode(record)
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 200,
			wantBody: map[string]interface{}{
				"exists":        true,
				"ext_id":        "ext123",
				"ext_system_id": ExternalSystemIDServiceNowSIRIncident,
			},
		},
		{
			name: "Entity exists but with different external system ID",
			request: fdk.RequestOf[CheckIfExtExistsReq]{
				Body: CheckIfExtExistsReq{
					InternalEntityID: "entity123",
					ExternalSystemID: ExternalSystemIDServiceNowIncident,
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					record := storage.ExternalEntityRecord{
						InternalEntityID: "entity123",
						ExternalEntityID: "ext123",
						ExternalSystemID: ExternalSystemIDServiceNowSIRIncident, // Different from requested
					}
					json.NewEncoder(writer).Encode(record)
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 200,
			wantBody: map[string]interface{}{
				"exists": false, // Should return false because external system IDs don't match
			},
		},
		{
			name: "Falcon client creation error",
			request: fdk.RequestOf[CheckIfExtExistsReq]{
				Body: CheckIfExtExistsReq{
					InternalEntityID: "entity123",
					ExternalSystemID: "servicenow",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// No setup needed as client creation will fail
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				return nil, "", fmt.Errorf("client creation error")
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "error creating Falcon client: client creation error",
				},
			},
		},
		{
			name: "Storage service error",
			request: fdk.RequestOf[CheckIfExtExistsReq]{
				Body: CheckIfExtExistsReq{
					InternalEntityID: "entity123",
					ExternalSystemID: "servicenow",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("connection error")
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to check if ticket exists: failed to check if external entity exists: connection error",
				},
			},
		},
		{
			name: "Invalid JSON response",
			request: fdk.RequestOf[CheckIfExtExistsReq]{
				Body: CheckIfExtExistsReq{
					InternalEntityID: "entity123",
					ExternalSystemID: "servicenow",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					writer.Write([]byte("invalid json"))
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to check if ticket exists: failed to unmarshal external entity record: invalid character 'i' looking for beginning of value",
				},
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Reset mock storage for each test
			s.SetupTest()

			tc.setupMockStore(s.mockStorage)

			mockClientBuilder := func(token string, logger *slog.Logger) (*client.CrowdStrikeAPISpecification, string, error) {
				client, cloud, err := tc.setupMockClient()
				if client != nil && err == nil {
					client.CustomStorage = s.mockStorage
				}
				return client, cloud, err
			}

			handler := &Handler{
				logger:           s.logger,
				falconClientFunc: mockClientBuilder,
			}

			response := handler.HandleCheckIfExtEntityExists(context.Background(), tc.request)
			s.Equal(tc.wantCode, response.Code, "Response code should match expected value")

			// Check for error responses (using wantErrors)
			if len(tc.wantErrors) > 0 {
				// For error cases, we expect the Body to be nil and the error to be in the Errors field
				s.Nil(response.Body, "Response body should be nil for error responses")
				s.NotEmpty(response.Errors, "Response errors should not be empty for error responses")
				s.Len(response.Errors, len(tc.wantErrors), "Response should have the expected number of errors")

				for i, wantErr := range tc.wantErrors {
					s.Equal(wantErr.Code, response.Errors[i].Code, "Error code should match expected value")
					s.Equal(wantErr.Message, response.Errors[i].Message, "Error message should match expected value")
				}
				return
			}

			jsonBytes, err := json.Marshal(response.Body)
			s.NoError(err, "Failed to marshal JSON body")

			var actual map[string]interface{}
			err = json.Unmarshal(jsonBytes, &actual)
			s.NoError(err, "Failed to unmarshal JSON body")

			for k, v := range tc.wantBody {
				actualVal, exists := actual[k]
				s.True(exists, "Expected key %q not found in response", k)
				s.Equal(v, actualVal, "For key %q, expected value should match actual value", k)
			}
		})
	}
}

// TestHandleCreateEntityMapping tests the Handler.HandleCreateEntityMapping method
func (s *HandlerTestSuite) TestHandleCreateEntityMapping() {
	// Define test cases
	tests := []struct {
		name            string
		request         fdk.RequestOf[CreateEntityMappingReq]
		setupMockStore  func(mockStorage *storage.MockStorageService)
		setupMockClient func() (*client.CrowdStrikeAPISpecification, string, error)
		wantCode        int
		wantBody        map[string]interface{}
		wantErrors      []fdk.APIError
	}{
		{
			name: "Successful entity mapping creation",
			request: fdk.RequestOf[CreateEntityMappingReq]{
				Body: CreateEntityMappingReq{
					InternalEntityID: "internal123",
					ExternalEntityID: "external123",
					ExternalSystemID: "servicenow",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 201,
			wantBody: map[string]interface{}{
				"internal_entity_id": "internal123",
				"external_entity_id": "external123",
				"external_system_id": "servicenow",
			},
		},
		{
			name: "Falcon client creation error",
			request: fdk.RequestOf[CreateEntityMappingReq]{
				Body: CreateEntityMappingReq{
					InternalEntityID: "internal123",
					ExternalEntityID: "external123",
					ExternalSystemID: "servicenow",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// No setup needed as client creation will fail
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				return nil, "", fmt.Errorf("client creation error")
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "error creating Falcon client: client creation error",
				},
			},
		},
		{
			name: "Storage service error",
			request: fdk.RequestOf[CreateEntityMappingReq]{
				Body: CreateEntityMappingReq{
					InternalEntityID: "internal123",
					ExternalEntityID: "external123",
					ExternalSystemID: "servicenow",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return nil, fmt.Errorf("storage error")
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "storage error",
				},
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Reset mock storage for each test
			s.SetupTest()

			tc.setupMockStore(s.mockStorage)

			mockClientBuilder := func(token string, logger *slog.Logger) (*client.CrowdStrikeAPISpecification, string, error) {
				client, cloud, err := tc.setupMockClient()
				if client != nil && err == nil {
					client.CustomStorage = s.mockStorage
				}
				return client, cloud, err
			}

			handler := &Handler{
				logger:           s.logger,
				falconClientFunc: mockClientBuilder,
			}

			response := handler.HandleCreateEntityMapping(context.Background(), tc.request)
			s.Equal(tc.wantCode, response.Code, "Response code should match expected value")

			// Marshal the body to JSON
			jsonBytes, err := json.Marshal(response.Body)
			s.NoError(err, "Failed to marshal JSON body")

			// Unmarshal into a map for comparison
			var actual map[string]interface{}
			err = json.Unmarshal(jsonBytes, &actual)
			s.NoError(err, "Failed to unmarshal JSON body")

			// Compare expected vs actual
			for k, v := range tc.wantBody {
				actualVal, exists := actual[k]
				s.True(exists, "Expected key %q not found in response", k)
				s.Equal(v, actualVal, "For key %q, expected value should match actual value", k)
			}
		})
	}
}

// TestHandleThrottle tests the Handler.HandleThrottle method
func (s *HandlerTestSuite) TestHandleThrottle() {
	// Define test cases
	tests := []struct {
		name            string
		request         fdk.RequestOf[ThrottleFunctionRequest]
		setupMockStore  func(mockStorage *storage.MockStorageService)
		setupMockClient func() (*client.CrowdStrikeAPISpecification, string, error)
		wantCode        int
		wantBody        map[string]interface{}
		wantErrors      []fdk.APIError
	}{
		{
			name: "Throttling allowed (not a duplicate)",
			request: fdk.RequestOf[ThrottleFunctionRequest]{
				Body: ThrottleFunctionRequest{
					InternalEntityID: "entity123",
					DedupObjType:     "alert",
					DedupObjID:       "alert123",
					TimeBucket:       "forever",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 200,
			wantBody: map[string]interface{}{
				"allowed": true,
			},
		},
		{
			name: "Throttling not allowed (is a duplicate)",
			request: fdk.RequestOf[ThrottleFunctionRequest]{
				Body: ThrottleFunctionRequest{
					InternalEntityID: "entity123",
					DedupObjType:     "alert",
					DedupObjID:       "alert123",
					TimeBucket:       "forever",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					record := storage.DedupStoreRecord{
						TimeBucket: storage.TimeBucketForever,
					}
					json.NewEncoder(writer).Encode(record)
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 200,
			wantBody: map[string]interface{}{
				"allowed": false,
			},
		},
		{
			name: "Falcon client creation error",
			request: fdk.RequestOf[ThrottleFunctionRequest]{
				Body: ThrottleFunctionRequest{
					InternalEntityID: "entity123",
					DedupObjType:     "alert",
					DedupObjID:       "alert123",
					TimeBucket:       "forever",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// No setup needed as client creation will fail
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				return nil, "", fmt.Errorf("client creation error")
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "error creating Falcon client: client creation error",
				},
			},
		},
		{
			name: "Invalid time bucket",
			request: fdk.RequestOf[ThrottleFunctionRequest]{
				Body: ThrottleFunctionRequest{
					InternalEntityID: "entity123",
					DedupObjType:     "alert",
					DedupObjID:       "alert123",
					TimeBucket:       "invalid_bucket",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// No specific setup needed as the validation will fail before storage is used
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "unsupported time bucket value: invalid_bucket (must be one of: forever, 5 minutes, 30 minutes)",
				},
			},
		},
		{
			name: "Storage service error",
			request: fdk.RequestOf[ThrottleFunctionRequest]{
				Body: ThrottleFunctionRequest{
					InternalEntityID: "entity123",
					DedupObjType:     "alert",
					DedupObjID:       "alert123",
					TimeBucket:       "forever",
				},
				AccessToken: "test-token",
			},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("connection error")
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to check dedup record: connection error",
				},
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Reset mock storage for each test
			s.SetupTest()

			tc.setupMockStore(s.mockStorage)

			mockClientBuilder := func(token string, logger *slog.Logger) (*client.CrowdStrikeAPISpecification, string, error) {
				client, cloud, err := tc.setupMockClient()
				if client != nil && err == nil {
					client.CustomStorage = s.mockStorage
				}
				return client, cloud, err
			}

			// Create handler with mock client builder
			handler := &Handler{
				logger:           s.logger,
				falconClientFunc: mockClientBuilder,
			}

			// Call function
			response := handler.HandleThrottle(context.Background(), tc.request)

			// Check status code
			s.Equal(tc.wantCode, response.Code, "Response code should match expected value")

			// Check for error responses (using wantErrors)
			if len(tc.wantErrors) > 0 {
				// For error cases, we expect the Body to be nil and the error to be in the Errors field
				s.Nil(response.Body, "Response body should be nil for error responses")
				s.NotEmpty(response.Errors, "Response errors should not be empty for error responses")
				s.Len(response.Errors, len(tc.wantErrors), "Response should have the expected number of errors")

				for i, wantErr := range tc.wantErrors {
					s.Equal(wantErr.Code, response.Errors[i].Code, "Error code should match expected value")
					s.Equal(wantErr.Message, response.Errors[i].Message, "Error message should match expected value")
				}
				return
			}

			// Marshal the body to JSON
			jsonBytes, err := json.Marshal(response.Body)
			s.NoError(err, "Failed to marshal JSON body")

			// Unmarshal into a map for comparison
			var actual map[string]interface{}
			err = json.Unmarshal(jsonBytes, &actual)
			s.NoError(err, "Failed to unmarshal JSON body")

			// Compare expected vs actual
			for k, v := range tc.wantBody {
				actualVal, exists := actual[k]
				s.True(exists, "Expected key %q not found in response", k)
				s.Equal(v, actualVal, "For key %q, expected value should match actual value", k)
			}
		})
	}
}

// MockAPIIntegrationsService implements the API Integrations service for testing
type MockAPIIntegrationsService struct {
	ExecuteCommandFunc func(*api_integrations.ExecuteCommandParams, ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error)
}

func (m *MockAPIIntegrationsService) ExecuteCommandProxy(params *api_integrations.ExecuteCommandProxyParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandProxyOK, error) {
	panic("not implemented")
}

func (m *MockAPIIntegrationsService) GetCombinedPluginConfigs(params *api_integrations.GetCombinedPluginConfigsParams, opts ...api_integrations.ClientOption) (*api_integrations.GetCombinedPluginConfigsOK, error) {
	panic("not implemented")
}

// ExecuteCommand implements the ExecuteCommand method for the mock
func (m *MockAPIIntegrationsService) ExecuteCommand(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
	if m.ExecuteCommandFunc != nil {
		return m.ExecuteCommandFunc(params, opts...)
	}
	return nil, nil
}

// SetTransport implements the SetTransport method for the mock
func (m *MockAPIIntegrationsService) SetTransport(transport runtime.ClientTransport) {
	// No-op for the mock
}

// TestHandleCreateIncident tests the Handler.HandleCreateIncident method
func (s *HandlerTestSuite) TestHandleCreateIncident() {
	// Define test cases
	tests := []struct {
		name                     string
		request                  fdk.RequestOf[CreateIncidentRequest]
		workflowCtx              fdk.WorkflowCtx
		setupMockStore           func(mockStorage *storage.MockStorageService)
		setupMockAPIIntegrations func(mockAPIIntegrations *MockAPIIntegrationsService)
		setupMockClient          func() (*client.CrowdStrikeAPISpecification, string, error)
		wantCode                 int
		wantBody                 map[string]interface{}
		wantErrors               []fdk.APIError
	}{
		{
			name: "Existing ticket found",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					ConfigID:         "config123",
					EntityID:         "entity123",
					ShortDescription: "Test incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					record := storage.ExternalEntityRecord{
						InternalEntityID: "entity123",
						ExternalEntityID: "ticket123",
						ExternalSystemID: ExternalSystemIDServiceNowIncident,
					}
					json.NewEncoder(writer).Encode(record)
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				// Even though API shouldn't be called, we need to set up a mock to avoid nil pointer dereference
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{
								{
									ResponseBody: map[string]interface{}{
										"result": map[string]interface{}{
											"sys_id":         "ticket123",
											"sys_class_name": "sn_si_incident",
										},
									},
								},
							},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 200,
			wantBody: map[string]interface{}{
				"exists":      true,
				"ticket_id":   "ticket123",
				"ticket_type": "incident",
			},
		},
		{
			name: "Successful new ticket creation",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					ConfigID:         "config123",
					EntityID:         "entity123",
					ShortDescription: "Test incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// First call - check if ticket exists
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				// Second call - store mapping
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Create a realistic mock response with ticket details based on actual ServiceNow response
					result := map[string]interface{}{
						"sys_id":            "c2a8a7e5db14301094ed6bfa4b9619d3",
						"number":            "INC0010005",
						"short_description": "User cannot access email",
						"description":       "User reports being unable to log into their email client since this morning",
						"category":          "software",
						"impact":            "2",
						"urgency":           "2",
						"priority":          "2",
						"state":             "1",
						"opened_at":         "2025-04-28 14:45:22",
						"caller_id": map[string]interface{}{
							"link":  "https://instance.service-now.com/api/now/table/sys_user/5137153cc611227c000bbd1bd8cd2005",
							"value": "5137153cc611227c000bbd1bd8cd2005",
						},
						"assignment_group": map[string]interface{}{
							"link":  "https://instance.service-now.com/api/now/table/sys_user_group/8a4dde73c6112278017a6a4baf547aa7",
							"value": "8a4dde73c6112278017a6a4baf547aa7",
						},
						"sys_class_name": "incident",
					}

					resource := &models.DomainExecuteCommandResultV1{
						ResponseBody: map[string]interface{}{
							"result": result,
						},
					}

					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{resource},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 201,
			wantBody: map[string]interface{}{
				"exists":      false,
				"ticket_id":   "c2a8a7e5db14301094ed6bfa4b9619d3",
				"ticket_type": "incident",
			},
		},
		{
			name: "Successful new ticket creation with custom fields",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					ConfigID:         "config123",
					EntityID:         "entity123",
					ShortDescription: "Test incident with custom fields",
					CustomFields:     `{"u_custom_field1": "value1", "u_custom_field2": 42, "u_custom_field3": true}`,
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// First call - check if ticket exists
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				// Second call - store mapping
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Verify that custom fields are included in the request payload
					requestJSON, ok := params.Body.Resources[0].Request.JSON.(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("expected request JSON to be a map[string]interface{}")
					}

					// Check if custom fields are present in the request
					if requestJSON["u_custom_field1"] != "value1" ||
						requestJSON["u_custom_field2"] != float64(42) ||
						requestJSON["u_custom_field3"] != true {
						return nil, fmt.Errorf("custom fields not properly included in request payload")
					}

					// Create a realistic mock response with ticket details
					result := map[string]interface{}{
						"sys_id":            "c2a8a7e5db14301094ed6bfa4b9619d4",
						"number":            "INC0010006",
						"short_description": "Test incident with custom fields",
						"u_custom_field1":   "value1",
						"u_custom_field2":   42,
						"u_custom_field3":   true,
						"sys_class_name":    "incident",
					}

					resource := &models.DomainExecuteCommandResultV1{
						ResponseBody: map[string]interface{}{
							"result": result,
						},
					}

					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{resource},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 201,
			wantBody: map[string]interface{}{
				"exists":      false,
				"ticket_id":   "c2a8a7e5db14301094ed6bfa4b9619d4",
				"ticket_type": "incident",
			},
		},
		{
			name: "Falcon client creation error",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// No setup needed as client creation will fail
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				// No setup needed as client creation will fail
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				return nil, "", fmt.Errorf("client creation error")
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "error creating Falcon client: client creation error",
				},
			},
		},
		{
			name: "Error checking if ticket exists",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("connection error")
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				// No setup needed as check will fail before API is called
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to check if ticket exists: failed to check if external entity exists: connection error",
				},
			},
		},
		{
			name: "Error executing ServiceNow command - Authentication failure",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Return a realistic authentication error response based on actual ServiceNow error
					errorResponse := map[string]interface{}{
						"error": map[string]interface{}{
							"message": "User Not Authenticated",
							"detail":  "Required authentication credential is missing or invalid",
						},
						"status": "failure",
					}

					// Convert to JSON string for the error message
					errorJSON, _ := json.Marshal(errorResponse)
					return nil, fmt.Errorf("401 Unauthorized: %s", string(errorJSON))
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to execute command: 401 Unauthorized: {\"error\":{\"detail\":\"Required authentication credential is missing or invalid\",\"message\":\"User Not Authenticated\"},\"status\":\"failure\"}",
				},
			},
		},
		{
			name: "Empty response payload",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					return &api_integrations.ExecuteCommandOK{
						Payload: nil,
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to execute command - nil response",
				},
			},
		},
		{
			name: "Error storing entity mapping",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// First call - check if ticket exists
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				// Second call - store mapping (fails)
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return nil, fmt.Errorf("storage error")
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Create a realistic mock response with ticket details based on actual ServiceNow response
					result := map[string]interface{}{
						"sys_id":            "new_ticket_123",
						"number":            "INC0010005",
						"short_description": "Test incident",
						"description":       "User reports being unable to log into their email client since this morning",
						"category":          "software",
						"impact":            "2",
						"urgency":           "2",
						"priority":          "2",
						"state":             "1",
						"opened_at":         "2025-04-28 14:45:22",
						"caller_id": map[string]interface{}{
							"link":  "https://instance.service-now.com/api/now/table/sys_user/5137153cc611227c000bbd1bd8cd2005",
							"value": "5137153cc611227c000bbd1bd8cd2005",
						},
						"assignment_group": map[string]interface{}{
							"link":  "https://instance.service-now.com/api/now/table/sys_user_group/8a4dde73c6112278017a6a4baf547aa7",
							"value": "8a4dde73c6112278017a6a4baf547aa7",
						},
						"sys_class_name": "incident",
					}

					resource := &models.DomainExecuteCommandResultV1{
						ResponseBody: map[string]interface{}{
							"result": result,
						},
					}

					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{resource},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "storage error",
				},
			},
		},
		{
			name: "Successful response with error field as string",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// First call - check if ticket exists
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				// Second call - store mapping
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Create a response with both a result and an error field
					result := map[string]interface{}{
						"sys_id":            "error_ticket_123",
						"number":            "INC0010006",
						"short_description": "Test incident with error",
						"sys_class_name":    "incident",
					}

					resource := &models.DomainExecuteCommandResultV1{
						ResponseBody: map[string]interface{}{
							"result": result,
							"error":  "Business rule validation failed: Incident requires approval",
						},
					}

					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{resource},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to execute command: ServiceNow Error: Business rule validation failed: Incident requires approval",
				},
			},
		},
		{
			name: "Successful response with error field as object",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// First call - check if ticket exists
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				// Second call - store mapping
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Create a response with both a result and a complex error object
					result := map[string]interface{}{
						"sys_id":            "error_ticket_456",
						"number":            "INC0010007",
						"short_description": "Test incident with complex error",
						"sys_class_name":    "incident",
					}

					errorObj := map[string]interface{}{
						"message":    "Validation Error",
						"code":       "VAL1001",
						"field":      "priority",
						"validation": "Priority must be set for high impact incidents",
					}

					resource := &models.DomainExecuteCommandResultV1{
						ResponseBody: map[string]interface{}{
							"result": result,
							"error":  errorObj,
						},
					}

					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{resource},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to execute command: ServiceNow Error: {\"code\":\"VAL1001\",\"field\":\"priority\",\"message\":\"Validation Error\",\"validation\":\"Priority must be set for high impact incidents\"}",
				},
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Reset mock storage for each test
			s.SetupTest()

			tc.setupMockStore(s.mockStorage)
			tc.setupMockAPIIntegrations(s.mockAPIIntegrations)

			mockClientBuilder := func(token string, logger *slog.Logger) (*client.CrowdStrikeAPISpecification, string, error) {
				client, cloud, err := tc.setupMockClient()
				if client != nil && err == nil {
					client.CustomStorage = s.mockStorage
					client.APIIntegrations = s.mockAPIIntegrations
				}
				return client, cloud, err
			}

			// Create handler with mock client builder
			handler := &Handler{
				logger:           s.logger,
				falconClientFunc: mockClientBuilder,
			}

			// Call function
			response := handler.HandleCreateIncident(context.Background(), tc.request, tc.workflowCtx)

			// Check status code
			s.Equal(tc.wantCode, response.Code, "Response code should match expected value")

			// Marshal the body to JSON
			jsonBytes, err := json.Marshal(response.Body)
			s.NoError(err, "Failed to marshal JSON body")

			// Unmarshal into a map for comparison
			var actual map[string]interface{}
			err = json.Unmarshal(jsonBytes, &actual)
			s.NoError(err, "Failed to unmarshal JSON body")

			// Compare expected vs actual
			for k, v := range tc.wantBody {
				actualVal, exists := actual[k]
				s.True(exists, "Expected key %q not found in response", k)
				s.Equal(v, actualVal, "For key %q, expected value should match actual value", k)
			}
		})
	}
}

// TestHandleCreateSIRIncident tests the Handler.HandleCreateSIRIncident method
func (s *HandlerTestSuite) TestHandleCreateSIRIncident() {
	// Define test cases
	tests := []struct {
		name                     string
		request                  fdk.RequestOf[CreateIncidentRequest]
		workflowCtx              fdk.WorkflowCtx
		setupMockStore           func(mockStorage *storage.MockStorageService)
		setupMockAPIIntegrations func(mockAPIIntegrations *MockAPIIntegrationsService)
		setupMockClient          func() (*client.CrowdStrikeAPISpecification, string, error)
		wantCode                 int
		wantBody                 map[string]interface{}
		wantErrors               []fdk.APIError
	}{
		{
			name: "Existing ticket found",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					ConfigID:         "config123",
					EntityID:         "entity123",
					ShortDescription: "Test SIR incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					record := storage.ExternalEntityRecord{
						InternalEntityID: "entity123",
						ExternalEntityID: "ticket123",
						ExternalSystemID: ExternalSystemIDServiceNowSIRIncident, // Use the correct external system ID
					}
					json.NewEncoder(writer).Encode(record)
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				// Even though API shouldn't be called, we need to set up a mock to avoid nil pointer dereference
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{
								{
									ResponseBody: map[string]interface{}{
										"result": map[string]interface{}{
											"sys_id":         "ticket123",
											"sys_class_name": "sn_si_incident",
										},
									},
								},
							},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 200,
			wantBody: map[string]interface{}{
				"exists":      true,
				"ticket_id":   "ticket123",
				"ticket_type": "sn_si_incident",
			},
		},
		{
			name: "Successful new ticket creation",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					ConfigID:         "config123",
					EntityID:         "entity123",
					ShortDescription: "Test SIR incident",
					Category:         "security_incident",
					Severity:         "1",
					State:            "new",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// First call - check if ticket exists
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				// Second call - store mapping
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Verify that the correct operation ID is used
					if params.Body.Resources[0].OperationID == nil || *params.Body.Resources[0].OperationID != pluginOpIDServiceNowCreateSIRIncident {
						return nil, fmt.Errorf("expected operation ID %s, got %s", pluginOpIDServiceNowCreateSIRIncident, *params.Body.Resources[0].OperationID)
					}

					// Create a realistic mock response with SIR ticket details based on actual ServiceNow response
					result := map[string]interface{}{
						"sys_id":            "c2a8a7e5db14301094ed6bfa4b9619d4",
						"number":            "SIR0010005",
						"short_description": "Security incident: Potential data breach",
						"description":       "Investigation into potential unauthorized access to customer data",
						"category":          "security_incident",
						"impact":            "1",
						"urgency":           "1",
						"priority":          "1",
						"state":             "1",
						"opened_at":         "2025-04-28 14:45:22",
						"caller_id": map[string]interface{}{
							"link":  "https://instance.service-now.com/api/now/table/sys_user/5137153cc611227c000bbd1bd8cd2005",
							"value": "5137153cc611227c000bbd1bd8cd2005",
						},
						"assignment_group": map[string]interface{}{
							"link":  "https://instance.service-now.com/api/now/table/sys_user_group/8a4dde73c6112278017a6a4baf547aa7",
							"value": "8a4dde73c6112278017a6a4baf547aa7",
						},
						"sys_class_name": "sn_si_incident",
					}

					resource := &models.DomainExecuteCommandResultV1{
						ResponseBody: map[string]interface{}{
							"result": result,
						},
					}

					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{resource},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 201,
			wantBody: map[string]interface{}{
				"exists":      false,
				"ticket_id":   "c2a8a7e5db14301094ed6bfa4b9619d4",
				"ticket_type": "sn_si_incident",
			},
		},
		{
			name: "Successful new SIR ticket creation with custom fields",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					ConfigID:         "config123",
					EntityID:         "entity123",
					ShortDescription: "Test SIR incident with custom fields",
					Category:         "security_incident",
					Severity:         "1",
					State:            "new",
					CustomFields:     `{"u_security_category": "malware", "u_affected_systems": 3, "u_has_pii_data": true}`,
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// First call - check if ticket exists
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				// Second call - store mapping
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Verify that the correct operation ID is used
					if params.Body.Resources[0].OperationID == nil || *params.Body.Resources[0].OperationID != pluginOpIDServiceNowCreateSIRIncident {
						return nil, fmt.Errorf("expected operation ID %s, got %s", pluginOpIDServiceNowCreateSIRIncident, *params.Body.Resources[0].OperationID)
					}

					// Verify that custom fields are included in the request payload
					requestJSON, ok := params.Body.Resources[0].Request.JSON.(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("expected request JSON to be a map[string]interface{}")
					}

					// Check if custom fields are present in the request
					if requestJSON["u_security_category"] != "malware" ||
						requestJSON["u_affected_systems"] != float64(3) ||
						requestJSON["u_has_pii_data"] != true {
						return nil, fmt.Errorf("custom fields not properly included in request payload")
					}

					// Create a realistic mock response with SIR ticket details
					result := map[string]interface{}{
						"sys_id":              "c2a8a7e5db14301094ed6bfa4b9619d5",
						"number":              "SIR0010006",
						"short_description":   "Test SIR incident with custom fields",
						"category":            "security_incident",
						"u_security_category": "malware",
						"u_affected_systems":  3,
						"u_has_pii_data":      true,
						"sys_class_name":      "sn_si_incident",
					}

					resource := &models.DomainExecuteCommandResultV1{
						ResponseBody: map[string]interface{}{
							"result": result,
						},
					}

					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{resource},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 201,
			wantBody: map[string]interface{}{
				"exists":      false,
				"ticket_id":   "c2a8a7e5db14301094ed6bfa4b9619d5",
				"ticket_type": "sn_si_incident",
			},
		},
		{
			name: "Falcon client creation error",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test SIR incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// No setup needed as client creation will fail
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				// No setup needed as client creation will fail
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				return nil, "", fmt.Errorf("client creation error")
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "error creating Falcon client: client creation error",
				},
			},
		},
		{
			name: "Error checking if ticket exists",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test SIR incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("connection error")
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				// No setup needed as check will fail before API is called
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to check if ticket exists: failed to check if external entity exists: connection error",
				},
			},
		},
		{
			name: "Error executing ServiceNow command",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test SIR incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Return a realistic authentication error response based on actual ServiceNow error
					errorResponse := map[string]interface{}{
						"error": map[string]interface{}{
							"message": "User Not Authenticated",
							"detail":  "Required authentication credential is missing or invalid",
						},
						"status": "failure",
					}

					// Convert to JSON string for the error message
					errorJSON, _ := json.Marshal(errorResponse)
					return nil, fmt.Errorf("401 Unauthorized: %s", string(errorJSON))
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to execute command: 401 Unauthorized: {\"error\":{\"detail\":\"Required authentication credential is missing or invalid\",\"message\":\"User Not Authenticated\"},\"status\":\"failure\"}",
				},
			},
		},
		{
			name: "Empty response payload",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test SIR incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					return &api_integrations.ExecuteCommandOK{
						Payload: nil,
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "failed to execute command - nil response",
				},
			},
		},
		{
			name: "Error storing entity mapping",
			request: fdk.RequestOf[CreateIncidentRequest]{
				Body: CreateIncidentRequest{
					EntityID:         "entity123",
					ShortDescription: "Test SIR incident",
				},
				AccessToken: "test-token",
			},
			workflowCtx: fdk.WorkflowCtx{},
			setupMockStore: func(mockStorage *storage.MockStorageService) {
				// First call - check if ticket exists
				mockStorage.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				// Second call - store mapping (fails)
				mockStorage.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return nil, fmt.Errorf("storage error")
				}
			},
			setupMockAPIIntegrations: func(mockAPIIntegrations *MockAPIIntegrationsService) {
				mockAPIIntegrations.ExecuteCommandFunc = func(params *api_integrations.ExecuteCommandParams, opts ...api_integrations.ClientOption) (*api_integrations.ExecuteCommandOK, error) {
					// Create a realistic mock response with SIR ticket details
					result := map[string]interface{}{
						"sys_id":            "c2a8a7e5db14301094ed6bfa4b9619d4",
						"number":            "SIR0010005",
						"short_description": "Security incident: Potential data breach",
						"description":       "Investigation into potential unauthorized access to customer data",
						"category":          "security_incident",
						"impact":            "1",
						"urgency":           "1",
						"priority":          "1",
						"state":             "1",
						"opened_at":         "2025-04-28 14:45:22",
						"caller_id": map[string]interface{}{
							"link":  "https://instance.service-now.com/api/now/table/sys_user/5137153cc611227c000bbd1bd8cd2005",
							"value": "5137153cc611227c000bbd1bd8cd2005",
						},
						"assignment_group": map[string]interface{}{
							"link":  "https://instance.service-now.com/api/now/table/sys_user_group/8a4dde73c6112278017a6a4baf547aa7",
							"value": "8a4dde73c6112278017a6a4baf547aa7",
						},
						"sys_class_name": "sn_si_incident",
					}

					resource := &models.DomainExecuteCommandResultV1{
						ResponseBody: map[string]interface{}{
							"result": result,
						},
					}

					return &api_integrations.ExecuteCommandOK{
						Payload: &models.DomainExecuteCommandResultsV1{
							Resources: []*models.DomainExecuteCommandResultV1{resource},
						},
					}, nil
				}
			},
			setupMockClient: func() (*client.CrowdStrikeAPISpecification, string, error) {
				mockClient := &client.CrowdStrikeAPISpecification{}
				return mockClient, "us-1", nil
			},
			wantCode: 500,
			wantErrors: []fdk.APIError{
				{
					Code:    500,
					Message: "storage error",
				},
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Reset mock storage for each test
			s.SetupTest()

			tc.setupMockStore(s.mockStorage)
			tc.setupMockAPIIntegrations(s.mockAPIIntegrations)

			mockClientBuilder := func(token string, logger *slog.Logger) (*client.CrowdStrikeAPISpecification, string, error) {
				client, cloud, err := tc.setupMockClient()
				if client != nil && err == nil {
					client.CustomStorage = s.mockStorage
					client.APIIntegrations = s.mockAPIIntegrations
				}
				return client, cloud, err
			}

			// Create handler with mock client builder
			handler := &Handler{
				logger:           s.logger,
				falconClientFunc: mockClientBuilder,
			}

			// Call function
			response := handler.HandleCreateSIRIncident(context.Background(), tc.request, tc.workflowCtx)

			// Check status code
			s.Equal(tc.wantCode, response.Code, "Response code should match expected value")

			// Marshal the body to JSON
			jsonBytes, err := json.Marshal(response.Body)
			s.NoError(err, "Failed to marshal JSON body")

			// Unmarshal into a map for comparison
			var actual map[string]interface{}
			err = json.Unmarshal(jsonBytes, &actual)
			s.NoError(err, "Failed to unmarshal JSON body")

			// Compare expected vs actual
			for k, v := range tc.wantBody {
				actualVal, exists := actual[k]
				s.True(exists, "Expected key %q not found in response", k)
				s.Equal(v, actualVal, "For key %q, expected value should match actual value", k)
			}
		})
	}
}

// TestHandlerSuite runs the handler test suite
func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
