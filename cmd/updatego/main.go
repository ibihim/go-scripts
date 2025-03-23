package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ibihim/go-scripts/pkg/gotools"
)

func main() {
	if err := app(); err != nil {
		fmt.Println(err)
	}
}

func app() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	checker := gotools.NewChecker()
	currentVersion := checker.GetInstalledVersion()
	latestVersion, err := checker.GetLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	needsUpdate, err := checker.NeedsUpdate(currentVersion, latestVersion)
	if err != nil {
		return fmt.Errorf("failed to check if update is needed: %w", err)
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Printf("Latest version: %s\n", latestVersion)
	fmt.Printf("Update needed: %t\n", needsUpdate)

	if !needsUpdate {
		return nil
	}

	downloader := gotools.NewDownloader()
	path, err := downloader.Download(ctx, latestVersion)
	if err != nil {
		return fmt.Errorf("failed to download latest version: %w", err)
	}

	verified, err := downloader.VerifyChecksum(ctx, path, latestVersion)
	if err != nil {
		return fmt.Errorf("failed to verify downloaded version: %w", err)
	}
	if !verified {
		return fmt.Errorf("downloaded version could not be verified")
	} else {
		fmt.Println("Downloaded version verified")
	}

	fmt.Printf("Version %s downloaded and verified at path %s\n", latestVersion, path)

	installer, err := gotools.NewInstaller()
	if err := installer.Install(ctx, path); err != nil {
		return fmt.Errorf("failed to install Go: %w", err)
	}

	fmt.Println("Go installed successfully")

	return nil
}
