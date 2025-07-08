package image

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/z1z0v1c/gclone/internal/gocker/registry"
	"github.com/z1z0v1c/gclone/pkg/http"
)

const (
	// RelativeImagesPath is the relative images path under the user's home directory.
	RelativeImagesPath = ".local/share/gocker/images/"

	authURLBase     = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
	manifestURLBase = "https://%s/v2/%s/manifests/"
	blobsURLBase    = "https://%s/v2/%s/blobs/"
)

var (
	authURL     string
	manifestURL string
	blobsURL    string
)

// Client encapsulates the parameters to pull and unpack an image.
type Client struct {
	imageName  string
	imageTag   string
	imagePath  string
	imageRoot  string
	configPath string
	repository string
	token      string

	manifest *registry.Manifest
	config   *registry.ImageConfig

	httpClient *http.Client

	downloadedLayers map[string][]byte
	mutex            sync.Mutex
}

// NewClient creates and initializes a new ImagePuller for the given image name.
func NewClient(imgName string, httpClient *http.Client) *Client {
	imgTag := "latest"
	homeDir := os.Getenv("HOME")

	imgPath := filepath.Join(homeDir, RelativeImagesPath, imgName)
	imgRoot := filepath.Join(imgPath, "rootfs")
	cfgPath := filepath.Join(imgPath, ".config.json")
	repository := filepath.Join("library", imgName)

	authURL = fmt.Sprintf(authURLBase, repository)
	manifestURL = fmt.Sprintf(manifestURLBase, registry.URL, repository)
	blobsURL = fmt.Sprintf(blobsURLBase, registry.URL, repository)

	return &Client{
		imageName:  imgName,
		imageTag:   imgTag,
		imagePath:  imgPath,
		imageRoot:  imgRoot,
		configPath: cfgPath,
		repository: repository,
		httpClient: httpClient,
	}
}

// Pull downloads and extracts the image.
func (i *Client) Pull() error {
	fmt.Printf("Pulling from %s using default tag: %s\n", i.repository, i.imageTag)

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

	if err := i.downloadImage(); err != nil {
		return err
	}

	if err := i.extractImage(); err != nil {
		return err
	}

	if err := i.fetchConfig(); err != nil {
		return err
	}

	fmt.Printf("Status: Downloaded image for %s:%s\n", i.imageName, i.imageTag)

	return nil
}

// authenticate retrieves an access token from Docker Hub.
func (i *Client) authenticate() error {
	var authResp registry.AuthResponse
	i.httpClient.SendRequestAndDecode(&authResp, http.MethodGet, authURL, nil)

	i.token = authResp.Token

	return nil
}

// fetchManifest retrieves the manifest or manifest index for the image.
func (i *Client) fetchManifest() error {
	headers := map[string]string{
		"Authorization": "Bearer " + i.token,
		"Accept":        "application/vnd.docker.distribution.manifest.v2+json",
	}

	resp, err := i.httpClient.SendRequest(http.MethodGet, manifestURL+i.imageTag, headers)
	if err != nil {
		return fmt.Errorf("failed to download layer: %v", err)
	}
	defer resp.Body.Close()

	i.manifest = &registry.Manifest{}
	ctype := resp.Header.Get("Content-Type")

	// Handle OCI Index (manifest list)
	if ctype == "application/vnd.oci.image.index.v1+json" || ctype == "application/vnd.docker.distribution.manifest.list.v2+json" {
		var index registry.ManifestIndex
		if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
			return fmt.Errorf("error decoding manifest index: %v", err)
		}

		fmt.Printf("Received index, contains %d platform manifests\n", len(index.Manifests))

		for _, m := range index.Manifests {
			if m.Platform.OS == runtime.GOOS && m.Platform.Architecture == runtime.GOARCH {
				fmt.Printf("Digest for %s/%s: %s\n", runtime.GOOS, runtime.GOARCH, m.Digest)

				return i.fetchManifestByDigest(m.Digest)
			}
		}

		return fmt.Errorf("no matching platform found in manifest index")
	}

	if err := json.NewDecoder(resp.Body).Decode(i.manifest); err != nil {
		return err
	}

	fmt.Printf("Found %d layers to download\n", len(i.manifest.Layers))

	return nil
}

// fetchManifestByDigest fetches a platform-specific manifest by its digest.
func (i *Client) fetchManifestByDigest(digest string) error {
	headers := map[string]string{
		"Authorization": "Bearer " + i.token,
		"Accept":        "application/vnd.docker.distribution.manifest.v2+json",
	}

	i.httpClient.SendRequestAndDecode(i.manifest, http.MethodGet, manifestURL+digest, headers)

	return nil
}

func (i *Client) downloadImage() error {
	i.downloadedLayers = make(map[string][]byte)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	for j, layer := range i.manifest.Layers {
		wg.Add(1)

		go func(index int, digest string) {
			defer wg.Done()

			if err := i.downloadLayer(ctx, index, digest); err != nil {
				select {
				case errChan <- err:
					cancel() // Cancel context to signal other goroutines to stop
				default:
				}
			}
		}(j, layer.Digest)
	}

	// Wait for either completion or first error
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Return first error if any
	if err := <-errChan; err != nil {
		return err
	}

	return nil
}

func (i *Client) downloadLayer(ctx context.Context, index int, digest string) error {
	fmt.Printf("Downloading layer %d/%d...\n", index+1, len(i.manifest.Layers))

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	headers := map[string]string{
		"Authorization": "Bearer " + i.token,
	}

	resp, err := i.httpClient.SendRequest(http.MethodGet, blobsURL+digest, headers)
	if err != nil {
		return fmt.Errorf("failed to download layer: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read layer %s: %v", digest, err)
	}

	// Verify digest
	hasher := sha256.Sum256(data)
	actual := "sha256:" + hex.EncodeToString(hasher[:])
	if actual != digest {
		return fmt.Errorf("digest mismatch for layer %d: expected %s, got %s", index+1, digest, actual)
	}

	// Thread-safe write to map
	i.mutex.Lock()
	i.downloadedLayers[digest] = data
	i.mutex.Unlock()

	return nil
}

func (i *Client) extractImage() error {
	for j, layer := range i.manifest.Layers {
		data, ok := i.downloadedLayers[layer.Digest]
		if !ok {
			return fmt.Errorf("layer data for %s not found", layer.Digest)
		}

		gr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create gzip reader for layer %d: %v", j+1, err)
		}
		defer gr.Close()

		tr := tar.NewReader(gr)

		if err := i.extractLayer(tr, i.imageRoot); err != nil {
			return fmt.Errorf("failed to extract layer %d: %v", j+1, err)
		}
	}

	return nil
}

// extractLayer unpacks the contents of a tar stream into the image root filesystem.
func (i *Client) extractLayer(tr *tar.Reader, imgRoot string) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		targetPath := filepath.Join(imgRoot, header.Name)

		// Prevent path traversal
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
			defer file.Close()

			if _, err := io.Copy(file, tr); err != nil {
				return fmt.Errorf("failed to write file %s: %v", targetPath, err)
			}

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

// fetchConfig downloads and saves the image configuration file.
func (i *Client) fetchConfig() error {
	digest := i.manifest.Config.Digest

	fmt.Printf("Downloading config file...\n")

	headers := make(map[string]string, 1)
	headers["Authorization"] = "Bearer " + i.token

	i.config = &registry.ImageConfig{}
	i.httpClient.SendRequestAndDecode(i.config, http.MethodGet, blobsURL+digest, headers)

	cfgData, err := json.MarshalIndent(i.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Save config data
	if err := os.WriteFile(i.configPath, cfgData, 0644); err != nil {
		return fmt.Errorf("failed to save config file: %v", err)
	}

	return nil
}

// mkdirRootfs removes existing image data and creates the rootfs directory structure.
func (i *Client) mkdirRootfs() error {
	if err := os.RemoveAll(i.imagePath); err != nil {
		return fmt.Errorf("failed to remove existing image dir: %v", err)
	}

	if err := os.MkdirAll(i.imageRoot, 0755); err != nil {
		return fmt.Errorf("failed to create image rootfs dir: %v", err)
	}

	return nil
}
