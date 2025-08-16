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
	"fmt"
	"io"
	log "log/slog"
	"net/http"
	"os"
	"runtime"

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

	fs := afero.NewOsFs()
	tempDir, err := afero.TempDir(fs, "", "kubebuilder"+version+"-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	binaryPath := tempDir + "/kubebuilder"
	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create the binary file: %w", err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Error("failed to close the file", "error", err)
		}
	}()

	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download the binary: %w", err)
	}
	defer func() {
		if err = response.Body.Close(); err != nil {
			log.Error("failed to close the connection", "error", err)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download the binary: HTTP %d", response.StatusCode)
	}

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write the binary content to file: %w", err)
	}

	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}
	return tempDir, nil
}
