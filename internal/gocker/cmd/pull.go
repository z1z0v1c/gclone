package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z1z0v1c/gocker/internal/gocker/image"
	"github.com/z1z0v1c/gocker/pkg/http"
)

// Pull is the Cobra command for pulling a container image from Docker Hub.
var Pull = &cobra.Command{
	Use:                   "pull image",
	Short:                 "Pull an image from Docker Hub",
	Long:                  "Pull an image from Docker Hub and extract it into local image storage",
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),
	Run:                   pull,
}

// pull is the command handler function that pulls the image.
func pull(c *cobra.Command, args []string) {
	imgName := args[0]
	httpClient := http.NewHttpClient()

	img := image.NewImagePuller(imgName, httpClient)

	if err := img.Pull(); err != nil {
		fmt.Printf("Error while pulling %q image: %v\n", imgName, err); os.Exit(1)
	}
}
