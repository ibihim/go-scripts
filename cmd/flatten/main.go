// This script traverses a source repository and copies all files into a single
// destination directory. It encodes the original folder structure into the file names,
// using "__" as a separator to avoid collisions. It supports file filtering based on
// include/exclude glob patterns (matched against the file's base name) using only the
// standard library.
package main

import (
	"flag"
	"fmt"
	"io"
	"iter"
	"log"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

// Options holds the raw command-line flag values.
type Options struct {
	Source          string
	Dest            string
	IncludePatterns []string
	ExcludePatterns []string
}

// ParseOptions reads the command-line flags and returns an Options instance.
// If -help is specified, it prints the usage and exits. This function centralizes
// flag parsing for cleaner separation of concerns.
func ParseOptions() *Options {
	// Define a custom help flag to explicitly print help when needed.
	helpFlag := flag.Bool("help", false, "Print help information")
	sourcePtr := flag.String("source", "", "Path to the source repository directory")
	destPtr := flag.String("dest", "", "Path to the destination directory for flattened files")
	includePtr := flag.String("include", "", "Comma-separated list of glob patterns to include (e.g. '*.go')")
	excludePtr := flag.String("exclude", "", "Comma-separated list of glob patterns to exclude (e.g. '*_test.go')")
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	opts := &Options{
		Source:          *sourcePtr,
		Dest:            *destPtr,
		IncludePatterns: parseCommaSeparated(*includePtr),
		ExcludePatterns: parseCommaSeparated(*excludePtr),
	}
	return opts
}

// parseCommaSeparated splits a comma-separated string and trims spaces.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// Validate ensures that required options are provided. This is inspired by
// similar validation functions found in projects like Kubernetes.
func (o *Options) Validate() error {
	if o.Source == "" {
		return fmt.Errorf("source directory must be provided")
	}
	return nil
}

// Config holds the processed configuration for file flattening.
// The include and exclude patterns are stored as sets (maps with keys only)
// for efficient lookup.
type Config struct {
	Source     string
	Dest       string
	IncludeSet map[string]bool
	ExcludeSet map[string]bool
}

// ApplyTo transfers the Options into the Config, converting the pattern slices
// into sets. This pattern of "applying" options makes it easier to manage configuration.
func (o *Options) Config() *Config {
	var cfg Config

	if o.Dest != "" {
		cfg.Dest = o.Dest
	} else {
		cfg.Dest = fmt.Sprintf("%s_flatten", o.Source)
	}

	cfg.Source = o.Source
	cfg.IncludeSet = make(map[string]bool)
	cfg.ExcludeSet = make(map[string]bool)

	maps.Insert(cfg.IncludeSet, setify(o.IncludePatterns))
	maps.Insert(cfg.ExcludeSet, setify(o.ExcludePatterns))

	return &cfg
}

func main() {
	// Parse and validate command-line options.
	opts := ParseOptions()
	if err := opts.Validate(); err != nil {
		log.Fatalf("Invalid options: %v", err)
	}

	// Create a Config instance from Options.
	config := opts.Config()

	// Create the destination directory if it doesn't exist.
	if err := os.MkdirAll(config.Dest, os.ModePerm); err != nil {
		log.Fatalf("Failed to create destination directory: %v", err)
	}

	// Walk the source directory.
	err := filepath.Walk(config.Source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Propagate any error encountered during traversal.
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}

		// Compute the file's relative path (to encode the original location).
		rel, err := filepath.Rel(config.Source, path)
		if err != nil {
			return err
		}

		// Use the file's base name for pattern matching.
		baseName := filepath.Base(path)

		// If include set is non-empty, the file must match at least one pattern.
		if len(config.IncludeSet) > 0 {
			matched := false
			for pattern := range config.IncludeSet {
				if ok, err := filepath.Match(pattern, baseName); err == nil && ok {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		// Exclude the file if it matches any exclusion pattern.
		for pattern := range config.ExcludeSet {
			if ok, err := filepath.Match(pattern, baseName); err == nil && ok {
				return nil
			}
		}

		// Encode the relative path into a flattened file name using "__" as separator.
		flattenedName := flattenName(rel)
		destPath := filepath.Join(config.Dest, flattenedName)
		// Resolve potential collisions by appending a counter.
		destPath = resolveCollision(destPath)

		// Copy the file while preserving permissions.
		if err := copyFile(path, destPath); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", path, destPath, err)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error processing files: %v", err)
	}
}

// flattenName converts a relative path into a flat file name by replacing all
// path separators with a double underscore "__". This helps avoid collisions.
func flattenName(relPath string) string {
	return strings.ReplaceAll(relPath, string(os.PathSeparator), "__")
}

// resolveCollision ensures the destination file name is unique by appending a counter.
func resolveCollision(filePath string) string {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return filePath
	}
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)
	counter := 1
	newPath := fmt.Sprintf("%s_%d%s", base, counter, ext)
	for {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
		newPath = fmt.Sprintf("%s_%d%s", base, counter, ext)
	}
}

// copyFile copies the file from src to dst and preserves file permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}

func setify[Slice ~[]E, E any](s Slice) iter.Seq2[E, bool] {
	return func(yield func(E, bool) bool) {
		for _, v := range s {
			if !yield(v, true) {
				return
			}
		}
	}
}
