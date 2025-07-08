package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gocker/internal/gocker/image"
	"github.com/z1z0v1c/gocker/pkg/http"
)

var Pull = &cobra.Command{
	Use:                   "pull image",
	Short:                 "Pull an image from Docker Hub",
	Long:                  "Pull an image from Docker Hub",
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	Run:                   pull,
}

func pull(c *cobra.Command, args []string) {
	imgName := args[0]
	httpClient := http.NewHttpClient()

	img := image.NewImagePuller(imgName, httpClient)

	if err := img.Pull(); err != nil {
		fmt.Printf("Error while pulling %q image: %v\n", imgName, err)
		os.Exit(1)
	}
}

func fatalf(format string, a ...any) {
	fmt.Printf(format, a...)
	os.Exit(1)
}
