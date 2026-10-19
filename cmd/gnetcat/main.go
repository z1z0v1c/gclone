package main

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/spf13/cobra"
)

var (
	listen bool
	port   int16
)

// gnc is the root Cobra command for the Gnetcat utility
var gnc = &cobra.Command{
	Use:   "gnc command [flags]",
	Short: "Simple Netcat clone written in Go",
}

func init() {
	gnc.PersistentFlags().BoolVarP(&listen, "listen", "l", false, "Bind and listen for incoming connections")
	gnc.PersistentFlags().Int16VarP(&port, "port", "p", 8888, "Specify source port to use")
}

func main() {
	if err := gnc.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}

	if listen {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
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
}
