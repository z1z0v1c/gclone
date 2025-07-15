package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// gurl is the root Cobra command for the gURL
var gurl = &cobra.Command{
	Use:   "gurl command [flags]",
	Short: "Simple cURL clone",
}

func main() {
	if err := gurl.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
