package cmd

import (
	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gocker/internal/gocker/container"
)

var Run = &cobra.Command{
	Use:                "run image command [flags]",
	Short:              "Run a container from a downloaded image",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                run,
}

func run(c *cobra.Command, args []string) {
	cnt, err := container.NewContainer(args)
	if err != nil {
		fatalf("Error during container creation: %v\n", err)
	}

	cnt.Run()
}
