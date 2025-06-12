package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/crowdstrike/gofalcon/falcon/client/custom_storage"
	"github.com/stretchr/testify/suite"
)

// StorageTestSuite defines the test suite for storage functionality
type StorageTestSuite struct {
	suite.Suite
	mockStorage *MockStorageService
	logger      *slog.Logger
}

// SetupTest runs before each test in the suite
func (s *StorageTestSuite) SetupTest() {
	s.mockStorage = &MockStorageService{}
	s.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestCreateTrackedEntityKey tests the CreateTrackedEntityKey function
func (s *StorageTestSuite) TestCreateTrackedEntityKey() {
	tests := []struct {
		name             string
		externalSystemID string
		internalEntityID string
		expected         string
		expectError      bool
		errorContains    string
	}{
		{
			name:             "Regular IDs",
			externalSystemID: "servicenow_incident",
			internalEntityID: "entity123",
			expected:         "servicenow_incident.entity123",
			expectError:      false,
		},
		{
			name:             "With special characters",
			externalSystemID: "servicenow/sir",
			internalEntityID: "entity@123",
			expected:         "servicenow_sir.entity_123",
			expectError:      false,
		},
		{
			name:             "Very long IDs",
			externalSystemID: strings.Repeat("a", 500),
			internalEntityID: strings.Repeat("b", 600),
			expected:         "",
			expectError:      true,
			errorContains:    "exceeds maximum length",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result, err := CreateTrackedEntityKey(tc.externalSystemID, tc.internalEntityID)

			if tc.expectError {
				s.Error(err)
				if tc.errorContains != "" {
					s.Contains(err.Error(), tc.errorContains)
				}
			} else {
				s.NoError(err)
				s.Equal(tc.expected, result)
			}
		})
	}
}

// TestSanitizeObjectKey tests the sanitizeObjectKey function
func (s *StorageTestSuite) TestSanitizeObjectKey() {
	tests := []struct {
		name          string
		input         string
		expected      string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid characters only",
			input:       "valid-key_123.abc",
			expected:    "valid-key_123.abc",
			expectError: false,
		},
		{
			name:        "With invalid characters",
			input:       "invalid/key:with@special#chars",
			expected:    "invalid_key_with_special_chars",
			expectError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    "",
			expectError: false,
		},
		{
			name:          "Very long key",
			input:         strings.Repeat("a", 1500),
			expected:      "",
			expectError:   true,
			errorContains: "exceeds maximum length",
		},
		{
			name:        "Unicode characters",
			input:       "key-with-unicode-😀-emoji",
			expected:    "key-with-unicode-_-emoji",
			expectError: false,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result, err := sanitizeObjectKey(tc.input)

			if tc.expectError {
				s.Error(err)
				if tc.errorContains != "" {
					s.Contains(err.Error(), tc.errorContains)
				}
			} else {
				s.NoError(err)
				s.Equal(tc.expected, result)
			}
		})
	}
}

// TestCheckThrottlingStore tests the CheckThrottlingStore function
func (s *StorageTestSuite) TestCheckThrottlingStore() {
	// Setup test cases
	tests := []struct {
		name             string
		internalEntityID string
		dedupObjType     string
		dedupObjID       string
		timeBucket       string
		mockSetup        func(*MockStorageService)
		expectedExists   bool
		expectError      bool
		errorContains    string
	}{
		{
			name:             "Invalid time bucket",
			internalEntityID: "entity123",
			dedupObjType:     "alert",
			dedupObjID:       "alert123",
			timeBucket:       "invalid",
			mockSetup:        func(client *MockStorageService) {},
			expectedExists:   false,
			expectError:      true,
			errorContains:    "unsupported time bucket value",
		},
		{
			name:             "Record doesn't exist - creates new record",
			internalEntityID: "entity123",
			dedupObjType:     "alert",
			dedupObjID:       "alert123",
			timeBucket:       string(TimeBucketFiveMin),
			mockSetup: func(client *MockStorageService) {
				client.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				client.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			expectedExists: false,
			expectError:    false,
		},
		{
			name:             "Record exists",
			internalEntityID: "entity123",
			dedupObjType:     "alert",
			dedupObjID:       "alert123",
			timeBucket:       string(TimeBucketFiveMin),
			mockSetup: func(client *MockStorageService) {
				// Mock Get to return a record
				client.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					record := DedupStoreRecord{TimeBucket: TimeBucketFiveMin}
					json.NewEncoder(writer).Encode(record)
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			expectedExists: true,
			expectError:    false,
		},
		{
			name:             "Error getting record",
			internalEntityID: "entity123",
			dedupObjType:     "alert",
			dedupObjID:       "alert123",
			timeBucket:       string(TimeBucketFiveMin),
			mockSetup: func(client *MockStorageService) {
				// Mock Get to return an error
				client.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("connection error")
				}
			},
			expectedExists: false,
			expectError:    true,
			errorContains:  "failed to check dedup record",
		},
		{
			name:             "Error creating record",
			internalEntityID: "entity123",
			dedupObjType:     "alert",
			dedupObjID:       "alert123",
			timeBucket:       string(TimeBucketFiveMin),
			mockSetup: func(client *MockStorageService) {
				// Mock Get to return 404
				client.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("status 404")
				}

				// Mock Upload to fail
				client.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return nil, fmt.Errorf("upload error")
				}
			},
			expectedExists: false,
			expectError:    true,
			errorContains:  "failed to store dedup record",
		},
		{
			name:             "Invalid JSON in response",
			internalEntityID: "entity123",
			dedupObjType:     "alert",
			dedupObjID:       "alert123",
			timeBucket:       string(TimeBucketFiveMin),
			mockSetup: func(client *MockStorageService) {
				// Mock Get to return invalid JSON
				client.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					writer.Write([]byte("invalid json"))
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			expectedExists: false,
			expectError:    true,
			errorContains:  "failed to unmarshal dedup record",
		},
	}

	// Run tests
	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Reset mock storage for each test
			s.SetupTest()

			tc.mockSetup(s.mockStorage)

			// Mock time.Now for calculateTimeBucket
			originalTimeNow := timeNow
			defer func() { timeNow = originalTimeNow }()
			timeNow = func() time.Time {
				return time.Date(2023, 5, 15, 10, 0, 0, 0, time.UTC)
			}

			exists, err := CheckThrottlingStore(context.Background(), s.mockStorage, s.logger,
				tc.internalEntityID, tc.dedupObjType, tc.dedupObjID, tc.timeBucket)

			if tc.expectError {
				s.Error(err)
				if tc.errorContains != "" {
					s.Contains(err.Error(), tc.errorContains)
				}
			} else {
				s.NoError(err)
			}

			s.Equal(tc.expectedExists, exists)
		})
	}
}

// TestCheckExternalEntityExists tests the CheckExternalEntityExists function
func (s *StorageTestSuite) TestCheckExternalEntityExists() {
	tests := []struct {
		name             string
		internalEntityID string
		externalSystemID string
		mockSetup        func(*MockStorageService)
		expectedExists   bool
		expectedRecord   *ExternalEntityRecord
		expectError      bool
		errorContains    string
	}{
		{
			name:             "Record doesn't exist",
			internalEntityID: "entity123",
			externalSystemID: "servicenow_incident",
			mockSetup: func(client *MockStorageService) {
				// Mock Get to return 404
				client.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					// Verify that the ObjectKey is correctly formed
					expectedKey, err := CreateTrackedEntityKey("servicenow_incident", "entity123")
					if err != nil {
						s.T().Errorf("Unexpected error creating tracked entity key: %v", err)
						return nil, err
					}
					s.Equal(expectedKey, params.ObjectKey, "ObjectKey should match expected value")
					return nil, fmt.Errorf("status 404")
				}
			},
			expectedExists: false,
			expectedRecord: nil,
			expectError:    false,
		},
		{
			name:             "Record exists",
			internalEntityID: "entity123",
			externalSystemID: "servicenow_incident",
			mockSetup: func(client *MockStorageService) {
				// Mock Get to return a record
				client.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					// Verify that the ObjectKey is correctly formed
					expectedKey, err := CreateTrackedEntityKey("servicenow_incident", "entity123")
					if err != nil {
						s.T().Errorf("Unexpected error creating tracked entity key: %v", err)
						return nil, err
					}
					s.Equal(expectedKey, params.ObjectKey, "ObjectKey should match expected value")
					record := ExternalEntityRecord{
						InternalEntityID: "entity123",
						ExternalEntityID: "ext123",
						ExternalSystemID: "servicenow_incident",
					}
					json.NewEncoder(writer).Encode(record)
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			expectedExists: true,
			expectedRecord: &ExternalEntityRecord{
				InternalEntityID: "entity123",
				ExternalEntityID: "ext123",
				ExternalSystemID: "servicenow_incident",
			},
			expectError: false,
		},
		{
			name:             "Error getting record",
			internalEntityID: "entity123",
			externalSystemID: "servicenow_incident",
			mockSetup: func(client *MockStorageService) {
				// Mock Get to return an error
				client.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					return nil, fmt.Errorf("connection error")
				}
			},
			expectedExists: false,
			expectedRecord: nil,
			expectError:    true,
			errorContains:  "failed to check if external entity exists",
		},
		{
			name:             "Invalid JSON in response",
			internalEntityID: "entity123",
			externalSystemID: "servicenow_incident",
			mockSetup: func(client *MockStorageService) {
				// Mock Get to return invalid JSON
				client.GetObjectFunc = func(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
					writer.Write([]byte("invalid json"))
					return &custom_storage.GetObjectOK{}, nil
				}
			},
			expectedExists: true, // The function returns true even if unmarshaling fails
			expectedRecord: nil,
			expectError:    true,
			errorContains:  "failed to unmarshal external entity record",
		},
	}

	// Run tests
	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Reset mock storage for each test
			s.SetupTest()

			tc.mockSetup(s.mockStorage)

			exists, record, err := CheckExternalEntityExists(context.Background(), s.mockStorage, s.logger, tc.internalEntityID, tc.externalSystemID)

			if tc.expectError {
				s.Error(err)
				if tc.errorContains != "" {
					s.Contains(err.Error(), tc.errorContains)
				}
			} else {
				s.NoError(err)
			}

			s.Equal(tc.expectedExists, exists)

			if tc.expectedRecord == nil {
				s.Nil(record)
			} else if tc.expectedRecord != nil && record != nil {
				s.Equal(tc.expectedRecord.InternalEntityID, record.InternalEntityID)
				s.Equal(tc.expectedRecord.ExternalEntityID, record.ExternalEntityID)
				s.Equal(tc.expectedRecord.ExternalSystemID, record.ExternalSystemID)
			}
		})
	}
}

// TestCreateOrUpdateExternalEntityMapping tests the createOrUpdateExternalEntityMapping function
func (s *StorageTestSuite) TestCreateOrUpdateExternalEntityMapping() {
	tests := []struct {
		name          string
		record        ExternalEntityRecord
		mockSetup     func(*MockStorageService)
		expectError   bool
		errorContains string
	}{
		{
			name: "Successful mapping creation",
			record: ExternalEntityRecord{
				InternalEntityID: "entity123",
				ExternalEntityID: "ext123",
				ExternalSystemID: "servicenow",
			},
			mockSetup: func(client *MockStorageService) {
				// Mock Upload to succeed
				client.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					// Verify that the ObjectKey is correctly formed
					expectedKey, err := CreateTrackedEntityKey("servicenow", "entity123")
					if err != nil {
						s.T().Errorf("Unexpected error creating tracked entity key: %v", err)
						return nil, err
					}
					s.Equal(expectedKey, params.ObjectKey, "ObjectKey should match expected value")
					return &custom_storage.PutObjectOK{}, nil
				}
			},
			expectError: false,
		},
		{
			name: "Error uploading record",
			record: ExternalEntityRecord{
				InternalEntityID: "entity123",
				ExternalEntityID: "ext123",
				ExternalSystemID: "servicenow",
			},
			mockSetup: func(client *MockStorageService) {
				// Mock Upload to fail
				client.PutObjectFunc = func(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
					return nil, fmt.Errorf("upload error")
				}
			},
			expectError:   true,
			errorContains: "error storing entity mapping in collection",
		},
	}

	// Run tests
	for _, tc := range tests {
		s.Run(tc.name, func() {
			// Reset mock storage for each test
			s.SetupTest()

			tc.mockSetup(s.mockStorage)

			err := CreateOrUpdateExternalEntityMapping(context.Background(), s.mockStorage, s.logger, tc.record)

			if tc.expectError {
				s.Error(err)
				if tc.errorContains != "" {
					s.Contains(err.Error(), tc.errorContains)
				}
			} else {
				s.NoError(err)
			}
		})
	}
}

// TestStorageSuite runs the storage test suite
func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}
