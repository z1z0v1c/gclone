package gnetcat

import (
	"fmt"
	"io"
	"net"
	"os"
)

type Gnetcat struct {
	// Flags
	listen   bool
	port     uint16
	protocol string
}

func NewGnetcat(listen bool, port uint16, protocol string) (*Gnetcat, error) {
	return &Gnetcat{
		listen:   listen,
		port:     port,
		protocol: protocol,
	}, nil
}

func (g *Gnetcat) Run() error {
	if g.listen {
		switch g.protocol {
		case "tcp":
			listener, err := net.Listen(g.protocol, fmt.Sprintf(":%d", g.port))
			if err != nil {
				panic(err)
			}
			defer listener.Close()

			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Accept error:", err)
			}
			defer conn.Close()

			_, _ = io.Copy(os.Stdout, conn)

		case "udp":
			addr := net.UDPAddr{
				Port: int(g.port),
				IP:   net.ParseIP("0.0.0.0"),
			}

			conn, err := net.ListenUDP(g.protocol, &addr)
			if err != nil {
				panic(err)
			}
			defer conn.Close()

			_, _ = io.Copy(os.Stdout, conn)
		}
	}

	return nil
}
