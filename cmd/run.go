package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var run = &cobra.Command{
  Use: "run",
  Short: "Equivalent to Docker run subcommand",
  Run: func(cmd *cobra.Command, args []string) {
    fmt.Println("Run subcommand is called.")
  },  
}
