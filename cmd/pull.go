package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

const (
	registry = "registry-1.docker.io."
	tag      = "latest"
	auth     = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
)

type AuthResponse struct {
	Token string `json:"token"`
}

var pull = &cobra.Command{
	Use:                "pull",
	Short:              "Pull an image from Docker Hub",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                Pull,
}

func Pull(c *cobra.Command, args []string) {
	image := args[0]
	repository := "library/" + image

	log.Printf("Pulling %s image from the %s repository in %s registry.", image, repository, registry)
}
