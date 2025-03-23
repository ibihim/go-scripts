// Package gotools provides functionality for managing Go installations
package gotools

import (
	"io"
	"net"
	"net/http"
	"time"
)

const (
	interval  = 3 * time.Second
	timeout   = 1 * time.Minute
	immediate = true
)

// DefaultTimeouts defines sensible defaults for HTTP operations
var DefaultTimeouts = struct {
	// Connect timeout limits the time spent establishing a TCP connection
	Connect time.Duration
	// TLSHandshake limits the time spent performing the TLS handshake
	TLSHandshake time.Duration
	// ResponseHeader limits the time spent waiting for the server's response headers
	ResponseHeader time.Duration
	// Request limits the time for the entire request (not including body read)
	Request time.Duration
	// IdleConnection limits how long connections stay in the pool
	IdleConnection time.Duration
}{
	Connect:        5 * time.Second,
	TLSHandshake:   5 * time.Second,
	ResponseHeader: 10 * time.Second,
	Request:        30 * time.Second,
	IdleConnection: 90 * time.Second,
}

// NewHTTPClient creates a properly configured HTTP client with timeouts
func NewHTTPClient() *http.Client {
	// Create a properly configured transport with reasonable timeouts
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   DefaultTimeouts.Connect,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       DefaultTimeouts.IdleConnection,
		TLSHandshakeTimeout:   DefaultTimeouts.TLSHandshake,
		ResponseHeaderTimeout: DefaultTimeouts.ResponseHeader,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   10, // Default is 2 which is too low
	}

	// Create a client with the configured transport
	return &http.Client{
		Transport: transport,
		Timeout:   DefaultTimeouts.Request, // Overall request timeout
	}
}

func safeClose(body io.Closer) {
	if body != nil {
		body.Close()
	}
}
