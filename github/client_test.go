package github

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/github/gh-skyline/errors"
)

type MockAPIClient struct {
	GetFunc  func(path string, response interface{}) error
	PostFunc func(path string, body io.Reader, response interface{}) error
}

func (m *MockAPIClient) Get(path string, response interface{}) error {
	return m.GetFunc(path, response)
}

func (m *MockAPIClient) Post(path string, body io.Reader, response interface{}) error {
	return m.PostFunc(path, body, response)
}

// mockAPIClient implements APIClient for testing
type mockAPIClient struct {
	getResponse  string
	postResponse string
	shouldError  bool
}

func (m *mockAPIClient) Get(_ string, response interface{}) error {
	if m.shouldError {
		return errors.New(errors.NetworkError, "mock error", nil)
	}
	return json.Unmarshal([]byte(m.getResponse), response)
}

func (m *mockAPIClient) Post(_ string, _ io.Reader, response interface{}) error {
	if m.shouldError {
		return errors.New(errors.NetworkError, "mock error", nil)
	}
	return json.Unmarshal([]byte(m.postResponse), response)
}

func TestNewClient(t *testing.T) {
	mock := &mockAPIClient{}
	client := NewClient(mock)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.api != mock {
		t.Error("NewClient did not set api client correctly")
	}
}

func TestGetAuthenticatedUser(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		shouldError   bool
		expectedUser  string
		expectedError bool
	}{
		{
			name:          "successful response",
			response:      `{"login": "testuser"}`,
			expectedUser:  "testuser",
			expectedError: false,
		},
		{
			name:          "empty username",
			response:      `{"login": ""}`,
			expectedError: true,
		},
		{
			name:          "network error",
			shouldError:   true,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAPIClient{
				getResponse: tt.response,
				shouldError: tt.shouldError,
			}
			client := NewClient(mock)

			user, err := client.GetAuthenticatedUser()
			if tt.expectedError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if user != tt.expectedUser {
				t.Errorf("expected user %q, got %q", tt.expectedUser, user)
			}
		})
	}
}

func TestFetchContributions(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		year          int
		response      string
		shouldError   bool
		expectedError bool
	}{
		{
			name:     "successful response",
			username: "testuser",
			year:     2023,
			response: `{"data":{"user":{"login":"testuser","contributionsCollection":{"contributionCalendar":{"totalContributions":100,"weeks":[]}}}}}`,
		},
		{
			name:          "empty username",
			username:      "",
			year:          2023,
			expectedError: true,
		},
		{
			name:          "invalid year",
			username:      "testuser",
			year:          2007,
			expectedError: true,
		},
		{
			name:          "network error",
			username:      "testuser",
			year:          2023,
			shouldError:   true,
			expectedError: true,
		},
		{
			name:          "user not found",
			username:      "testuser",
			year:          2023,
			response:      `{"data":{"user":{"login":""}}}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAPIClient{
				postResponse: tt.response,
				shouldError:  tt.shouldError,
			}
			client := NewClient(mock)

			resp, err := client.FetchContributions(tt.username, tt.year)
			if tt.expectedError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectedError && resp == nil {
				t.Error("expected response but got nil")
			}
		})
	}
}

func TestGetUserJoinYear(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		response      string
		shouldError   bool
		expectedYear  int
		expectedError bool
	}{
		{
			name:          "successful response",
			username:      "testuser",
			response:      `{"data":{"user":{"createdAt":"2015-01-01T00:00:00Z"}}}`,
			expectedYear:  2015,
			expectedError: false,
		},
		{
			name:          "empty username",
			username:      "",
			expectedError: true,
		},
		{
			name:          "network error",
			username:      "testuser",
			shouldError:   true,
			expectedError: true,
		},
		{
			name:          "invalid date format",
			username:      "testuser",
			response:      `{"data":{"user":{"createdAt":"invalid-date"}}}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAPIClient{
				postResponse: tt.response,
				shouldError:  tt.shouldError,
			}
			client := NewClient(mock)

			joinYear, err := client.GetUserJoinYear(tt.username)
			if tt.expectedError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if joinYear != tt.expectedYear {
				t.Errorf("expected year %d, got %d", tt.expectedYear, joinYear)
			}
		})
	}
}
