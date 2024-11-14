package main

import (
	"io"
	"testing"
	"time"

	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-skyline/github"
	"github.com/github/gh-skyline/types"
)

// MockGitHubClient implements the github.APIClient interface
type MockGitHubClient struct {
	username string
	joinYear int
}

// Get implements the APIClient interface
func (m *MockGitHubClient) Get(_ string, _ interface{}) error {
	return nil
}

// Post implements the APIClient interface
func (m *MockGitHubClient) Post(path string, body io.Reader, response interface{}) error {
	if path == "graphql" {
		// Read the request body to determine which GraphQL query is being made
		bodyBytes, _ := io.ReadAll(body)
		bodyStr := string(bodyBytes)

		if strings.Contains(bodyStr, "UserJoinDate") {
			// Handle user join date query
			resp := response.(*struct {
				Data struct {
					User struct {
						CreatedAt string `json:"createdAt"`
					} `json:"user"`
				} `json:"data"`
			})
			resp.Data.User.CreatedAt = time.Date(m.joinYear, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
			return nil
		}

		if strings.Contains(bodyStr, "ContributionGraph") {
			// Handle contribution graph query (existing logic)
			return json.Unmarshal(contributionResponse(m.username), response)
		}
	}
	return nil
}

// Helper function to generate mock contribution response
func contributionResponse(username string) []byte {
	response := fmt.Sprintf(`{
        "data": {
            "user": {
                "login": "%s",
                "contributionsCollection": {
                    "contributionCalendar": {
                        "totalContributions": 1,
                        "weeks": [
                            {
                                "contributionDays": [
                                    {
                                        "contributionCount": 1,
                                        "date": "2024-01-01"
                                    }
                                ]
                            }
                        ]
                    }
                }
            }
        }
    }`, username)
	return []byte(response)
}

func (m *MockGitHubClient) GetAuthenticatedUser() (string, error) {
	return m.username, nil
}

func (m *MockGitHubClient) GetUserJoinYear(_ string) (int, error) {
	return m.joinYear, nil
}

func (m *MockGitHubClient) FetchContributions(username string, year int) (*types.ContributionsResponse, error) {
	// Return minimal valid response
	resp := &types.ContributionsResponse{}
	resp.Data.User.Login = username
	// Add a single week with a single day for minimal valid data
	week := struct {
		ContributionDays []types.ContributionDay `json:"contributionDays"`
	}{
		ContributionDays: []types.ContributionDay{
			{
				ContributionCount: 1,
				Date:              time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
			},
		},
	}
	resp.Data.User.ContributionsCollection.ContributionCalendar.Weeks = []struct {
		ContributionDays []types.ContributionDay `json:"contributionDays"`
	}{week}
	return resp, nil
}

func TestFormatYearRange(t *testing.T) {
	tests := []struct {
		name      string
		startYear int
		endYear   int
		want      string
	}{
		{
			name:      "same year",
			startYear: 2024,
			endYear:   2024,
			want:      "2024",
		},
		{
			name:      "different years",
			startYear: 2020,
			endYear:   2024,
			want:      "20-24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatYearRange(tt.startYear, tt.endYear)
			if got != tt.want {
				t.Errorf("formatYearRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateOutputFilename(t *testing.T) {
	tests := []struct {
		name      string
		user      string
		startYear int
		endYear   int
		want      string
	}{
		{
			name:      "single year",
			user:      "testuser",
			startYear: 2024,
			endYear:   2024,
			want:      "testuser-2024-github-skyline.stl",
		},
		{
			name:      "year range",
			user:      "testuser",
			startYear: 2020,
			endYear:   2024,
			want:      "testuser-20-24-github-skyline.stl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateOutputFilename(tt.user, tt.startYear, tt.endYear)
			if got != tt.want {
				t.Errorf("generateOutputFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseYearRange(t *testing.T) {
	tests := []struct {
		name          string
		yearRange     string
		wantStart     int
		wantEnd       int
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "single year",
			yearRange: "2024",
			wantStart: 2024,
			wantEnd:   2024,
			wantErr:   false,
		},
		{
			name:      "year range",
			yearRange: "2020-2024",
			wantStart: 2020,
			wantEnd:   2024,
			wantErr:   false,
		},
		{
			name:          "invalid format",
			yearRange:     "2020-2024-2025",
			wantErr:       true,
			wantErrString: "invalid year range format",
		},
		{
			name:      "invalid number",
			yearRange: "abc-2024",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parseYearRange(tt.yearRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseYearRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantErrString != "" && err.Error() != tt.wantErrString {
				t.Errorf("parseYearRange() error = %v, wantErrString %v", err, tt.wantErrString)
				return
			}
			if !tt.wantErr {
				if start != tt.wantStart {
					t.Errorf("parseYearRange() start = %v, want %v", start, tt.wantStart)
				}
				if end != tt.wantEnd {
					t.Errorf("parseYearRange() end = %v, want %v", end, tt.wantEnd)
				}
			}
		})
	}
}

func TestValidateYearRange(t *testing.T) {
	tests := []struct {
		name      string
		startYear int
		endYear   int
		wantErr   bool
	}{
		{
			name:      "valid range",
			startYear: 2020,
			endYear:   2024,
			wantErr:   false,
		},
		{
			name:      "invalid start year",
			startYear: 2007,
			endYear:   2024,
			wantErr:   true,
		},
		{
			name:      "start after end",
			startYear: 2024,
			endYear:   2020,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateYearRange(tt.startYear, tt.endYear)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateYearRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSkyline(t *testing.T) {
	// Save original client creation function
	originalInitFn := initializeGitHubClient
	defer func() {
		initializeGitHubClient = originalInitFn
	}()

	tests := []struct {
		name       string
		startYear  int
		endYear    int
		targetUser string
		full       bool
		mockClient *MockGitHubClient
		wantErr    bool
	}{
		{
			name:       "single year",
			startYear:  2024,
			endYear:    2024,
			targetUser: "testuser",
			full:       false,
			mockClient: &MockGitHubClient{
				username: "testuser",
				joinYear: 2020,
			},
			wantErr: false,
		},
		{
			name:       "year range",
			startYear:  2020,
			endYear:    2024,
			targetUser: "testuser",
			full:       false,
			mockClient: &MockGitHubClient{
				username: "testuser",
				joinYear: 2020,
			},
			wantErr: false,
		},
		{
			name:       "full range",
			startYear:  2020,
			endYear:    2024,
			targetUser: "testuser",
			full:       true,
			mockClient: &MockGitHubClient{
				username: "testuser",
				joinYear: 2020,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override the client initialization for testing
			initializeGitHubClient = func() (*github.Client, error) {
				return github.NewClient(tt.mockClient), nil
			}

			err := generateSkyline(tt.startYear, tt.endYear, tt.targetUser, tt.full)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateSkyline() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
