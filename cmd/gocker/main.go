package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	g "github.com/z1z0v1c/gocker/internal/gocker"
)

var gocker = &cobra.Command{
	Use:   "gocker command image [subcommand] [flags]",
	Short: "Simple Docker clone",
}

func init() {
	gocker.AddCommand(g.RunCmd, g.PullCmd)
}

func main() {
	if err := gocker.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
