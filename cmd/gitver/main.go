package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Check if we're in a git repository with commits
	if err := validateGitRepo(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// Generate the pseudo-version
	version, err := generatePseudoVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print just the version string
	fmt.Print(version)
}

// validateGitRepo checks if we're in a git repository with commits
func validateGitRepo() error {
	// Verify we're in a git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not inside a git repository")
	}

	// Check for at least one commit
	cmd = exec.Command("git", "log", "-1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("repository has no commits")
	}

	return nil
}

// getCommitTimestamp gets the timestamp of the latest commit
func getCommitTimestamp() (time.Time, error) {
	output, err := exec.Command("git", "log", "-1", "--format=%ct").Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get commit timestamp: %w", err)
	}

	timestamp := strings.TrimSpace(string(output))
	unixTime, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return time.Unix(unixTime, 0), nil
}

// getCommitHash gets the hash of the latest commit
func getCommitHash() (string, error) {
	output, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	fullHash := strings.TrimSpace(string(output))
	if len(fullHash) >= 12 {
		return fullHash[:12], nil
	}

	return fullHash, nil
}

// generatePseudoVersion creates a pseudo-version string using the current git commit
func generatePseudoVersion() (string, error) {
	// Get the commit timestamp
	commitTime, err := getCommitTimestamp()
	if err != nil {
		return "", err
	}

	formattedTime := commitTime.UTC().Format("20060102150405")

	// Get the commit hash
	shortHash, err := getCommitHash()
	if err != nil {
		return "", err
	}

	// Return the pseudo-version
	return fmt.Sprintf("v0.0.0-%s-%s", formattedTime, shortHash), nil
}
