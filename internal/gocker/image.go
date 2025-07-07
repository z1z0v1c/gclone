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
	// Token      string

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
		return err
	}

	if err := i.fetchManifest(); err != nil {
		return err
	}

	must(os.RemoveAll(i.Path), "Failed to remove existing image")

	// Create rootfs directory
	must(os.MkdirAll(i.Root, 0755), "Failed to create image rootfs directory")

	for j, layer := range i.Manifest.Layers {
		fmt.Printf("Downloading layer %d/%d: %s\n", j+1, len(i.Manifest.Layers), layer.Digest)

		must(downloadAndExtractLayer(i.Root, layer.Digest), "Failed to download layer")
	}

	fmt.Printf("Downloading config: %s\n", i.Manifest.Config.Digest)

	var cfg ImageConfig
	must(fetchConfig(&cfg, i.Manifest.Config.Digest), "Failed to fetch config")

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

	// i.Token = authResp.Token
	token = authResp.Token

	return nil
}

func (i *Image) fetchManifest() error {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, i.Repository, i.Tag)

	fmt.Printf("Fetching manifest from: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	setRequestHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch manifest with status: %s", resp.Status)
	}

	ctype := resp.Header.Get("Content-Type")

	// Handle OCI Index (manifest list)
	if ctype == "application/vnd.oci.image.index.v1+json" || ctype == "application/vnd.docker.distribution.manifest.list.v2+json" {
		var index ManifestIndex
		if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
			return fmt.Errorf("error decoding manifest index: %w", err)
		}

		fmt.Printf("Received index, contains %d platform manifests\n", len(index.Manifests))

		for _, m := range index.Manifests {
			if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" {
				fmt.Printf("Selected manifest digest for linux/amd64: %s", m.Digest)

				return i.fetchManifestByDigest(m.Digest)
			}
		}

		return fmt.Errorf("no matching platform found in manifest index")
	}

	if err := json.NewDecoder(resp.Body).Decode(i.Manifest); err != nil {
		return err
	}

	fmt.Printf("Found %d layers to download\n", len(i.Manifest.Layers))

	return nil
}

func setRequestHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("User-Agent", "gocker/1.0")
}

func (i *Image) fetchManifestByDigest(digest string) error {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repository, digest)

	fmt.Printf("Fetching platform-specific manifest: %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	setRequestHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch manifest by digest, status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(i.Manifest); err != nil {
		return err
	}

	return nil
}