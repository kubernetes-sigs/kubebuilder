/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helpers

import (
	"context"
	"fmt"
	"io"
	log "log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/afero"
)

const KubebuilderReleaseURL = "https://github.com/kubernetes-sigs/kubebuilder/releases/download/%s/kubebuilder_%s_%s"

func BuildReleaseURL(version string) string {
	return fmt.Sprintf(KubebuilderReleaseURL, version, runtime.GOOS, runtime.GOARCH)
}

// DownloadReleaseVersionWith downloads the specified released version from GitHub releases and saves it
// to a temporary directory with executable permissions.
// Returns the temporary directory path containing the binary.
func DownloadReleaseVersionWith(version string) (string, error) {
	url := BuildReleaseURL(version)

	// Create temp directory
	fs := afero.NewOsFs()
	tempDir, err := afero.TempDir(fs, "", "kubebuilder"+version+"-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Ensure cleanup on any error after this point
	cleanupOnErr := func() {
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			log.Error("failed to remove temporary directory", "dir", tempDir, "error", rmErr)
		}
	}

	binaryPath := filepath.Join(tempDir, "kubebuilder")
	f, err := fs.Create(binaryPath)
	if err != nil {
		cleanupOnErr()
		return "", fmt.Errorf("failed to create the binary file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			log.Error("failed to close the binary file", "error", closeErr)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cleanupOnErr()
		return "", fmt.Errorf("failed to build download request: %w", err)
	}
	req.Header.Set("User-Agent", "kubebuilder-updater/1.0 (+https://github.com/kubernetes-sigs/kubebuilder)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cleanupOnErr()
		return "", fmt.Errorf("failed to download the binary: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error("failed to close HTTP response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		cleanupOnErr()
		return "", fmt.Errorf("failed to download the binary: HTTP %d", resp.StatusCode)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		cleanupOnErr()
		return "", fmt.Errorf("failed to write the binary content to file: %w", err)
	}

	// Flush to disk before changing mode (best effort)
	if syncErr := f.Sync(); syncErr != nil {
		log.Warn("failed to sync binary to disk (continuing)", "error", syncErr)
	}

	if err := os.Chmod(binaryPath, 0o755); err != nil {
		cleanupOnErr()
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	return tempDir, nil
}
