package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gclone/internal/gurl"
)

// root is the root Cobra command for gURL
var root = &cobra.Command{
	Use:   "gurl command [flags]",
	Short: "Simple cURL clone",
	Args:  cobra.ExactArgs(1),
	Run:   start,
}

func start(c *cobra.Command, args []string) {
	gurl.NewGurl(args[0]).Start()
}

func main() {
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
