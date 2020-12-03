/*
Copyright 2020 The Kubernetes authors.

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

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"k8s.io/klog"
)

const (
	// defaultKubebuilderPath is the fixed value defined in the controller-runtime to looking for the
	// binaries.
	defaultKubebuilderPath = "/usr/local/kubebuilder/bin"

	// kubernetesVersion defines the version used to download the binaries
	kubernetesVersion = "1.16.4"
)

func main() {
	klog.Info("Checking EnvTest binaries")
	assetsEnv := os.Getenv("KUBEBUILDER_ASSETS")
	if strings.TrimSpace(assetsEnv) != "" {
		klog.Infof("EnvTest binaries configured via KUBEBUILDER_ASSETS: %s", assetsEnv)
		os.Exit(0)
	}

	binPath := filepath.Join("..", "bin")
	if hasBinaries(binPath) {
		klog.Info("EnvTest binaries found in bin/")
		os.Exit(0)
	}

	klog.Infof("Downloading EnvTest tools")
	envtest_tools_archive_name := getEnvToolsArchiveName()
	envtest_tools_download_url := "https://storage.googleapis.com/kubebuilder-tools/" + envtest_tools_archive_name

	err := downloadFile(envtest_tools_archive_name, envtest_tools_download_url)
	if err != nil {
		klog.Fatalf("unable to download the file (%v) from (%v): %v", envtest_tools_archive_name, envtest_tools_download_url, err)

	}

	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		klog.Infof("Creating bin/ directory")
		err = os.Mkdir(binPath, 0755)
		if err != nil {
			klog.Fatalf("error to create the bin/ directory : %s", err)
		}
	}

	cmd := exec.Command("tar", "-C", binPath, "--strip-components=2", "-zvxf", envtest_tools_archive_name)
	if err := cmd.Run(); err != nil {
		klog.Fatalf("error to untar bin %v: %v", filepath.Join(binPath, envtest_tools_archive_name), err)
	}

	cmd = exec.Command("rm", "-rf", envtest_tools_archive_name)
	if err := cmd.Run(); err != nil {
		klog.Fatal(err)
	}
	klog.Infof("EnvTest binaries are in the directory %v", binPath)
}

// getEnvToolsArchiveName will return the name of the env tools archive based or the arch and so
func getEnvToolsArchiveName() string {
	cmd := exec.Command("go", "env", "GOARCH")
	goarch, err := cmd.CombinedOutput()
	if err != nil {
		klog.Fatal(err)
	}

	cmd = exec.Command("go", "env", "GOOS")
	goos, err := cmd.CombinedOutput()
	if err != nil {
		klog.Fatal(err)
	}
	return fmt.Sprintf("kubebuilder-tools-%s-%s-%s.tar.gz", kubernetesVersion, strings.TrimSpace(string(goos)), strings.TrimSpace(string(goarch)))
}

// hasBinaries will return true when the envtest binaries are found
func hasBinaries(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(filepath.Join(path, "etcd")); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(filepath.Join(path, "kube-apiserver")); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(filepath.Join(path, "kubectl")); os.IsNotExist(err) {
		return false
	}
	return true
}

// downloadFile will download a url to a local file.
func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
