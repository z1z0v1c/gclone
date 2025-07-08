package image

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
	"runtime"
	"strings"

	"github.com/z1z0v1c/gocker/pkg/http"
)

const (
	// RelativeImagesPath is the relative images path under the user's home directory
	RelativeImagesPath = ".local/share/gocker/images/"

	registry        = "registry-1.docker.io"
	authURLBase     = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
	manifestURLBase = "https://%s/v2/%s/manifests/"
	blobsURLBase    = "https://%s/v2/%s/blobs/"
)

var (
	authURL     string
	manifestURL string
	blobsURL    string
)

// ImagePuller encapsulates the parameters to pull and unpack an image.
type ImagePuller struct {
	imgName    string
	imgTag     string
	imgPath    string
	imgRoot    string
	cfgPath    string
	repository string
	token      string

	manifest *Manifest
	cfg      *ImageConfig

	httpClient *http.Client
}

// NewImagePuller creates and initializes a new ImagePuller for the given image name.
func NewImagePuller(imgName string, httpClient *http.Client) *ImagePuller {
	tag := "latest"
	homeDir := os.Getenv("HOME")

	imgPath := filepath.Join(homeDir, RelativeImagesPath, imgName)
	imgRoot := filepath.Join(imgPath, "rootfs")
	cfgPath := filepath.Join(imgPath, ".config.json")
	repository := filepath.Join("library", imgName)

	authURL = fmt.Sprintf(authURLBase, repository)
	manifestURL = fmt.Sprintf(manifestURLBase, registry, repository)
	blobsURL = fmt.Sprintf(blobsURLBase, registry, repository)

	return &ImagePuller{
		imgName:    imgName,
		imgTag:     tag,
		imgPath:    imgPath,
		imgRoot:    imgRoot,
		cfgPath:    cfgPath,
		repository: repository,
		httpClient: httpClient,
	}
}

// Pull downloads and extracts the image.
func (i *ImagePuller) Pull() error {
	fmt.Printf("Pulling from %s using default tag: %s\n", i.repository, i.imgTag)

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

	if err := i.downloadAndExtractImage(); err != nil {
		return err
	}

	if err := i.fetchConfig(); err != nil {
		return err
	}

	fmt.Printf("Status: Downloaded image for %s:%s\n", i.imgName, i.imgTag)

	return nil
}

// authenticate retrieves an access token from Docker Hub.
func (i *ImagePuller) authenticate() error {
	var authResp AuthResponse
	i.httpClient.SendRequestAndDecode(&authResp, http.MethodGet, authURL, nil)

	i.token = authResp.Token

	return nil
}

// fetchManifest retrieves the manifest or manifest index for the image.
func (i *ImagePuller) fetchManifest() error {
	headers := map[string]string{
		"Authorization": "Bearer " + i.token,
		"Accept":        "application/vnd.docker.distribution.manifest.v2+json",
	}

	resp, err := i.httpClient.SendRequest(http.MethodGet, manifestURL+i.imgTag, headers)
	if err != nil {
		return fmt.Errorf("failed to download layer: %v", err)
	}
	defer resp.Body.Close()

	i.manifest = &Manifest{}
	ctype := resp.Header.Get("Content-Type")

	// Handle OCI Index (manifest list)
	if ctype == "application/vnd.oci.image.index.v1+json" || ctype == "application/vnd.docker.distribution.manifest.list.v2+json" {
		var index ManifestIndex
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
func (i *ImagePuller) fetchManifestByDigest(digest string) error {
	headers := map[string]string{
		"Authorization": "Bearer " + i.token,
		"Accept":        "application/vnd.docker.distribution.manifest.v2+json",
	}

	i.httpClient.SendRequestAndDecode(i.manifest, http.MethodGet, manifestURL+digest, headers)

	return nil
}

// downloadAndExtractImage downloads and extracts all layers listed in the manifest.
func (i *ImagePuller) downloadAndExtractImage() error {
	for j, layer := range i.manifest.Layers {
		fmt.Printf("Downloading layer %d/%d...\n", j+1, len(i.manifest.Layers))

		if err := i.downloadAndExtractLayer(layer.Digest); err != nil {
			return err
		}
	}

	return nil
}

// downloadAndExtractLayer downloads and extracts a single layer by digest.
func (i *ImagePuller) downloadAndExtractLayer(digest string) error {
	headers := map[string]string{
		"Authorization": "Bearer " + i.token,
	}

	resp, err := i.httpClient.SendRequest(http.MethodGet, blobsURL+digest, headers)
	if err != nil {
		return fmt.Errorf("failed to download layer: %v", err)
	}
	defer resp.Body.Close()

	// Verify digest
	hasher := sha256.New()
	reader := io.TeeReader(resp.Body, hasher)

	gr, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	i.extractLayer(tr, i.imgRoot)

	// Verify the digest
	actualDigest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if actualDigest != digest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", digest, actualDigest)
	}

	return nil
}

// extractLayer unpacks the contents of a tar stream into the image root filesystem.
func (i *ImagePuller) extractLayer(tr *tar.Reader, imgRoot string) error {
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
func (i *ImagePuller) fetchConfig() error {
	digest := i.manifest.Config.Digest

	fmt.Printf("Downloading config file...\n")

	headers := make(map[string]string, 1)
	headers["Authorization"] = "Bearer " + i.token

	i.cfg = &ImageConfig{}
	i.httpClient.SendRequestAndDecode(i.cfg, http.MethodGet, blobsURL+digest, headers)

	cfgData, err := json.MarshalIndent(i.cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Save config data
	if err := os.WriteFile(i.cfgPath, cfgData, 0644); err != nil {
		return fmt.Errorf("failed to save config file: %v", err)
	}

	return nil
}

// mkdirRootfs removes existing image data and creates the rootfs directory structure.
func (i *ImagePuller) mkdirRootfs() error {
	if err := os.RemoveAll(i.imgPath); err != nil {
		return fmt.Errorf("failed to remove existing image dir: %v", err)
	}

	if err := os.MkdirAll(i.imgRoot, 0755); err != nil {
		return fmt.Errorf("failed to create image rootfs dir: %v", err)
	}

	return nil
}
