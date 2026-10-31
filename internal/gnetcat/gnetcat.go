package gnetcat

import (
	"fmt"
	"io"
	"net"
	"os"
)

type Gnetcat struct {
	// Flags
	listen bool
	port   int16
}

func NewGnetcat(listen bool, port int16) (*Gnetcat, error) {
	return &Gnetcat{
		listen: listen,
		port:   port,
	}, nil
}

func (g *Gnetcat) Run() error {
	if g.listen {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", g.port))
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
	}

	return nil
}