package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gclone/internal/gocker/container"
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
	imgName, cmd, args := args[0], args[1], args[2:]

	cn, err := container.NewContainer(imgName, cmd, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during container creation: %v\n", err)

		os.Exit(1)
	}

	if err := cn.Run(); err != nil {
		// Handle exit error for proper exit code propagation
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		} else {
			fmt.Fprintf(os.Stderr, "Error during container excecution: %v\n", err)
			
			os.Exit(1)
		}
	}
}
