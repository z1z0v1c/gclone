package gocker

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var PullCmd = &cobra.Command{
	Use:                   "pull [image]",
	Short:                 "Pull an image from Docker Hub",
	Long:                  "Pull an image from Docker Hub",
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	Run:                   pull,
}

func pull(c *cobra.Command, args []string) {
	imgName := args[0]

	img := NewImage(imgName)

	if err := img.pull(); err != nil {
		fmt.Printf("Error while pulling %q image: %v\n", img.Name, err)
		os.Exit(1)
	}
}

func fatalf(format string, a ...any) {
	fmt.Printf(format, a...)
	os.Exit(1)
}
