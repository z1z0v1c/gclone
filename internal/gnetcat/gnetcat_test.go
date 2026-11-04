package gnetcat

import (
	"testing"
)

func TestNewGnetcat(t *testing.T) {
	tests := []struct {
		name     string
		listen   bool
		port     uint16
		protocol string
	}{
		{
			name:     "listening enabled",
			listen:   true,
			port:     8080,
			protocol: "tcp",
		},
		{
			name:     "listening disabled",
			listen:   false,
			port:     8080,
			protocol: "tcp",
		},
		{
			name:     "port zero",
			listen:   false,
			port:     0,
			protocol: "tcp",
		},
		{
			name:     "maximum uint16 port",
			listen:   true,
			port:     65535,
			protocol: "tcp",
		},
		{
			name:     "UDP protocol",
			listen:   true,
			port:     8080,
			protocol: "udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGnetcat(tt.listen, tt.port, tt.protocol)

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

			if got.protocol != tt.protocol {
				t.Errorf("protocol = %s, want %s", got.protocol, tt.protocol)
			}
		})
	}
}
