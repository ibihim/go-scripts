// Package install handles the installation of Go releases
package gotools

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// Installer handles the installation process
type Installer struct {
	// InstallDir specifies the installation directory
	InstallDir string
	// BinDir specifies where to symlink the go binary
	BinDir string
}

// NewInstaller creates a new installer with non-sudo defaults
func NewInstaller() (*Installer, error) {
	installDir := ""
	binDir := ""

	usr, err := user.Current()
	if err != nil {
		// Fallback to $HOME
		homeDir := os.Getenv("HOME")
		if homeDir != "" {
			installDir = filepath.Join(homeDir, ".local", "lib")
			binDir = filepath.Join(homeDir, ".local", "bin")
		} else {
			return nil, fmt.Errorf("failed to determine installation directory: %w", err)
		}
	} else {
		installDir = filepath.Join(usr.HomeDir, ".local", "lib")
		binDir = filepath.Join(usr.HomeDir, ".local", "bin")
	}

	return &Installer{
		InstallDir: installDir,
		BinDir:     binDir,
	}, nil
}

// Install installs Go from the given tarball
func (i *Installer) Install(ctx context.Context, tarballPath string) error {
	if err := i.ensureDirectories(); err != nil {
		return fmt.Errorf("failed to create installation directories: %w", err)
	}

	if err := i.removeExisting(); err != nil {
		return fmt.Errorf("failed to remove existing installation: %w", err)
	}

	if err := i.extractTarball(ctx, tarballPath); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	if err := i.createSymlinks(); err != nil {
		return fmt.Errorf("failed to create symlinks: %w", err)
	}

	return nil
}

// ensureDirectories creates the necessary directories for installation
func (i *Installer) ensureDirectories() error {
	if err := os.MkdirAll(i.InstallDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory %s: %w", i.InstallDir, err)
	}

	if err := os.MkdirAll(i.BinDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory %s: %w", i.BinDir, err)
	}

	return nil
}

// removeExisting removes any existing Go installation
func (i *Installer) removeExisting() error {
	goDir := filepath.Join(i.InstallDir, "go")
	if _, err := os.Stat(goDir); os.IsNotExist(err) {
		return nil
	}
	if err := os.RemoveAll(goDir); err != nil {
		return fmt.Errorf("failed to remove existing Go installation: %w", err)
	}

	goLink := filepath.Join(i.BinDir, "go")
	if _, err := os.Lstat(goLink); err == nil {
		if err := os.Remove(goLink); err != nil {
			return fmt.Errorf("failed to remove existing Go symlink: %w", err)
		}
	}

	goFmtLink := filepath.Join(i.BinDir, "gofmt")
	if _, err := os.Lstat(goFmtLink); err == nil {
		if err := os.Remove(goFmtLink); err != nil {
			return fmt.Errorf("failed to remove existing gofmt symlink: %w", err)
		}
	}

	return nil
}

// extractTarball extracts the Go tarball to the installation directory
func (i *Installer) extractTarball(ctx context.Context, tarballPath string) error {
	archive, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("failed to open tarball: %w", err)
	}
	defer archive.Close()

	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("extraction cancelled: %w", ctx.Err())
		default:
		}

		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct target path - header.Name is name of the file entry.
		target := filepath.Join(i.InstallDir, header.Name)

		// SECURITY: Verify the target path is still within the installation directory
		// This is critical to prevent path traversal attacks from malicious tarballs
		// that might contain entries with "../" to try to write files outside the intended directory
		if !strings.HasPrefix(target, i.InstallDir) {
			return fmt.Errorf("invalid tar entry (path traversal attempt): %s", header.Name)
		}

		// Handle different file types
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}

		case tar.TypeReg: // indicates a regular file
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", target, err)
			}

			// Give me a fresh, empty file that I can write to, creating it if needed or clearing it if it already exists.
			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			file.Close()

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for symlink %s: %w", target, err)
			}

			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", target, header.Linkname, err)
			}

		default:
			log.Printf("Skipping unsupported file type %c: %s", header.Typeflag, header.Name)
		}
	}

	return nil
}

// createSymlinks creates symlinks to Go binaries
func (i *Installer) createSymlinks() error {
	goSrc := filepath.Join(i.InstallDir, "go", "bin", "go")
	goDst := filepath.Join(i.BinDir, "go")
	if err := os.Symlink(goSrc, goDst); err != nil {
		return fmt.Errorf("failed to create symlink for go: %w", err)
	}

	goFmtSrc := filepath.Join(i.InstallDir, "go", "bin", "gofmt")
	goFmtDst := filepath.Join(i.BinDir, "gofmt")
	if err := os.Symlink(goFmtSrc, goFmtDst); err != nil {
		return fmt.Errorf("failed to create symlink for gofmt: %w", err)
	}

	return nil
}

// Verify verifies that Go was installed correctly
func (i *Installer) Verify(ctx context.Context) error {
	goPath := filepath.Join(i.BinDir, "go")

	if _, err := os.Stat(goPath); os.IsNotExist(err) {
		return fmt.Errorf("Go binary not found at %s", goPath)
	}

	// Test the Go installation by running 'go version'
	output := &strings.Builder{}
	err := i.runGoCommand(ctx, goPath, []string{"version"}, output)
	if err != nil {
		return fmt.Errorf("Go installation verification failed: %s: %w", output.String(), err)
	}

	return nil
}

// runGoCommand runs a Go command and captures its output
func (i *Installer) runGoCommand(ctx context.Context, goPath string, args []string, output *strings.Builder) error {
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	defer r.Close()

	cmd := &exec.Cmd{
		Path:   goPath,
		Args:   append([]string{goPath}, args...),
		Stdout: w,
		Stderr: w,
	}

	if err := cmd.Start(); err != nil {
		w.Close()
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Close write end of pipe immediately after starting the command
	// This is crucial - it signals to the read end (r) that no more data will be written
	// Using defer w.Close() would cause io.ReadAll(r) to block indefinitely
	// since it would wait for EOF which only happens when all writers are closed
	w.Close()

	outputBytes, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read command output: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	select {
	case err := <-waitCh:
		if err != nil {
			return fmt.Errorf("command failed: %w", err)
		}
	case <-ctx.Done():
		cmd.Process.Kill()
		return ctx.Err()
	}

	output.Write(outputBytes)
	return nil
}

// GetPathUpdateInstructions returns instructions for updating the PATH
func (i *Installer) GetPathUpdateInstructions() string {
	// For user installs, provide instructions
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\nTo use Go, ensure '%s' is in your PATH.\n", i.BinDir))
	sb.WriteString("\nYou can add it to your shell profile (~/.bashrc, ~/.zshrc, etc.):\n\n")
	sb.WriteString(fmt.Sprintf("  export PATH=\"$PATH:%s\"\n", i.BinDir))

	return sb.String()
}
