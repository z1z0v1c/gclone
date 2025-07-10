package server

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestServer is a helper func that creates a test server
func setupTestServer(t *testing.T) (*Server, string) {
	tempDir, err := os.MkdirTemp("", "server_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Assign port dinamically
	server, err := NewServer(0, tempDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	return server, tempDir
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		name        string
		port        uint16
		wwwRoot     string
		expectError bool
	}{
		{
			name:        "valid args",
			port:        8080,
			wwwRoot:     ".",
			expectError: false,
		},
		{
			name:        "invalid www root",
			port:        8080,
			wwwRoot:     "./invalid_path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer(tt.port, tt.wwwRoot)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil.")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				} else if s == nil {
					t.Error("Expected server to be created")
				} else if s.port != tt.port {
					t.Errorf("Expected port %d, got %d", tt.port, s.port)
				}
			}
		})
	}
}

func TestGetAbsPath(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name           string
		inputPath      string
		expectError    bool
		expectedStatus string
		expectedPath   string
	}{
		{
			name:         "root becomes index.html",
			inputPath:    "/",
			expectError:  false,
			expectedPath: filepath.Join(tempDir, "index.html"),
		},
		{
			name:         "valid path",
			inputPath:    "/test.html",
			expectError:  false,
			expectedPath: filepath.Join(tempDir, "test.html"),
		},
		{
			name:           "directory traversal",
			inputPath:      "/../etc/passwd",
			expectError:    true,
			expectedStatus: "403 Forbidden",
		},
		{
			name:           "complex directory traversal",
			inputPath:      "/test/../../../etc/passwd",
			expectError:    true,
			expectedStatus: "403 Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, status, err := server.getAbsPath(tt.inputPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if status != tt.expectedStatus {
					t.Errorf("Expected status %s, got %s", tt.expectedStatus, status)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if path != tt.expectedPath {
					t.Errorf("Expected path %s, got %s", tt.expectedPath, path)
				}
			}
		})
	}
}
