package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

// gurl is the root Cobra command for the gURL
var gurl = &cobra.Command{
	Use:   "gurl command [flags]",
	Short: "Simple cURL clone",
	Args:  cobra.ExactArgs(1),
	Run: func(c *cobra.Command, args []string) {
		url, err := url.Parse(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid url: %v.\n", err)
			os.Exit(1)
		}

		protocol := url.Scheme
		if protocol != "http" {
			fmt.Fprintf(os.Stderr, "Invalid protocol. Only http is supported.\n")
			os.Exit(1)
		}

		host := url.Hostname()
		port := url.Port()
		if port == "" {
			port = "80"
		}

		path := url.Path
		if path == "" {
			path = "/"
		}
		
		fmt.Printf("Connecting to %s\n", host)
		fmt.Printf("Sending request GET %s HTTP/1.1\n", path)
		fmt.Printf("Host: %s\n", host)
		fmt.Printf("Accept: */*\n")
	},
}

func main() {
	if err := gurl.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
