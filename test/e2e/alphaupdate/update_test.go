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

package alphaupdate

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

const (
	fromVersion = "v4.5.2"
	toVersion   = "v4.6.0"

	// Binary patterns for cleanup
	binFromVersionPattern = "/tmp/kubebuilder" + fromVersion + "-*"
	binToVersionPattern   = "/tmp/kubebuilder" + toVersion + "-*"
)

var _ = Describe("kubebuilder", func() {
	Context("alpha update", func() {
		var (
			mockProjectDir     string
			binFromVersionPath string
			kbc                *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			By("setting up test context with current kubebuilder binary")
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			By("creating isolated mock project directory in /tmp to avoid git conflicts")
			mockProjectDir, err = os.MkdirTemp("/tmp", "kubebuilder-mock-project-")
			Expect(err).NotTo(HaveOccurred())

			By("downloading kubebuilder v4.5.2 binary to isolated /tmp directory")
			binFromVersionPath, err = downloadKubebuilder()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("cleaning up test artifacts")

			_ = os.RemoveAll(mockProjectDir)
			_ = os.RemoveAll(filepath.Dir(binFromVersionPath))

			// Clean up kubebuilder alpha update downloaded binaries
			binaryPatterns := []string{
				binFromVersionPattern,
				binToVersionPattern,
			}

			for _, pattern := range binaryPatterns {
				matches, _ := filepath.Glob(pattern)
				for _, path := range matches {
					_ = os.RemoveAll(path)
				}
			}

			// Clean up TestContext
			if kbc != nil {
				kbc.Destroy()
			}
		})

		It("should update project from v4.5.2 to v4.6.0 preserving custom code", func() {
			By("creating mock project with kubebuilder v4.5.2")
			createMockProject(mockProjectDir, binFromVersionPath)

			By("injecting custom code in API and controller")
			injectCustomCode(mockProjectDir)

			By("initializing git repository and committing mock project")
			initializeGitRepo(mockProjectDir)

			By("running alpha update from v4.5.2 to v4.6.0")
			runAlphaUpdate(mockProjectDir, kbc)

			By("validating custom code preservation")
			validateCustomCodePreservation(mockProjectDir)
		})
	})
})

// downloadKubebuilder downloads the --from-version kubebuilder binary to a temporary directory
func downloadKubebuilder() (string, error) {
	binaryDir, err := os.MkdirTemp("", "kubebuilder-v4.5.2-")
	if err != nil {
		return "", fmt.Errorf("failed to create binary directory: %w", err)
	}

	url := fmt.Sprintf(
		"https://github.com/kubernetes-sigs/kubebuilder/releases/download/%s/kubebuilder_%s_%s",
		fromVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
	binaryPath := filepath.Join(binaryDir, "kubebuilder")

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download kubebuilder %s: %w", fromVersion, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download kubebuilder %s: HTTP %d", fromVersion, resp.StatusCode)
	}

	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create binary file: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write binary: %w", err)
	}

	err = os.Chmod(binaryPath, 0o755)
	if err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	return binaryPath, nil
}

func createMockProject(projectDir, binaryPath string) {
	err := os.Chdir(projectDir)
	Expect(err).NotTo(HaveOccurred())

	By("running kubebuilder init")
	cmd := exec.Command(binaryPath, "init", "--domain", "example.com", "--repo", "github.com/example/test-operator")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	By("running kubebuilder create api")
	cmd = exec.Command(
		binaryPath, "create", "api",
		"--group", "webapp",
		"--version", "v1",
		"--kind", "TestOperator",
		"--resource", "--controller",
	)
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	By("running make generate manifests")
	cmd = exec.Command("make", "generate", "manifests")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
}

func injectCustomCode(projectDir string) {
	typesFile := filepath.Join(projectDir, "api", "v1", "testoperator_types.go")
	err := pluginutil.InsertCode(
		typesFile,
		"Foo string `json:\"foo,omitempty\"`",
		`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:default=1
	// Size is the size of the memcached deployment
	Size int32 `+"`json:\"size,omitempty\"`",
	)
	Expect(err).NotTo(HaveOccurred())
	controllerFile := filepath.Join(projectDir, "internal", "controller", "testoperator_controller.go")
	err = pluginutil.InsertCode(
		controllerFile,
		"// TODO(user): your logic here",
		`// Custom reconciliation logic
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling TestOperator")

	// Fetch the TestOperator instance
	testOperator := &webappv1.TestOperator{}
	err := r.Get(ctx, req.NamespacedName, testOperator)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Custom logic: log the size field
	log.Info("TestOperator size", "size", testOperator.Spec.Size)`,
	)
	Expect(err).NotTo(HaveOccurred())
}

func initializeGitRepo(projectDir string) {
	By("initializing git repository")
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	By("adding all files to git")
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	By("committing initial project state")
	cmd = exec.Command("git", "commit", "-m", "Initial project with custom code")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	By("ensuring main branch exists and is current")
	cmd = exec.Command("git", "checkout", "-b", "main")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	if err != nil {
		// If main branch already exists, just switch to it
		cmd = exec.Command("git", "checkout", "main")
		cmd.Dir = projectDir
		_, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
	}
}

func runAlphaUpdate(projectDir string, kbc *utils.TestContext) {
	err := os.Chdir(projectDir)
	Expect(err).NotTo(HaveOccurred())

	// Use TestContext to run alpha update command
	cmd := exec.Command(kbc.BinaryName, "alpha", "update",
		"--from-version", fromVersion, "--to-version", toVersion, "--from-branch", "main")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Alpha update failed: %s", string(output)))
}

func validateCustomCodePreservation(projectDir string) {
	typesFile := filepath.Join(projectDir, "api", "v1", "testoperator_types.go")
	content, err := os.ReadFile(typesFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(ContainSubstring("Size int32 `json:\"size,omitempty\"`"))
	Expect(string(content)).To(ContainSubstring("Size is the size of the memcached deployment"))

	controllerFile := filepath.Join(projectDir, "internal", "controller", "testoperator_controller.go")
	content, err = os.ReadFile(controllerFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(ContainSubstring("Custom reconciliation logic"))
	Expect(string(content)).To(ContainSubstring("log.Info(\"Reconciling TestOperator\")"))
	Expect(string(content)).To(ContainSubstring("log.Info(\"TestOperator size\", \"size\", testOperator.Spec.Size)"))
}
