package gocker

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	repository string
	token      string
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

	repository = filepath.Join("library", imgName)

	imgPath := filepath.Join(os.Getenv("HOME"), relativeImagesPath, imgName)
	imgRoot := filepath.Join(imgPath, "rootfs")
	cfgPath := filepath.Join(imgPath, ".config.json")

	fmt.Printf("Pulling %q image from the %q repository in %q registry...\n", img.Name, repository, registry)

	must(authenticate(), "Authentication failed")

	var mf Manifest
	must(fetchManifest(&mf), "Failed to fetch manifest")

	must(os.RemoveAll(imgPath), "Failed to remove existing image")

	// Create rootfs directory
	must(os.MkdirAll(imgRoot, 0755), "Failed to create image rootfs directory")

	for i, layer := range mf.Layers {
		fmt.Printf("Downloading layer %d/%d: %s\n", i+1, len(mf.Layers), layer.Digest)

		must(downloadAndExtractLayer(imgRoot, layer.Digest), "Failed to download layer")
	}

	fmt.Printf("Downloading config: %s\n", mf.Config.Digest)

	var cfg ImageConfig
	must(fetchConfig(&cfg, mf.Config.Digest), "Failed to fetch config")

	cfgData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fatalf("Failed to marshal config: %v", err)
	}

	// Save config data
	must(os.WriteFile(cfgPath, cfgData, 0644), "Failed to write config")

	fmt.Printf("Image %q pulled successfully to %q\n", args[0], imgPath)
}

func fatalf(format string, a ...any) {
	fmt.Printf(format, a...)
	os.Exit(1)
}

func setRequestHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("User-Agent", "gocker/1.0")
}

func authenticate() error {
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

func fetchManifest(mf *Manifest) error {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repository, tag)

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
		return fmt.Errorf("failed to fetch manifest with status: %d", resp.StatusCode)
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

				return fetchManifestByDigest(mf, m.Digest)
			}
		}

		return fmt.Errorf("no matching platform found in manifest index")
	}

	if err := json.NewDecoder(resp.Body).Decode(mf); err != nil {
		return err
	}

	fmt.Printf("Found %d layers to download\n", len(mf.Layers))

	return nil
}

func fetchManifestByDigest(mf *Manifest, digest string) error {
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

	if err := json.NewDecoder(resp.Body).Decode(mf); err != nil {
		return err
	}

	return nil
}

func downloadAndExtractLayer(imgRoot, digest string) error {
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, repository, digest)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
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
	gr, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	// Create tar reader
	tr := tar.NewReader(gr)

	// Extract files
	extractLayer(tr, imgRoot)

	// Verify the digest
	actualDigest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if actualDigest != digest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", digest, actualDigest)
	}

	return nil
}

func extractLayer(tr *tar.Reader, imgRoot string) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		targetPath := filepath.Join(imgRoot, header.Name)

		// Security check: prevent path traversal
		if !strings.HasPrefix(targetPath, imgRoot) {
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

			if _, err := io.Copy(file, tr); err != nil {
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
			linkTarget := filepath.Join(imgRoot, header.Linkname)
			if err := os.Link(linkTarget, targetPath); err != nil {
				// Ignore if link already exists
				if !os.IsExist(err) {
					return fmt.Errorf("failed to create hard link %s: %v", targetPath, err)
				}
			}
		}
	}

	return nil
}

func fetchConfig(config *ImageConfig, digest string) error {
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch config with status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return err
	}

	return nil
}
