// Package gotools provides functionality for detecting and comparing Go versions
package gotools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/wait"
)

// GoRelease represents a Go release from the official download page
type GoRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// Checker provides methods to check Go versions
type Checker struct {
	goVersionURL string
	client       *http.Client
}

// NewChecker creates a new version checker with properly configured HTTP client
func NewChecker() *Checker {
	return &Checker{
		goVersionURL: "https://golang.org/dl/?mode=json",
		client:       NewHTTPClient(),
	}
}

// GetInstalledVersion checks the currently installed Go version using runtime.Version()
func (c *Checker) GetInstalledVersion() string {
	return strings.TrimPrefix(runtime.Version(), "go")
}

// GetLatestVersion fetches the latest stable Go release version
func (c *Checker) GetLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.goVersionURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	releases, err := c.getReleasesWithRetry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}

	// Pick first stable release. Assume that it is ordered properly by version.
	for _, release := range releases {
		if release.Stable && strings.HasPrefix(release.Version, "go") {
			return strings.TrimPrefix(release.Version, "go"), nil
		}
	}

	return "", fmt.Errorf("no stable Go releases found")
}

// getReleasesWithRetry tries to fetch the Go releases with retries based on interval and timeout.
func (c *Checker) getReleasesWithRetry(ctx context.Context, req *http.Request) ([]GoRelease, error) {
	// lastErrSeen is used to store the last error encountered during retries.
	var lastErrSeen error
	// body stores the response body from the HTTP request. Can be nil in err case.
	var body io.ReadCloser
	defer safeClose(body)

	timeoutErr := wait.PollUntilContextTimeout(ctx, interval, timeout, immediate, func(ctx context.Context) (bool, error) {
		resp, err := c.client.Do(req)
		if err != nil {
			lastErrSeen = err
			return false, nil
		}

		if resp.StatusCode != http.StatusOK {
			lastErrSeen = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return false, nil
		}

		body = resp.Body

		return true, nil
	})

	if timeoutErr != nil {
		if lastErrSeen != nil {
			return nil, fmt.Errorf("failed to fetch version info: %w", lastErrSeen)
		}

		return nil, fmt.Errorf("failed to fetch version info: %w", timeoutErr)
	}

	var releases []GoRelease
	if err := json.NewDecoder(body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse version info: %w", err)
	}

	return releases, nil
}

// NeedsUpdate determines if an update is needed based on version comparison
func (c *Checker) NeedsUpdate(installed, latest string) (bool, error) {
	// If no version is installed, update is needed
	if installed == "" {
		return true, nil
	}

	// Compare versions
	installedSplit := strings.Split(installed, ".")
	latestSplit := strings.Split(latest, ".")

	// Ensure we have at least 3 parts (major.minor.patch)
	if len(installedSplit) < 3 || len(latestSplit) < 3 {
		return false, fmt.Errorf("invalid version format: versions must be in the format X.Y.Z")
	}

	// Compare major version
	majorInstalled, err := strconv.Atoi(installedSplit[0])
	if err != nil {
		return false, fmt.Errorf("invalid major version in %s: %w", installed, err)
	}

	majorLatest, err := strconv.Atoi(latestSplit[0])
	if err != nil {
		return false, fmt.Errorf("invalid major version in %s: %w", latest, err)
	}

	if majorInstalled < majorLatest {
		return true, nil
	} else if majorInstalled > majorLatest {
		return false, nil
	}

	// Compare minor version
	minorInstalled, err := strconv.Atoi(installedSplit[1])
	if err != nil {
		return false, fmt.Errorf("invalid minor version in %s: %w", installed, err)
	}

	minorLatest, err := strconv.Atoi(latestSplit[1])
	if err != nil {
		return false, fmt.Errorf("invalid minor version in %s: %w", latest, err)
	}

	if minorInstalled < minorLatest {
		return true, nil
	} else if minorInstalled > minorLatest {
		return false, nil
	}

	// Compare patch version
	patch1, err := strconv.Atoi(installedSplit[2])
	if err != nil {
		return false, fmt.Errorf("invalid patch version in %s: %w", installed, err)
	}

	patch2, err := strconv.Atoi(latestSplit[2])
	if err != nil {
		return false, fmt.Errorf("invalid patch version in %s: %w", latest, err)
	}

	if patch1 < patch2 {
		return true, nil
	} else if patch1 > patch2 {
		return false, nil
	}

	// Versions are equal
	return false, nil
}
