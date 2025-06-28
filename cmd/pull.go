package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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

	log.Printf("Pulling %q image from the %q repository in %q registry...", image, repository, registry)

	log.Printf("Authenticating for the %q repository...", repository)

	token, err := authenticate(repository)
	if err != nil {
		log.Fatalf("authentication failed: %v", err)
	}

	log.Printf("Token: %s", token)
}

func authenticate(repository string) (string, error) {
	// For Docker Hub, we need to get a token from auth.docker.io
	authURL := fmt.Sprintf(auth, repository)

	resp, err := http.Get(authURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}

	return authResp.Token, nil
}
