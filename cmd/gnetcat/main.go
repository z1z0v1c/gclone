package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	g "github.com/z1z0v1c/gclone/internal/gnetcat"
)

var (
	listen bool
	port   uint16
	udp    bool
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
}

func run(c *cobra.Command, args []string) {
	protocol := "tcp"
	if udp {
		protocol = "udp"
	}

	g, err := g.NewGnetcat(listen, port, protocol)
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
