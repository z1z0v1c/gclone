package server

import (
	"testing"
)

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
			name:    "invalid www root",
			port:    8080,
			wwwRoot: "./invalid_path",
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
