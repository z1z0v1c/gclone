package gocker

import (
	"github.com/spf13/cobra"
)

var RunCmd = &cobra.Command{
	Use:                "run image command [flags]",
	Short:              "Run a container from a downloaded image",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                run,
}

func run(c *cobra.Command, args []string) {
	cnt, err := NewContainer(args)
	if err != nil {
		fatalf("Error during container creation: %v\n", err)
	}

	cnt.run()
}
