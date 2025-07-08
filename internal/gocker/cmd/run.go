package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gocker/internal/gocker/container"
)

// Run is the Cobra command to launch a container from a previously pulled image.
var Run = &cobra.Command{
	Use:                "run image command [flags]",
	Short:              "Run a container from a downloaded image",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                run,
}

// run is the command handler function that creates and runs the container.
func run(c *cobra.Command, args []string) {
	cn, err := container.NewContainer(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during container creation: %v\n", err); os.Exit(1)
	}

	cn.Run()
}
