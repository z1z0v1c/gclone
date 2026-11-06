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
	zeroio   bool
}

func NewGnetcat(listen bool, port uint16, protocol string, zeroio bool) (*Gnetcat, error) {
	return &Gnetcat{
		listen:   listen,
		port:     port,
		protocol: protocol,
		zeroio:   zeroio,
	}, nil
}

func (g *Gnetcat) Run() error {
	if g.listen {
		return g.Listen()
	}

	return nil
}

func (g *Gnetcat) Listen() error {
	switch g.protocol {
	case "tcp":
		return g.ListenTCP()
	case "udp":
		return g.ListenUDP()
	default:
		return fmt.Errorf("unsupported protocol: %s", g.protocol)
	}
}

func (g *Gnetcat) ListenTCP() error {
	listener, err := net.Listen(g.protocol, fmt.Sprintf(":%d", g.port))
	if err != nil {
		return err
	}
	defer listener.Close()

	return g.serveTCP(listener, os.Stdout)
}

func (g *Gnetcat) serveTCP(listener net.Listener, output io.Writer) error {
	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("accept error: %w", err)
	}
	defer conn.Close()

	_, err = io.Copy(output, conn)

	return err
}

func (g *Gnetcat) ListenUDP() error {
	addr := net.UDPAddr{
		Port: int(g.port),
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP(g.protocol, &addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	return g.serveUDP(conn, os.Stdout)
}

func (g *Gnetcat) serveUDP(conn *net.UDPConn, output io.Writer) error {
    buf := make([]byte, 4096)

    for {
        n, _, err := conn.ReadFromUDP(buf)
        if err != nil {
            return err
        }

        if _, err := output.Write(buf[:n]); err != nil {
            return err
        }
    }
}
