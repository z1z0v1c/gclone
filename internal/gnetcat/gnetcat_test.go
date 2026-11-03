package gnetcat

import (
	"testing"
)

func TestNewGnetcat(t *testing.T) {
	tests := []struct {
		name   string
		listen bool
		port   uint16
	}{
		{
			name:   "listening enabled",
			listen: true,
			port:   8080,
		},
		{
			name:   "listening disabled",
			listen: false,
			port:   8080,
		},
		{
			name:   "port zero",
			listen: false,
			port:   0,
		},
		{
			name:   "maximum uint16 port",
			listen: false,
			port:   65535,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGnetcat(tt.listen, tt.port)

			if err != nil {
				t.Fatalf("NewGnetcat() returned unexpected error: %v", err)
			}

			if got == nil {
				t.Fatal("NewGnetcat() returned nil")
			}

			if got.listen != tt.listen {
				t.Errorf("listen = %v, want %v", got.listen, tt.listen)
			}

			if got.port != tt.port {
				t.Errorf("port = %d, want %d", got.port, tt.port)
			}
		})
	}
}
