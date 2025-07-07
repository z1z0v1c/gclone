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
	Cfg      *ImageConfig
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

	// Create rootfs directory
	if err := i.mkdirRootfs(); err != nil {
		return err
	}

	if err := i.downloadAndExtract(); err != nil {
		return err
	}

	if err := i.fetchConfig(i.Manifest.Config.Digest); err != nil {
		return err
	}

	cfgData, err := json.MarshalIndent(i.Cfg, "", "  ")
	if err != nil {
		fatalf("Failed to marshal config: %v", err)
	}

	// Save config data
	if err := os.WriteFile(i.CfgPath, cfgData, 0644); err != nil {
		return err
	}

	fmt.Printf("Image %q pulled successfully to %q\n", i.Name, i.Path)

	return nil
}

func (i *Image) authenticate() error {
	// For Docker Hub, we need to get a token from auth.docker.io
	authURL := fmt.Sprintf(authBaseURL, i.Repository)

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

	i.Token = authResp.Token

	return nil
}

func (i *Image) fetchManifest() error {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, i.Repository, i.Tag)

	fmt.Printf("Fetching manifest from: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	i.setRequestHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch manifest with status: %s", resp.Status)
	}

	i.Manifest = &Manifest{}
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

func (i *Image) setRequestHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+i.Token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("User-Agent", "gocker/1.0")
}

func (i *Image) fetchManifestByDigest(digest string) error {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, i.Repository, digest)

	fmt.Printf("Fetching platform-specific manifest: %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	i.setRequestHeaders(req)

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

func (i *Image) downloadAndExtract() error {
	for j, layer := range i.Manifest.Layers {
		fmt.Printf("Downloading layer %d/%d: %s\n", j+1, len(i.Manifest.Layers), layer.Digest)

		if err := i.downloadAndExtractLayer(layer.Digest); err != nil {
			return err
		}
	}

	return nil
}

func (i *Image) downloadAndExtractLayer(digest string) error {
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, i.Repository, digest)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+i.Token)

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
	i.extractLayer(tr, i.Root)

	// Verify the digest
	actualDigest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if actualDigest != digest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", digest, actualDigest)
	}

	return nil
}

func (i *Image) extractLayer(tr *tar.Reader, imgRoot string) error {
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

func (i *Image) fetchConfig(digest string) error {
	fmt.Printf("Downloading config: %s\n", i.Manifest.Config.Digest)

	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, i.Repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+i.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch config with status: %d", resp.StatusCode)
	}

	i.Cfg = &ImageConfig{}
	if err := json.NewDecoder(resp.Body).Decode(i.Cfg); err != nil {
		return err
	}

	return nil
}

func (i *Image) mkdirRootfs() error {
	if err := os.RemoveAll(i.Path); err != nil {
		return fmt.Errorf("failed to remove existing image dir: %v", err)
	}

	// Create rootfs directory
	if err := os.MkdirAll(i.Root, 0755); err != nil {
		return fmt.Errorf("failed to create image rootfs dir: %v", err)
	}

	return nil
}
