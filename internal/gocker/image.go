package gocker

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/z1z0v1c/gocker/pkg/http"
)

const (
	registry        = "registry-1.docker.io"
	authBaseURL     = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
	manifestBaseURL = "https://%s/v2/%s/manifests/"
	blobsBaseURL    = "https://%s/v2/%s/blobs/"
)

var (
	authURL     string
	manifestURL string
	blobsURL    string
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

	HttpClient *http.Client
}

func NewImage(name string, httpClient *http.Client) *Image {
	tag := "latest"
	homeDir := os.Getenv("HOME")

	imgPath := filepath.Join(homeDir, relativeImagesPath, name)
	imgRoot := filepath.Join(imgPath, "rootfs")
	cfgPath := filepath.Join(imgPath, ".config.json")
	repository := filepath.Join("library", name)

	authURL = fmt.Sprintf(authBaseURL, repository)
	manifestURL = fmt.Sprintf(manifestBaseURL, registry, repository)
	blobsURL = fmt.Sprintf(blobsBaseURL, registry, repository)

	return &Image{
		Name:       name,
		Tag:        tag,
		Path:       imgPath,
		Root:       imgRoot,
		CfgPath:    cfgPath,
		Repository: repository,
		HttpClient: httpClient,
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

	if err := i.fetchConfig(); err != nil {
		return err
	}

	fmt.Printf("Image %q pulled successfully to %q\n", i.Name, i.Path)

	return nil
}

func (i *Image) authenticate() error {
	fmt.Printf("Authenticating with: %s\n", authURL)

	var authResp AuthResponse
	i.HttpClient.SendRequestAndDecode(&authResp, http.MethodGet, authURL, nil)

	fmt.Printf("Authentication successful, token length: %d\n", len(authResp.Token))

	i.Token = authResp.Token

	return nil
}

func (i *Image) fetchManifest() error {
	headers := make(map[string]string, 1)
	headers["Authorization"] = "Bearer " + i.Token
	headers["Accept"] = "application/vnd.docker.distribution.manifest.v2+json"

	resp, err := i.HttpClient.SendRequest(http.MethodGet, manifestURL+i.Tag, headers)
	if err != nil {
		return fmt.Errorf("failed to download layer: %v", err)
	}
	defer resp.Body.Close()

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

func (i *Image) fetchManifestByDigest(digest string) error {
	fmt.Printf("Fetching platform-specific manifest: %s", digest)

	headers := make(map[string]string, 1)
	headers["Authorization"] = "Bearer " + i.Token
	headers["Accept"] = "application/vnd.docker.distribution.manifest.v2+json"

	i.HttpClient.SendRequestAndDecode(i.Manifest, http.MethodGet, manifestURL+digest, headers)

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
	headers := make(map[string]string, 1)
	headers["Authorization"] = "Bearer " + i.Token

	resp, err := i.HttpClient.SendRequest(http.MethodGet, blobsURL+digest, headers)
	if err != nil {
		return fmt.Errorf("failed to download layer: %v", err)
	}
	defer resp.Body.Close()

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

func (i *Image) fetchConfig() error {
	digest := i.Manifest.Config.Digest

	fmt.Printf("Downloading config: %s\n", digest)

	headers := make(map[string]string, 1)
	headers["Authorization"] = "Bearer " + i.Token

	i.Cfg = &ImageConfig{}
	i.HttpClient.SendRequestAndDecode(i.Cfg, http.MethodGet, blobsURL+digest, headers)

	cfgData, err := json.MarshalIndent(i.Cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Save config data
	if err := os.WriteFile(i.CfgPath, cfgData, 0644); err != nil {
		return fmt.Errorf("failed to save config file: %v", err)
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
