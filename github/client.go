// Package github provides a client for interacting with the GitHub API,
// including fetching authenticated user information and contribution data.
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/github/gh-skyline/errors"
	"github.com/github/gh-skyline/types"
)

// APIClient interface defines the methods we need from the client
type APIClient interface {
	Get(path string, response interface{}) error
	Post(path string, body io.Reader, response interface{}) error
}

// Client holds the API client
type Client struct {
	api APIClient
}

// NewClient creates a new GitHub client
func NewClient(apiClient APIClient) *Client {
	return &Client{api: apiClient}
}

// GetAuthenticatedUser fetches the authenticated user's login name from GitHub.
func (c *Client) GetAuthenticatedUser() (string, error) {
	response := struct{ Login string }{}
	err := c.api.Get("user", &response)
	if err != nil {
		return "", errors.New(errors.NetworkError, "failed to fetch authenticated user", err)
	}

	if response.Login == "" {
		return "", errors.New(errors.ValidationError, "received empty username from GitHub API", nil)
	}

	return response.Login, nil
}

// FetchContributions retrieves the contribution data for a given username and year from GitHub.
func (c *Client) FetchContributions(username string, year int) (*types.ContributionsResponse, error) {
	if username == "" {
		return nil, errors.New(errors.ValidationError, "username cannot be empty", nil)
	}

	if year < 2008 {
		return nil, errors.New(errors.ValidationError, "year cannot be before GitHub's launch (2008)", nil)
	}

	startDate := fmt.Sprintf("%d-01-01", year)
	endDate := fmt.Sprintf("%d-12-31", year)

	query := `
    query ContributionGraph($username: String!, $from: DateTime!, $to: DateTime!) {
        user(login: $username) {
            login
            contributionsCollection(from: $from, to: $to) {
                contributionCalendar {
                    totalContributions
                    weeks {
                        contributionDays {
                            contributionCount
                            date
                        }
                    }
                }
            }
        }
    }`

	variables := map[string]interface{}{
		"username": username,
		"from":     startDate + "T00:00:00Z",
		"to":       endDate + "T23:59:59Z",
	}

	payload := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var resp types.ContributionsResponse
	if err := c.api.Post("graphql", bytes.NewBuffer(body), &resp); err != nil {
		return nil, errors.New(errors.GraphQLError, "failed to fetch contributions", err)
	}

	// Validate response
	if resp.Data.User.Login == "" {
		return nil, errors.New(errors.GraphQLError, "user not found", nil)
	}

	return &resp, nil
}

// GetUserJoinYear fetches the year a user joined GitHub using the GitHub API.
func (c *Client) GetUserJoinYear(username string) (int, error) {
	if username == "" {
		return 0, errors.New(errors.ValidationError, "username cannot be empty", nil)
	}

	query := `
    query UserJoinDate($username: String!) {
        user(login: $username) {
            createdAt
        }
    }`

	variables := map[string]interface{}{
		"username": username,
	}

	payload := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	var resp struct {
		Data struct {
			User struct {
				CreatedAt string `json:"createdAt"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := c.api.Post("graphql", bytes.NewBuffer(body), &resp); err != nil {
		return 0, errors.New(errors.GraphQLError, "failed to fetch user join date", err)
	}

	// Parse the join date
	joinDate, err := time.Parse(time.RFC3339, resp.Data.User.CreatedAt)
	if err != nil {
		return 0, errors.New(errors.ValidationError, "failed to parse join date", err)
	}

	return joinDate.Year(), nil
}
