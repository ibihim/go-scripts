package gotools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemporaryDirCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "goupdate-*")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("tmpDir path:", tmpDir)

	filename := "go1.24.1.linux-amd64.tar.gz"
	outputPath := filepath.Join(tmpDir, filename)

	// Create output file
	output, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("failed to create output file: %v", err)
	}
	defer output.Close()
}
