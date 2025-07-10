package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gclone/internal/ginx/server"
)

var (
	port    uint16
	wwwRoot string
)

var Start = &cobra.Command{
	Use:   "start [flags]",
	Short: "Start starts a Ginx instance",
	Run:   start,
}

func init() {
	Start.PersistentFlags().Uint16VarP(&port, "port", "p", 80, "Port number")
	Start.PersistentFlags().StringVarP(&wwwRoot, "root", "r", "./internal/ginx/www", "Root directory")
}

func start(c *cobra.Command, args []string) {
	s, err := server.NewServer(port, wwwRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	if err := s.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}
