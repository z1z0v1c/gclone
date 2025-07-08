package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gocker/internal/gocker/cmd"
)

// gocker is the root Cobra command for the gocker CLI tool.
var gocker = &cobra.Command{
	Use:   "gocker command image [subcommand] [flags]",
	Short: "Simple Docker clone",
}

// init registers the subcommands within the root command.
func init() {
	gocker.AddCommand(cmd.Run, cmd.Pull)
}

func main() {
	if err := gocker.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err); os.Exit(1)
	}
}

