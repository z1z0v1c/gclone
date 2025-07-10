package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gclone/internal/ginx/cmd"
)

// ginx is the root Cobra command for the Ginx Web Server
var ginx = &cobra.Command{
	Use:   "ginx command [flags]",
	Short: "Simple Web Server",
}

// init registers the subcommands within the root command.
func init() {
	ginx.AddCommand(cmd.Start)
}

func main() {
	if err := ginx.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
