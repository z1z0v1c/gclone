package gocker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

const (
	registry    = "registry-1.docker.io"
	tag         = "latest"
	authBaseURL = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
)

type Image struct {
	Name       string
	Tag        string
	Path       string
	Root       string
	CfgPath    string
	Repository string
	Token      string

	Manifest *Manifest
	Config   *ImageConfig
}

func NewImage(name string) *Image {
	home := os.Getenv("HOME")

	imgPath := filepath.Join(home, relativeImagesPath, name)
	imgRoot := filepath.Join(imgPath, "rootfs")
	cfgPath := filepath.Join(imgPath, ".config.json")
	repository := filepath.Join("library", name)

	return &Image{
		Name:       name,
		Tag:        "latest",
		Path:       imgPath,
		Root:       imgRoot,
		CfgPath:    cfgPath,
		Repository: repository,
	}
}

func (i *Image) pull() error {
	fmt.Printf("Pulling %q image from the %q repository in %q registry...\n",
		i.Name, i.Repository, registry)

	if err := i.authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	var mf Manifest
	must(fetchManifest(&mf), "Failed to fetch manifest")

	must(os.RemoveAll(i.Path), "Failed to remove existing image")

	// Create rootfs directory
	must(os.MkdirAll(i.Root, 0755), "Failed to create image rootfs directory")

	for j, layer := range mf.Layers {
		fmt.Printf("Downloading layer %d/%d: %s\n", j+1, len(mf.Layers), layer.Digest)

		must(downloadAndExtractLayer(i.Root, layer.Digest), "Failed to download layer")
	}

	fmt.Printf("Downloading config: %s\n", mf.Config.Digest)

	var cfg ImageConfig
	must(fetchConfig(&cfg, mf.Config.Digest), "Failed to fetch config")

	cfgData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fatalf("Failed to marshal config: %v", err)
	}

	// Save config data
	must(os.WriteFile(i.CfgPath, cfgData, 0644), "Failed to write config")

	fmt.Printf("Image %q pulled successfully to %q\n", i.Name, i.Path)

	return nil
}

func (i *Image) authenticate() error {
	// For Docker Hub, we need to get a token from auth.docker.io
	authURL := fmt.Sprintf(authBaseURL, repository)

	fmt.Printf("Authenticating with: %s\n", authURL)

	resp, err := http.Get(authURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}

	fmt.Printf("Authentication successful, token length: %d\n", len(authResp.Token))

	token = authResp.Token

	return nil
}
