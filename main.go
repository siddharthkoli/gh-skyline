// Package main provides the entry point for the GitHub Skyline Generator.
// It generates a 3D model of GitHub contributions in STL format.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-skyline/ascii"
	"github.com/github/gh-skyline/errors"
	"github.com/github/gh-skyline/github"
	"github.com/github/gh-skyline/logger"
	"github.com/github/gh-skyline/stl"
	"github.com/github/gh-skyline/types"
)

const (
	// githubLaunchYear represents the year GitHub was launched and contributions began
	githubLaunchYear = 2008
	// outputFileFormat defines the format for the generated STL file
	outputFileFormat = "%s-%s-github-skyline.stl"
)

// formatYearRange returns a formatted string representation of the year range
func formatYearRange(startYear, endYear int) string {
	if startYear == endYear {
		return fmt.Sprintf("%d", startYear)
	}
	return fmt.Sprintf("%02d-%02d", startYear%100, endYear%100)
}

// generateOutputFilename creates a consistent filename for the STL output
func generateOutputFilename(user string, startYear, endYear int) string {
	yearStr := formatYearRange(startYear, endYear)
	return fmt.Sprintf(outputFileFormat, user, yearStr)
}

// generateSkyline creates a 3D model with ASCII art preview of GitHub contributions for the specified year range, or "full lifetime" of the user
func generateSkyline(startYear, endYear int, targetUser string, full bool) error {
	log := logger.GetLogger()

	client, err := initializeGitHubClient()
	if err != nil {
		return errors.New(errors.NetworkError, "failed to initialize GitHub client", err)
	}

	if targetUser == "" {
		if err := log.Debug("No target user specified, using authenticated user"); err != nil {
			return err
		}
		username, err := client.GetAuthenticatedUser()
		if err != nil {
			return errors.New(errors.NetworkError, "failed to get authenticated user", err)
		}
		targetUser = username
	}

	if full {
		joinYear, err := client.GetUserJoinYear(targetUser)
		if err != nil {
			return errors.New(errors.NetworkError, "failed to get user join year", err)
		}
		startYear = joinYear
		endYear = time.Now().Year()
	}

	var allContributions [][][]types.ContributionDay
	for year := startYear; year <= endYear; year++ {
		contributions, err := fetchContributionData(client, targetUser, year)
		if err != nil {
			return err
		}
		allContributions = append(allContributions, contributions)

		// Generate ASCII art for each year
		asciiArt, err := ascii.GenerateASCII(contributions, targetUser, year, year == startYear)
		if err != nil {
			if warnErr := log.Warning("Failed to generate ASCII preview: %v", err); warnErr != nil {
				return warnErr
			}
		} else {
			if year == startYear {
				// For first year, show full ASCII art including header
				fmt.Println(asciiArt)
			} else {
				// For subsequent years, skip the header
				lines := strings.Split(asciiArt, "\n")
				gridStart := 0
				for i, line := range lines {
					if strings.Contains(line, string(ascii.EmptyBlock)) ||
						strings.Contains(line, string(ascii.FoundationLow)) {
						gridStart = i
						break
					}
				}
				// Print just the grid and user info
				fmt.Println(strings.Join(lines[gridStart:], "\n"))
			}
		}
	}

	// Generate filename
	outputPath := generateOutputFilename(targetUser, startYear, endYear)

	// Generate the STL file
	if len(allContributions) == 1 {
		return stl.GenerateSTL(allContributions[0], outputPath, targetUser, startYear)
	}
	return stl.GenerateSTLRange(allContributions, outputPath, targetUser, startYear, endYear)
}

// Variable for client initialization - allows for testing
var initializeGitHubClient = defaultGitHubClient

// defaultGitHubClient is the default implementation of client initialization
func defaultGitHubClient() (*github.Client, error) {
	apiClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}
	return github.NewClient(apiClient), nil
}

// fetchContributionData retrieves and formats the contribution data for the specified year.
func fetchContributionData(client *github.Client, username string, year int) ([][]types.ContributionDay, error) {
	resp, err := client.FetchContributions(username, year)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contributions: %w", err)
	}

	// Convert weeks data to 2D array for STL generation
	weeks := resp.Data.User.ContributionsCollection.ContributionCalendar.Weeks
	contributionGrid := make([][]types.ContributionDay, len(weeks))
	for i, week := range weeks {
		contributionGrid[i] = week.ContributionDays
	}

	return contributionGrid, nil
}

// main is the entry point for the GitHub Skyline Generator.
func main() {
	yearRange := flag.String("year", fmt.Sprintf("%d", time.Now().Year()), "Year or year range (e.g., 2024 or 2014-2024)")
	user := flag.String("user", "", "GitHub username (optional, defaults to authenticated user)")
	full := flag.Bool("full", false, "Generate contribution graph from join year to current year")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	log := logger.GetLogger()
	if *debug {
		log.SetLevel(logger.DEBUG)
		if err := log.Debug("Debug logging enabled"); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to enable debug logging: %v\n", err)
			os.Exit(1)
		}
	}

	// Parse year range
	startYear, endYear, err := parseYearRange(*yearRange)
	if err != nil {
		if logErr := log.Error("Invalid year range: %v", err); logErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to log error: %v\n", logErr)
		}
		os.Exit(1)
	}

	if err := generateSkyline(startYear, endYear, *user, *full); err != nil {
		if logErr := log.Error("Failed to generate skyline: %v", err); logErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to log error: %v\n", logErr)
		}
		os.Exit(1)
	}
}

// Parse year range string (e.g., "2024" or "2014-2024")
func parseYearRange(yearRange string) (startYear, endYear int, err error) {
	if strings.Contains(yearRange, "-") {
		parts := strings.Split(yearRange, "-")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid year range format")
		}
		startYear, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		endYear, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	} else {
		year, err := strconv.Atoi(yearRange)
		if err != nil {
			return 0, 0, err
		}
		startYear, endYear = year, year
	}
	return startYear, endYear, validateYearRange(startYear, endYear)
}

func validateYearRange(startYear, endYear int) error {
	currentYear := time.Now().Year()
	if startYear < githubLaunchYear || endYear > currentYear {
		return fmt.Errorf("years must be between %d and %d", githubLaunchYear, currentYear)
	}
	if startYear > endYear {
		return fmt.Errorf("start year cannot be after end year")
	}
	return nil
}
