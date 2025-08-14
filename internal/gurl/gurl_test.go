package gurl

import (
	"strings"
	"testing"
)

func TestNewGurl(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		verbose     bool
		method      string
		data        string
		header      string
		expected    *Gurl
		expectError bool
		errMsg      string
	}{
		{
			name: "valid http url with default port",
			url:  "http://example.com/path",
			expected: &Gurl{
				protocol: "http",
				host:     "example.com",
				port:     "80",
				path:     "/path",
			},
		},
		{
			name: "valid http url with explicit port",
			url:  "http://example.com:8080/path",
			expected: &Gurl{
				protocol: "http",
				host:     "example.com",
				port:     "8080",
				path:     "/path",
			},
		},
		{
			name: "valid http url with empty path",
			url:  "http://example.com",
			expected: &Gurl{
				protocol: "http",
				host:     "example.com",
				port:     "80",
				path:     "/",
			},
		},
		{
			name:        "invalid protocol",
			url:         "https://example.com",
			expectError: true,
			errMsg:      "invalid protocol (only HTTP is supported)",
		},
		{
			name:        "invalid url",
			url:         "http://example.com:abc",
			expectError: true,
			errMsg:      "invalid url",
		},
		{
			name:   "valid method",
			url:    "http://example.com",
			method: "PUT",
			expected: &Gurl{
				verbose: false,
				method:  "PUT",
				data:    "",
				header:  "",
			},
		},
		{
			name:   "invalid method",
			url:    "http://example.com",
			method: "METHOD",
			expected: &Gurl{
				verbose: false,
				method:  "METHOD",
				data:    "",
				header:  "",
			},
		},
		{
			name: "existing data",
			url:  "http://example.com",
			data: "{\"data\": \"exists\"}",
			expected: &Gurl{
				verbose: false,
				method:  "",
				data:    "{\"data\": \"exists\"}",
				header:  "",
			},
		},
		{
			name:   "existing header",
			url:    "http://example.com",
			header: "{\"header\": \"exists\"}",
			expected: &Gurl{
				verbose: false,
				method:  "",
				data:    "",
				header:  "{\"header\": \"exists\"}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := NewGurl(tt.url, tt.verbose, tt.method, tt.data, tt.header)
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expected.protocol != "" && g.protocol != tt.expected.protocol {
				t.Errorf("expected protocol %q, got %q", tt.expected.protocol, g.protocol)
			}

			if tt.expected.host != "" && g.host != tt.expected.host {
				t.Errorf("expected host %q, got %q", tt.expected.host, g.host)
			}

			if tt.expected.port != "" && g.port != tt.expected.port {
				t.Errorf("expected port %q, got %q", tt.expected.port, g.port)
			}

			if tt.expected.path != "" && g.path != tt.expected.path {
				t.Errorf("expected path %q, got %q", tt.expected.path, g.path)
			}

			if g.verbose != tt.expected.verbose {
				t.Errorf("expected verbose %v, got %v", tt.expected.verbose, g.verbose)
			}

			if g.method != tt.expected.method {
				t.Errorf("expected method %q, got %q", tt.expected.method, g.method)
			}

			if g.data != tt.expected.data {
				t.Errorf("expected method %q, got %q", tt.expected.method, g.method)
			}

			if g.header != tt.expected.header {
				t.Errorf("expected method %q, got %q", tt.expected.method, g.method)
			}
		})
	}
}
