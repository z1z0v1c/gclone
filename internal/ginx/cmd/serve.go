package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gclone/internal/ginx/server"
)

var port uint16
var wwwRoot string

var Serve = &cobra.Command{
	Use:   "serve",
	Short: "Serve runs a Ginx instance",
	Run:   serve,
}

func init() {
	Serve.PersistentFlags().Uint16VarP(&port, "port", "p", 80, "Port number")
	Serve.PersistentFlags().StringVarP(&wwwRoot, "root", "r", "./internal/ginx/www", "Root directory")
}

func serve(c *cobra.Command, args []string) {
	s := server.NewServer(port, wwwRoot)

	if err := s.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		os.Exit(1)
	}
}
