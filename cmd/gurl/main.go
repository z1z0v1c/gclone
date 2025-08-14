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
	data    string
	header  string
)

// gurl is the root Cobra command for gURL
var gurl = &cobra.Command{
	Use:   "gurl command [flags]",
	Short: "Simple cURL clone",
	Args:  cobra.ExactArgs(1),
	Run:   run,
}

func init() {
	gurl.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Make the operation more talkative")
	gurl.PersistentFlags().StringVarP(&method, "request", "X", "GET", "Change the method to use when starting the transfer")
	gurl.PersistentFlags().StringVarP(&data, "data", "d", "", "Sends the specified data in a POST request to the HTTP server")
	gurl.PersistentFlags().StringVarP(&header, "header", "H", "", "Extra header to include in information sent")
}

func run(c *cobra.Command, args []string) {
	g, err := g.NewGurl(args[0], verbose, method, data, header)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	err = g.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}

func main() {
	gurl.Execute()
}
