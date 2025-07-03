package cmd

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	registry = "registry-1.docker.io"
	tag      = "latest"
	auth     = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
)

var repository string

var pull = &cobra.Command{
	Use:                "pull [image]",
	Short:              "Pull an image from Docker Hub",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run:                Pull,
}

func Pull(c *cobra.Command, args []string) {
	repository = filepath.Join("library", args[0])
	imagePath := filepath.Join(os.Getenv("HOME"), relativeImagesPath, args[0])

	log.Printf("Pulling %q image from the %q repository in %q registry...", args[0], repository, registry)

	token, err := authenticate(repository)
	if err != nil {
		log.Fatalf("authentication failed: %v", err)
	}

	var manifest Manifest
	must(fetchManifest(&manifest, token), "failed to fetch manifest")

	must(os.RemoveAll(imagePath), "failed to remove existing rootfs")

	// Create rootfs directory
	must(os.MkdirAll(filepath.Join(imagePath, "rootfs"), 0755), "failed to create rootfs directory")

	for i, layer := range manifest.Layers {
		log.Printf("Downloading layer %d/%d: %s\n", i+1, len(manifest.Layers), layer.Digest)

		imageRoot := filepath.Join(imagePath, "rootfs")
		must(downloadAndExtractLayer(imageRoot, layer.Digest, token), "failed to download layer")
	}

	fmt.Printf("Downloading config: %s\n", manifest.Config.Digest)

	config, err := fetchConfig(registry, repository, manifest.Config.Digest, token)
	if err != nil {
		log.Fatalf("failed to fetch config: %v", err)
	}

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal config: %v", err)
	}

	// Save config data
	configPath := filepath.Join(imagePath, ".config.json")
	must(os.WriteFile(configPath, configData, 0644), "failed to write config")

	fmt.Printf("Image %q pulled successfully to %q\n", args[0], imagePath)
}

func authenticate(repository string) (string, error) {
	// For Docker Hub, we need to get a token from auth.docker.io
	authURL := fmt.Sprintf(auth, repository)

	log.Printf("Authenticating with: %s\n", authURL)

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

	log.Printf("Authentication successful, token length: %d\n", len(authResp.Token))

	return authResp.Token, nil
}

func fetchManifest(manifest *Manifest, token string) error {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repository, tag)

	log.Printf("Fetching manifest from: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("User-Agent", "gocker/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch manifest with status: %d", resp.StatusCode)
	}

	ctype := resp.Header.Get("Content-Type")

	// Handle OCI Index (manifest list)
	if ctype == "application/vnd.oci.image.index.v1+json" || ctype == "application/vnd.docker.distribution.manifest.list.v2+json" {
		var index ManifestIndex
		if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
			return fmt.Errorf("error decoding manifest index: %w", err)
		}

		log.Printf("Received index, contains %d platform manifests", len(index.Manifests))

		for _, m := range index.Manifests {
			if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" {
				log.Printf("Selected manifest digest for linux/amd64: %s", m.Digest)

				return fetchManifestByDigest(manifest, m.Digest, token)
			}
		}

		return fmt.Errorf("no matching platform found in manifest index")
	}

	if err := json.NewDecoder(resp.Body).Decode(manifest); err != nil {
		return err
	}

	log.Printf("Found %d layers to download\n", len(manifest.Layers))

	return nil
}

func fetchManifestByDigest(manifest *Manifest, digest, token string) error {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repository, digest)

	log.Printf("Fetching platform-specific manifest: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("User-Agent", "gocker/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch manifest by digest, status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(manifest); err != nil {
		return err
	}

	return nil
}

func downloadAndExtractLayer(rootfs, digest, token string) error {
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download layer with status: %d", resp.StatusCode)
	}

	// Verify digest
	hasher := sha256.New()
	reader := io.TeeReader(resp.Body, hasher)

	// Create gzip reader
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		targetPath := filepath.Join(rootfs, header.Name)

		// Security check: prevent path traversal
		if !strings.HasPrefix(targetPath, rootfs) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", targetPath, err)
			}
		case tar.TypeReg:
			// Create directory for file if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %v", targetPath, err)
			}

			// Create and write file
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %v", targetPath, err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file %s: %v", targetPath, err)
			}
			file.Close()
		case tar.TypeSymlink:
			// Create symbolic link
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				// Ignore if symlink already exists
				if !os.IsExist(err) {
					return fmt.Errorf("failed to create symlink %s: %v", targetPath, err)
				}
			}
		case tar.TypeLink:
			// Create hard link
			linkTarget := filepath.Join(rootfs, header.Linkname)
			if err := os.Link(linkTarget, targetPath); err != nil {
				// Ignore if link already exists
				if !os.IsExist(err) {
					return fmt.Errorf("failed to create hard link %s: %v", targetPath, err)
				}
			}
		}
	}

	// Verify the digest
	actualDigest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if actualDigest != digest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", digest, actualDigest)
	}

	return nil
}

func fetchConfig(registry, repository, digest, token string) (*ImageConfig, error) {
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch config with status: %d", resp.StatusCode)
	}

	var config ImageConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
