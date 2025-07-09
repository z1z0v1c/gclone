package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gclone/internal/ginx/server"
)

var Serve = &cobra.Command{
	Use:   "serve",
	Short: "Serve runs a Ginx instance",
	Run:   serve,
}

func serve(c *cobra.Command, args []string) {
	port := 80
	wwwRoot := "./internal/ginx/www"

	s := server.NewServer(port, wwwRoot)

	if err := s.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		os.Exit(1)
	}
}
