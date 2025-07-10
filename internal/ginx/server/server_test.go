package server

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestServerIntegration(t *testing.T) {
	s, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	// Create test files
	indexContent := "<html><body>Index Page</body></html>"
	testContent := "<html><body>Test Page</body></html>"

	err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte(indexContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create index.html: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "test.html"), []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test.html: %v", err)
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer ln.Close()

	// Start server in a goroutine
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handleConnection(conn)
		}
	}()

	// Get the actual port
	addr := ln.Addr().(*net.TCPAddr)
	port := addr.Port

	// Test cases
	tests := []struct {
		name           string
		request        string
		expectedStatus string
		expectedBody   string
	}{
		{
			name:           "GET index page",
			request:        "GET / HTTP/1.1\r\n\r\n",
			expectedStatus: "200 OK",
			expectedBody:   indexContent,
		},
		{
			name:           "GET test page",
			request:        "GET /test.html HTTP/1.1\r\n\r\n",
			expectedStatus: "200 OK",
			expectedBody:   testContent,
		},
		{
			name:           "GET non-existent page",
			request:        "GET /nonexistent.html HTTP/1.1\r\n\r\n",
			expectedStatus: "404 Not Found",
		},
		{
			name:           "POST request (not allowed)",
			request:        "POST / HTTP/1.1\r\n\r\n",
			expectedStatus: "405 Method Not Allowed",
		},
		{
			name:           "Directory traversal attempt",
			request:        "GET /../etc/passwd HTTP/1.1\r\n\r\n",
			expectedStatus: "403 Forbidden",
		},
		{
			name:           "Malformed request",
			request:        "GET\r\n\r\n",
			expectedStatus: "400 Bad Request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Connect to server
			conn, err := net.Dial("tcp", "0.0.0.0:"+strconv.Itoa(port))
			if err != nil {
				t.Fatalf("Failed to connect to server: %v", err)
			}
			defer conn.Close()

			// Send request
			_, err = conn.Write([]byte(tt.request))
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}

			// Read response
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			reader := bufio.NewReader(conn)
			header, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("Failed to read response header: %v", err)
			}

			// Check status
			if !strings.Contains(header, tt.expectedStatus) {
				t.Errorf("Expected status %s in response, got: %s", tt.expectedStatus, header)
			}

			// Check body for successful requests
			if tt.expectedBody != "" {
				// Throw away first empty line
				_, _ = reader.ReadString('\n')
				body, err := reader.ReadString('\n')
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}
				if !strings.Contains(body, tt.expectedBody) {
					t.Errorf("Expected body to contain %s, got: %s", tt.expectedBody, header)
				}
			}
		})
	}
}
