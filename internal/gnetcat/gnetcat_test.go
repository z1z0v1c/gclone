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
		zeroio   bool
	}{
		{
			name:     "listening enabled",
			listen:   true,
			port:     8080,
			protocol: "tcp",
			zeroio:   false,
		},
		{
			name:     "listening disabled",
			listen:   false,
			port:     8080,
			protocol: "tcp",
			zeroio:   false,
		},
		{
			name:     "port zero",
			listen:   false,
			port:     0,
			protocol: "tcp",
			zeroio:   false,
		},
		{
			name:     "maximum uint16 port",
			listen:   true,
			port:     65535,
			protocol: "tcp",
			zeroio:   false,
		},
		{
			name:     "UDP protocol",
			listen:   true,
			port:     8080,
			protocol: "udp",
			zeroio:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGnetcat(tt.listen, tt.port, tt.protocol, tt.zeroio)

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

			if got.zeroio != tt.zeroio {
				t.Errorf("zeroio = %v, want %v", got.zeroio, tt.zeroio)
			}
		})
	}
}

func TestListenUnsupportedProtocol(t *testing.T) {
    g := &Gnetcat{
        protocol: "icmp",
        port:     8888,
    }

    err := g.Listen()

    if err == nil {
        t.Fatal("expected error for unsupported protocol")
    }

    want := "unsupported protocol: icmp"
    if err.Error() != want {
        t.Errorf("got %q, want %q", err.Error(), want)
    }
}
