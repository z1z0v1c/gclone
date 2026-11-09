package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	g "github.com/z1z0v1c/gclone/internal/gnetcat"
)

var (
	listen bool
	port   uint16
	udp    bool
	zeroio bool
)

// gnc is the root Cobra command for the Gnetcat utility
var gnc = &cobra.Command{
	Use:   "gnc command [flags]",
	Short: "Simple Netcat clone written in Go",
	Run:   run,
}

func init() {
	gnc.PersistentFlags().BoolVarP(&listen, "listen", "l", false, "Bind and listen for incoming connections")
	gnc.PersistentFlags().Uint16VarP(&port, "port", "p", 8888, "Specify source port to use")
	gnc.PersistentFlags().BoolVarP(&udp, "udp", "u", false, "Use UDP instead of TCP")
	gnc.PersistentFlags().BoolVarP(&zeroio, "zeroio", "z", false, "Use zero I/O mode")
}

func run(c *cobra.Command, args []string) {
	var host string
	var port uint16

	if len(args) > 0 {
		host = args[0]
	}

	if len(args) > 1 {
		p, err := strconv.ParseUint(args[1], 10, 16)
		if err != nil || p < 1 || p > 65535 {
			fmt.Fprintf(os.Stderr, "Error: invalid port number")
			os.Exit(1)
		}

		port = uint16(p)
	}

	protocol := "tcp"
	if udp {
		protocol = "udp"
	}

	g, err := g.NewGnetcat(listen, host, port, protocol, zeroio)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}

	err = g.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}

func main() {
	gnc.Execute()
}
