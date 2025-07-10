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
	s, tempDir := setupTestServer(t)
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
			path, status, err := s.getAbsPath(tt.inputPath)

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

func TestReadDataFromFile(t *testing.T) {
	s, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	// Create test file
	testContent := "Hello, World!"
	testFile := filepath.Join(tempDir, "test.html")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create test directory
	testDir := filepath.Join(tempDir, "testdir")
	err = os.Mkdir(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name           string
		filePath       string
		expectError    bool
		expectedStatus string
		expectedData   []byte
	}{
		{
			name:         "read existing file",
			filePath:     testFile,
			expectError:  false,
			expectedData: []byte(testContent),
		},
		{
			name:           "file not found",
			filePath:       filepath.Join(tempDir, "nonexisting.html"),
			expectError:    true,
			expectedStatus: "404 Not Found",
		},
		{
			name:           "read directory",
			filePath:       testDir,
			expectError:    true,
			expectedStatus: "403 Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, status, err := s.readDataFromFile(tt.filePath)

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
				if string(data) != string(tt.expectedData) {
					t.Errorf("Expected data %s, got %s", tt.expectedData, data)
				}
			}
		})
	}
}

