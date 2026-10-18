package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	listen bool
	port   int
)

// gnc is the root Cobra command for the Gnetcat utility
var gnc = &cobra.Command{
	Use:   "gnc command [flags]",
	Short: "Simple Netcat clone written in Go",
}

func init() {
	gnc.PersistentFlags().BoolVarP(&listen, "listen", "l", false, "Bind and listen for incoming connections")
	gnc.PersistentFlags().IntVarP(&port, "port", "p", 8888, "Port to listen on or connect to")
}

func main() {
	if err := gnc.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}

	if listen {
		fmt.Println("Listening on port", port)
	}
}
