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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

const (
	fromVersion           = "v4.5.2"
	toVersion             = "v4.6.0"
	toVersionWithConflict = "v4.7.0"

	// Binary patterns for cleanup
	binFromVersionPath           = "/tmp/kubebuilder" + fromVersion + "-*"
	pathBinToVersion             = "/tmp/kubebuilder" + toVersion + "-*"
	pathBinToVersionWithConflict = "/tmp/kubebuilder" + toVersionWithConflict + "-*"

	controllerImplementation = `// Fetch the TestOperator instance
	testOperator := &webappv1.TestOperator{}
	err := r.Get(ctx, req.NamespacedName, testOperator)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("testOperator resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get testOperator")
		return ctrl.Result{}, err
	}

	log.Info("testOperator reconciled")`
)

var _ = Describe("kubebuilder", func() {
	Context("alpha update", func() {
		var (
			mockProjectDir     string
			pathBinFromVersion string
			kbc                *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			By("setting up test context with binary build from source")
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			By("creating isolated mock project directory in /tmp to avoid git conflicts")
			mockProjectDir, err = os.MkdirTemp("/tmp", "kubebuilder-mock-project-")
			Expect(err).NotTo(HaveOccurred())

			By("downloading kubebuilder v4.5.2 binary to isolated /tmp directory")
			pathBinFromVersion, err = downloadKubebuilder()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("cleaning up test artifacts")
			_ = os.RemoveAll(mockProjectDir)
			_ = os.RemoveAll(filepath.Dir(pathBinFromVersion))

			// Clean up kubebuilder alpha update downloaded binaries
			binaryPatterns := []string{
				pathBinFromVersion,
				pathBinToVersion,
				pathBinToVersionWithConflict,
			}

			for _, pattern := range binaryPatterns {
				matches, _ := filepath.Glob(pattern)
				for _, path := range matches {
					_ = os.RemoveAll(path)
				}
			}

			_ = os.RemoveAll(kbc.Dir)
		})

		It("should update project from v4.5.2 to v4.6.0 without conflicts", func() {
			By("creating mock project with kubebuilder v4.5.2")
			createMockProject(mockProjectDir, pathBinFromVersion)

			By("adding custom code in API and controller")
			updateAPI(mockProjectDir)
			updateController(mockProjectDir)

			By("initializing git repository and committing mock project")
			initializeGitRepo(mockProjectDir)

			By("running alpha update from v4.5.2 to v4.6.0")
			runAlphaUpdate(mockProjectDir, kbc)

			By("validating custom code preservation")
			validateCustomCodePreservation(mockProjectDir)

			By("validating no conflict markers are present")
			validateConflictMarkers(mockProjectDir, false)
		})

		It("should update project from v4.5.2 to v4.7.0 with --force flag and create conflict markers", func() {
			By("creating mock project with kubebuilder v4.5.2")
			createMockProject(mockProjectDir, pathBinFromVersion)

			By("adding custom code in API and controller")
			updateAPI(mockProjectDir)
			updateController(mockProjectDir)

			By("initializing git repository and committing mock project")
			initializeGitRepo(mockProjectDir)

			By("running alpha update from v4.5.2 to v4.7.0 with --force flag")
			runAlphaUpdateWithForce(mockProjectDir, kbc)

			By("validating conflict markers are present")
			validateConflictMarkers(mockProjectDir, true)
		})

		It("should stop when updating the project from v4.5.2 to v4.7.0 without the flag force "+
			"to allow manual conflicts resolution", func() {
			By("creating mock project with kubebuilder v4.5.2")
			createMockProject(mockProjectDir, pathBinFromVersion)

			By("adding custom code in API and controller")
			updateAPI(mockProjectDir)
			updateController(mockProjectDir)

			By("initializing git repository and committing mock project")
			initializeGitRepo(mockProjectDir)

			By("running alpha update from v4.5.2 to v4.7.0 without --force flag")
			runAlphaUpdateWithoutForce(mockProjectDir, kbc)

			By("validating merge stopped in conflict state for manual resolution")
			validateConflictState(mockProjectDir)
		})
	})
})

// downloadKubebuilder downloads the --from-version kubebuilder binary to a temporary directory
func downloadKubebuilder() (string, error) {
	return downloadKubebuilderVersion(fromVersion)
}

// downloadKubebuilderVersion downloads a specific kubebuilder version binary to a temporary directory
func downloadKubebuilderVersion(version string) (string, error) {
	binaryDir, err := os.MkdirTemp("", "kubebuilder-"+version+"-")
	if err != nil {
		return "", fmt.Errorf("failed to create binary directory: %w", err)
	}

	url := fmt.Sprintf(
		"https://github.com/kubernetes-sigs/kubebuilder/releases/download/%s/kubebuilder_%s_%s",
		version,
		runtime.GOOS,
		runtime.GOARCH,
	)
	binaryPath := filepath.Join(binaryDir, "kubebuilder")

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download kubebuilder %s: %w", version, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download kubebuilder %s: HTTP %d", version, resp.StatusCode)
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
	By("running kubebuilder init")
	cmd := exec.Command(binaryPath, "init", "--domain", "example.com", "--repo", "github.com/example/test-operator")
	cmd.Dir = projectDir
	_, err := cmd.CombinedOutput()
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

func updateController(projectDir string) {
	controllerFile := filepath.Join(projectDir, "internal", "controller", "testoperator_controller.go")

	err := pluginutil.ReplaceInFile(
		controllerFile,
		"_ = logf.FromContext(ctx)",
		"log := logf.FromContext(ctx)",
	)
	Expect(err).NotTo(HaveOccurred())

	err = pluginutil.ReplaceInFile(
		controllerFile,
		"// TODO(user): your logic here",
		controllerImplementation,
	)
	Expect(err).NotTo(HaveOccurred())
}

func updateAPI(projectDir string) {
	typesFile := filepath.Join(projectDir, "api", "v1", "testoperator_types.go")
	err := pluginutil.ReplaceInFile(
		typesFile,
		"Foo string `json:\"foo,omitempty\"`",
		`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:default=1
	Size int32 `+"`json:\"size,omitempty\"`",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to update testoperator_types.go")
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
	cmd := exec.Command(kbc.BinaryName, "alpha", "update",
		"--from-version", fromVersion, "--to-version", toVersion, "--from-branch", "main")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Alpha update failed: %s", string(output)))
}

func runAlphaUpdateWithForce(projectDir string, kbc *utils.TestContext) {
	cmd := exec.Command(kbc.BinaryName, "alpha", "update", "--from-version", fromVersion,
		"--to-version", toVersionWithConflict, "--from-branch", "main", "--force")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Alpha update with force failed: %s", string(output)))
}

func runAlphaUpdateWithoutForce(projectDir string, kbc *utils.TestContext) {
	cmd := exec.Command(kbc.BinaryName, "alpha", "update",
		"--from-version", fromVersion, "--to-version", toVersionWithConflict, "--from-branch", "main")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	Expect(err).To(HaveOccurred())
	Expect(string(output)).To(ContainSubstring("merge stopped due to conflicts"))
}

func validateCustomCodePreservation(projectDir string) {
	By("validating the API")
	typesFile := filepath.Join(projectDir, "api", "v1", "testoperator_types.go")
	content, err := os.ReadFile(typesFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(ContainSubstring("Size int32 `json:\"size,omitempty\"`"))
	Expect(string(content)).To(ContainSubstring("// +kubebuilder:validation:Minimum=0"))
	Expect(string(content)).To(ContainSubstring("// +kubebuilder:validation:Maximum=3"))
	Expect(string(content)).To(ContainSubstring("// +kubebuilder:default=1"))

	By("validating the Controller")
	controllerFile := filepath.Join(projectDir, "internal", "controller", "testoperator_controller.go")
	content, err = os.ReadFile(controllerFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(ContainSubstring(controllerImplementation))
}

func validateConflictMarkers(projectDir string, expectMarkers bool) {
	if expectMarkers {
		By("validating conflict markers are present")
	} else {
		By("validating no conflict markers are present")
	}

	filesToCheck := []string{
		filepath.Join(projectDir, "api", "v1", "testoperator_types.go"),
		filepath.Join(projectDir, "internal", "controller", "testoperator_controller.go"),
	}

	conflictMarkersFound := false
	for _, file := range filesToCheck {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		fileContent := string(content)
		if strings.Contains(fileContent, "<<<<<<<") && strings.Contains(fileContent, "=======") &&
			strings.Contains(fileContent, ">>>>>>>") {
			conflictMarkersFound = true
			break
		}
	}

	if expectMarkers {
		Expect(conflictMarkersFound).To(BeTrue(), "Expected to find conflict markers in at least one file")
	} else {
		Expect(conflictMarkersFound).To(BeFalse(), "Expected no conflict markers, but found them in files")
	}
}

func validateConflictState(projectDir string) {
	By("validating merge stopped with conflicts requiring manual resolution")
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	statusOutput := strings.TrimSpace(string(output))
	Expect(statusOutput).NotTo(BeEmpty(), "Working directory should have uncommitted changes from merge conflict")

	validateConflictMarkers(projectDir, true)
}
