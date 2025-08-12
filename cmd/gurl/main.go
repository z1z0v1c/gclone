package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	g "github.com/z1z0v1c/gclone/internal/gurl"
)

var (
	verbose bool
	method  string
)

// gurl is the root Cobra command for gURL
var gurl = &cobra.Command{
	Use:   "gurl command [flags]",
	Short: "Simple cURL clone",
	Args:  cobra.ExactArgs(1),
	Run:   start,
}

func init() {
	gurl.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Make the operation more talkative")
	gurl.PersistentFlags().StringVarP(&method, "request", "X", "GET", "Change the method to use when starting the transfer")
}

func start(c *cobra.Command, args []string) {
	g.NewGurl(args[0], verbose, method).Start()
}

func main() {
	if err := gurl.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
