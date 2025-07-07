package gocker

import (
	"os"

	"github.com/spf13/cobra"
)

var RunCmd = &cobra.Command{
	Use:                "run image command [flags]",
	Short:              "Run a container from a downloaded image",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                run,
}

func must(err error, errMsg string) {
	if err != nil {
		fatalf(errMsg+": %v\n", err)
		os.Exit(1)
	}
}

func run(c *cobra.Command, args []string) {
	cnt, err := NewContainer(args)
	if err != nil {
		fatalf("Error during container creation: %v\n", err)
	}

	cnt.run()
}
