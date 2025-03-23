package gotools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Downloader handles downloading Go releases
type Downloader struct {
	client *http.Client
}

// NewDownloader creates a new downloader with the given options
func NewDownloader() *Downloader {
	return &Downloader{
		client: NewHTTPClient(), // Using the shared HTTP client
	}
}

// Download downloads the Go release for the given version
func (d *Downloader) Download(ctx context.Context, version string) (string, error) {
	// Create temporary directory and create file handle.
	tmpDir, err := os.MkdirTemp("", "goupdate-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	filename := fmt.Sprintf("go%s.linux-amd64.tar.gz", version)
	url := fmt.Sprintf("https://dl.google.com/go/%s", filename)
	outputPath := filepath.Join(tmpDir, filename)

	output, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer output.Close()

	// Try to download the file.
	var lastSeenErr error
	err = wait.PollUntilContextTimeout(ctx, interval, timeout, immediate, func(ctx context.Context) (bool, error) {
		// Reset file position and truncate file to the beginning.
		if _, err := output.Seek(0, 0); err != nil {
			return false, fmt.Errorf("failed to reset file position: %w", err)
		}
		if err := output.Truncate(0); err != nil {
			return false, fmt.Errorf("failed to truncate file: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return false, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			lastSeenErr = fmt.Errorf("failed to perform HTTP request: %w", err)
			return false, nil // Temporary error, retry
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastSeenErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return false, nil // Non-200 status code, retry
		}

		_, err = io.Copy(output, resp.Body)
		if err != nil {
			lastSeenErr = fmt.Errorf("failed to copy response body: %w", err)
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		if lastSeenErr != nil {
			return "", fmt.Errorf("download failed with: %w", lastSeenErr)
		}

		return "", fmt.Errorf("download failed after retries: %w", err)
	}

	return outputPath, nil
}

// VerifyChecksum verifies the downloaded file checksum
func (d *Downloader) VerifyChecksum(ctx context.Context, filePath, version string) (bool, error) {
	// Get expected checksum
	expectedSum, err := d.fetchChecksum(ctx, version)
	if err != nil {
		return false, fmt.Errorf("failed to fetch checksum: %w", err)
	}

	// Calculate actual checksum
	actualSum, err := d.calculateChecksum(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return expectedSum == actualSum, nil
}

// fetchChecksum fetches the expected checksum for a version
func (d *Downloader) fetchChecksum(ctx context.Context, version string) (string, error) {
	checksumURL := fmt.Sprintf("https://dl.google.com/go/go%s.linux-amd64.tar.gz.sha256", version)

	var checksumBytes []byte
	var lastSeenErr error

	err := wait.PollUntilContextTimeout(ctx, interval, timeout, immediate, func(ctx context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", checksumURL, nil)
		if err != nil {
			return false, fmt.Errorf("failed to create checksum request: %w", err)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			lastSeenErr = fmt.Errorf("failed to perform HTTP request: %w", err)
			return false, nil // Temporary error, retry
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastSeenErr = fmt.Errorf("unexpected status code %d", resp.StatusCode)
			return false, nil // Non-200 status code, retry
		}

		// Read checksum (should be a single line with the SHA256 hash)
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			lastSeenErr = fmt.Errorf("failed to read checksum response: %w", err)
			return false, nil // Read error, retry
		}

		checksumBytes = bytes
		return true, nil // Success, don't retry
	})

	if err != nil {
		if lastSeenErr != nil {
			return "", fmt.Errorf("failed to fetch with: %w", lastSeenErr)
		}

		return "", fmt.Errorf("failed to fetch checksum after retries: %w", err)
	}

	// Extract just the hash part (format is usually "<hash>  <filename>")
	checksum := string(checksumBytes)
	parts := strings.Fields(checksum)
	if len(parts) > 0 {
		return parts[0], nil
	}

	// Then it is just the <hash>
	return strings.TrimSpace(checksum), nil
}

// calculateChecksum calculates the SHA256 checksum of a file
func (d *Downloader) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for checksum calculation: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file for checksum calculation: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
