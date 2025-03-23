package gotools

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReadCloseOnNil(t *testing.T) {
	t.Run("shouldn't panic with nil body", func(t *testing.T) {
		var body io.ReadCloser

		safeClose(body)
	})

	t.Run("non nil body", func(t *testing.T) {
		body := newTestReadCloser()

		safeClose(body)

		if !body.IsClosed() {
			t.Errorf("body should not be closed")
		}
	})
}

type testReadCloser struct {
	closed bool
}

func (trc *testReadCloser) Close() error {
	trc.closed = true
	return nil
}

func (trc *testReadCloser) IsClosed() bool {
	return trc.closed
}

func newTestReadCloser() *testReadCloser {
	return &testReadCloser{
		closed: false,
	}
}

func TestNeedsUpdate(t *testing.T) {
	checker := NewChecker()

	tests := []struct {
		name      string
		installed string
		latest    string
		expected  bool
		wantErr   bool
	}{
		{
			name:      "no version installed",
			installed: "",
			latest:    "1.17.1",
			expected:  true,
			wantErr:   false,
		},
		{
			name:      "same version",
			installed: "1.17.1",
			latest:    "1.17.1",
			expected:  false,
			wantErr:   false,
		},
		{
			name:      "newer version available",
			installed: "1.16.0",
			latest:    "1.17.1",
			expected:  true,
			wantErr:   false,
		},
		{
			name:      "installed version newer (unusual case)",
			installed: "1.18.0",
			latest:    "1.17.1",
			expected:  false,
			wantErr:   false,
		},
		{
			name:      "invalid installed version",
			installed: "1.a.1",
			latest:    "1.17.1",
			expected:  false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			have, err := checker.NeedsUpdate(tt.installed, tt.latest)
			if (err != nil) != tt.wantErr {
				t.Errorf("NeedsUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && have != tt.expected {
				t.Errorf("NeedsUpdate() = %v, want %v", have, tt.expected)
			}
		})
	}
}

// TestGetLatestVersion tests the GetLatestVersion function
func TestGetLatestVersion(t *testing.T) {
	// Create a test server that serves a mock response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{"version": "go1.17.1", "stable": true},
			{"version": "go1.16.8", "stable": true},
			{"version": "go1.18beta1", "stable": false}
		]`))
	}))
	defer server.Close()

	// Create a checker that uses the test server
	checker := NewChecker()
	checker.goVersionURL = server.URL

	// Test with a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	version, err := checker.GetLatestVersion(ctx)
	if err != nil {
		t.Fatalf("GetLatestVersion() error = %v", err)
	}

	expected := "1.17.1"
	if version != expected {
		t.Errorf("GetLatestVersion() = %v, want %v", version, expected)
	}
}
