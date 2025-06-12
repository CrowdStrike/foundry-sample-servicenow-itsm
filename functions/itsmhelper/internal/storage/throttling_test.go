package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ThrottlingTestSuite defines the test suite for throttling functionality
type ThrottlingTestSuite struct {
	suite.Suite
	originalTimeNow func() time.Time
}

// SetupSuite runs once before all tests in the suite
func (s *ThrottlingTestSuite) SetupSuite() {
	s.originalTimeNow = timeNow
}

// TearDownSuite runs once after all tests in the suite
func (s *ThrottlingTestSuite) TearDownSuite() {
	timeNow = s.originalTimeNow
}

// withMockedTime sets the timeNow variable to return a fixed time during test execution
func (s *ThrottlingTestSuite) withMockedTime(mockTime time.Time, testFunc func()) {
	// Replace with mock
	timeNow = func() time.Time {
		return mockTime
	}

	// Run the test function
	testFunc()

	// Restore original
	timeNow = s.originalTimeNow
}

// TestCalculateTimeBucket_ValidInputs tests the calculateTimeBucket function with valid inputs
func (s *ThrottlingTestSuite) TestCalculateTimeBucket_ValidInputs() {
	// Test cases for each valid TimeBucket value
	tests := []struct {
		name     string
		bucket   TimeBucket
		expected string
		mockTime time.Time
	}{
		{
			name:     "Forever bucket",
			bucket:   TimeBucketForever,
			expected: "forever_bucket",
			mockTime: time.Time{}, // Not used for forever bucket
		},
		{
			name:     "Five minute bucket",
			bucket:   TimeBucketFiveMin,
			expected: "2023-05-15_10:15",
			mockTime: time.Date(2023, 5, 15, 10, 17, 30, 0, time.UTC),
		},
		{
			name:     "Thirty minute bucket",
			bucket:   TimeBucketThirtyMin,
			expected: "2023-05-15_10:00",
			mockTime: time.Date(2023, 5, 15, 10, 17, 30, 0, time.UTC),
		},
	}

	// Execute tests with time mocking
	for _, tc := range tests {
		s.Run(tc.name, func() {
			if tc.bucket == TimeBucketForever {
				// For TimeBucketForever, we don't need to mock time
				result, err := calculateTimeBucket(tc.bucket)
				s.NoError(err)
				s.Equal(tc.expected, result)
			} else {
				// For time-based buckets, we need to mock time
				s.withMockedTime(tc.mockTime, func() {
					result, err := calculateTimeBucket(tc.bucket)
					s.NoError(err)
					s.Equal(tc.expected, result)
				})
			}
		})
	}
}

// TestCalculateTimeBucket_InvalidInput tests the calculateTimeBucket function with invalid inputs
func (s *ThrottlingTestSuite) TestCalculateTimeBucket_InvalidInput() {
	// Test cases for invalid TimeBucket values
	tests := []struct {
		name          string
		bucket        TimeBucket
		expectedError string
	}{
		{
			name:          "Empty bucket",
			bucket:        TimeBucket(""),
			expectedError: "invalid time bucket: ",
		},
		{
			name:          "Invalid bucket",
			bucket:        TimeBucket("invalid_bucket"),
			expectedError: "invalid time bucket: invalid_bucket",
		},
		{
			name:          "Numeric bucket",
			bucket:        TimeBucket("123"),
			expectedError: "invalid time bucket: 123",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result, err := calculateTimeBucket(tc.bucket)

			s.Error(err)
			s.Equal(tc.expectedError, err.Error())
			s.Empty(result)
		})
	}
}

// TestCalculateTimeBucket_EdgeCases tests the calculateTimeBucket function with edge cases
func (s *ThrottlingTestSuite) TestCalculateTimeBucket_EdgeCases() {
	// Test cases for edge time values
	tests := []struct {
		name     string
		bucket   TimeBucket
		mockTime time.Time
		expected string
	}{
		// Five minute bucket tests
		{
			name:     "Five minute bucket at exact 5-min interval",
			bucket:   TimeBucketFiveMin,
			mockTime: time.Date(2023, 5, 15, 10, 15, 0, 0, time.UTC),
			expected: "2023-05-15_10:15",
		},
		{
			name:     "Five minute bucket just before interval",
			bucket:   TimeBucketFiveMin,
			mockTime: time.Date(2023, 5, 15, 10, 19, 59, 999, time.UTC),
			expected: "2023-05-15_10:15",
		},
		{
			name:     "Five minute bucket at start of interval",
			bucket:   TimeBucketFiveMin,
			mockTime: time.Date(2023, 5, 15, 10, 20, 0, 0, time.UTC),
			expected: "2023-05-15_10:20",
		},
		{
			name:     "Five minute bucket with odd minute",
			bucket:   TimeBucketFiveMin,
			mockTime: time.Date(2023, 5, 15, 10, 23, 45, 0, time.UTC),
			expected: "2023-05-15_10:20",
		},
		{
			name:     "Five minute bucket at midnight",
			bucket:   TimeBucketFiveMin,
			mockTime: time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
			expected: "2023-05-15_00:00",
		},
		{
			name:     "Five minute bucket at end of hour",
			bucket:   TimeBucketFiveMin,
			mockTime: time.Date(2023, 5, 15, 10, 59, 59, 999, time.UTC),
			expected: "2023-05-15_10:55",
		},

		// Thirty minute bucket tests
		{
			name:     "Thirty minute bucket at exact 30-min interval",
			bucket:   TimeBucketThirtyMin,
			mockTime: time.Date(2023, 5, 15, 10, 30, 0, 0, time.UTC),
			expected: "2023-05-15_10:30",
		},
		{
			name:     "Thirty minute bucket just before interval",
			bucket:   TimeBucketThirtyMin,
			mockTime: time.Date(2023, 5, 15, 10, 59, 59, 999, time.UTC),
			expected: "2023-05-15_10:30",
		},
		{
			name:     "Thirty minute bucket at start of hour",
			bucket:   TimeBucketThirtyMin,
			mockTime: time.Date(2023, 5, 15, 10, 0, 0, 0, time.UTC),
			expected: "2023-05-15_10:00",
		},
		{
			name:     "Thirty minute bucket at day change",
			bucket:   TimeBucketThirtyMin,
			mockTime: time.Date(2023, 5, 15, 23, 45, 0, 0, time.UTC),
			expected: "2023-05-15_23:30",
		},

		// Date formatting tests
		{
			name:     "Single digit day",
			bucket:   TimeBucketFiveMin,
			mockTime: time.Date(2023, 5, 5, 10, 15, 0, 0, time.UTC),
			expected: "2023-05-05_10:15",
		},
		{
			name:     "Single digit month",
			bucket:   TimeBucketFiveMin,
			mockTime: time.Date(2023, 1, 15, 10, 15, 0, 0, time.UTC),
			expected: "2023-01-15_10:15",
		},
		{
			name:     "Year boundary",
			bucket:   TimeBucketThirtyMin,
			mockTime: time.Date(2023, 12, 31, 23, 45, 0, 0, time.UTC),
			expected: "2023-12-31_23:30",
		},
	}

	// Execute tests with time mocking
	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.withMockedTime(tc.mockTime, func() {
				result, err := calculateTimeBucket(tc.bucket)
				s.NoError(err)
				s.Equal(tc.expected, result)
			})
		})
	}
}

// TestThrottlingSuite runs the throttling test suite
func TestThrottlingSuite(t *testing.T) {
	suite.Run(t, new(ThrottlingTestSuite))
}
