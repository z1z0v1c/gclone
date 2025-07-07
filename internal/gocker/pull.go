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

	if err := img.pull(); err != nil {
		fmt.Printf("Error while pulling %q image: %v\n", img.Name, err)
		os.Exit(1)
	}
}

func fatalf(format string, a ...any) {
	fmt.Printf(format, a...)
	os.Exit(1)
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
