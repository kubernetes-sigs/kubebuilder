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
	"bytes"
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

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

const (
	fromVersion           = "v4.5.2"
	toVersion             = "v4.6.0"
	toVersionWithConflict = "v4.7.0"

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

	customField = `// +kubebuilder:validation:Minimum=0
// +kubebuilder:validation:Maximum=3
// +kubebuilder:default=1
Size int32 ` + "`json:\"size,omitempty\"`" + `
`
)

var _ = Describe("kubebuilder", func() {
	Context("alpha update", func() {
		var (
			pathBinFromVersion string
			kbc                *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			By("setting up test context with binary build from source")
			kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			pathBinFromVersion, err = downloadKubebuilderVersion(fromVersion)
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command(pathBinFromVersion, "init", "--domain", "example.com", "--repo",
				"github.com/example/test-operator")
			cmd.Dir = kbc.Dir
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("init failed: %s", output))

			cmd = exec.Command(pathBinFromVersion, "create", "api", "--group", "webapp", "--version", "v1",
				"--kind", "TestOperator", "--resource", "--controller")
			cmd.Dir = kbc.Dir
			output, err = cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("create api failed: %s", output))
			Expect(kbc.Make("generate", "manifests")).To(Succeed())

			updateAPI(kbc.Dir)
			updateController(kbc.Dir)
			initializeGitRepo(kbc.Dir)
		})

		AfterEach(func() {
			By("cleaning up test artifacts")
			_ = os.RemoveAll(filepath.Dir(pathBinFromVersion))
			_ = os.RemoveAll(kbc.Dir)
			kbc.Destroy()
		})

		It("should update project from v4.5.2 to v4.6.0 without conflicts", func() {
			By("running alpha update from v4.5.2 to v4.6.0")
			cmd := exec.Command(
				kbc.BinaryName, "alpha", "update",
				"--from-version", fromVersion,
				"--to-version", toVersion,
				"--from-branch", "main",
			)
			cmd.Dir = kbc.Dir
			out, err := kbc.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), string(out))

			By("checking that custom code is preserved")
			validateCustomCodePreservation(kbc.Dir)

			By("checking that no conflict markers are present in the project files")
			Expect(hasConflictMarkers(kbc.Dir)).To(BeFalse())

			By("checking that go module is upgraded")
			validateCommonGoModule(kbc.Dir)

			By("checking that Makefile is updated")
			validateMakefileContent(kbc.Dir)

			By("checking temporary branches were cleaned up locally")
			outRefs, err := exec.Command("git", "-C", kbc.Dir, "for-each-ref",
				"--format=%(refname:short)", "refs/heads").CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(outRefs))
			Expect(string(outRefs)).NotTo(ContainSubstring("tmp-ancestor"))
			Expect(string(outRefs)).NotTo(ContainSubstring("tmp-original"))
			Expect(string(outRefs)).NotTo(ContainSubstring("tmp-upgrade"))
			Expect(string(outRefs)).NotTo(ContainSubstring("tmp-merge"))
		})

		It("should update project from v4.5.2 to v4.7.0 with --force flag and create conflict markers", func() {
			By("modifying original Makefile to use CONTROLLER_TOOLS_VERSION v0.17.3")
			modifyMakefileControllerTools(kbc.Dir, "v0.17.3")

			By("running alpha update with --force (default behavior is squash)")
			cmd := exec.Command(
				kbc.BinaryName, "alpha", "update",
				"--from-version", fromVersion,
				"--to-version", toVersionWithConflict,
				"--from-branch", "main",
				"--force",
			)
			cmd.Dir = kbc.Dir
			out, err := kbc.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), string(out))

			By("checking that custom code is preserved")
			validateCustomCodePreservation(kbc.Dir)

			By("checking that conflict markers are present in the project files")
			Expect(hasConflictMarkers(kbc.Dir)).To(BeTrue())

			By("checking that go module is upgraded to expected versions")
			validateCommonGoModule(kbc.Dir)

			By("checking that Makefile is updated and has conflict between old and new versions in Makefile")
			makefilePath := filepath.Join(kbc.Dir, "Makefile")
			content, err := os.ReadFile(makefilePath)
			Expect(err).NotTo(HaveOccurred(), "Failed to read Makefile after update")
			makefileStr := string(content)

			// Should update to the new version
			Expect(makefileStr).To(ContainSubstring(`GOLANGCI_LINT_VERSION ?= v2.1.6`))

			// The original project was scaffolded with v0.17.2 (from v4.5.2).
			// The user manually updated it to v0.17.3.
			// The target upgrade version (v4.7.0) introduces v0.18.0.
			//
			// Because both the user's version (v0.17.3) and the scaffold version (v0.18.0) differ,
			// we expect Git to insert conflict markers around this line in the Makefile:
			//
			// <<<<<<< HEAD
			// CONTROLLER_TOOLS_VERSION ?= v0.18.0
			// =======
			// CONTROLLER_TOOLS_VERSION ?= v0.17.3
			// >>>>>>> tmp-original-*
			Expect(makefileStr).To(ContainSubstring("<<<<<<<"),
				"Expected conflict marker <<<<<<< in Makefile")
			Expect(makefileStr).To(ContainSubstring("======="),
				"Expected conflict separator ======= in Makefile")
			Expect(makefileStr).To(ContainSubstring(">>>>>>>"),
				"Expected conflict marker >>>>>>> in Makefile")
			Expect(makefileStr).To(ContainSubstring("CONTROLLER_TOOLS_VERSION ?= v0.17.3"),
				"Expected original user version in conflict")
			Expect(makefileStr).To(ContainSubstring("CONTROLLER_TOOLS_VERSION ?= v0.18.0"),
				"Expected latest scaffold version in conflict")

			By("checking that the output branch (squashed) exists and is 1 commit ahead of main")
			prBranch := "kubebuilder-update-from-" + fromVersion + "-to-" + toVersionWithConflict

			git := func(args ...string) ([]byte, error) {
				cmd := exec.Command("git", args...)
				cmd.Dir = kbc.Dir
				return cmd.CombinedOutput()
			}

			By("checking that the squashed branch exists")
			_, err = git("rev-parse", "--verify", prBranch)
			Expect(err).NotTo(HaveOccurred())

			By("checking that exactly one squashed commit ahead of main")
			count, err := git("rev-list", "--count", prBranch, "^main")
			Expect(err).NotTo(HaveOccurred(), string(count))
			Expect(strings.TrimSpace(string(count))).To(Equal("1"))

			By("checking commit message of the squashed branch")
			msg, err := git("log", "-1", "--pretty=%B", prBranch)
			Expect(err).NotTo(HaveOccurred(), string(msg))
			expected := fmt.Sprintf(
				":warning: (chore) [with conflicts] scaffold update: %s -> %s", fromVersion, toVersionWithConflict)
			Expect(string(msg)).To(ContainSubstring(expected))
		})

		It("should stop when updating the project from v4.5.2 to v4.7.0 without the flag force", func() {
			By("running alpha update without --force flag")
			cmd := exec.Command(
				kbc.BinaryName, "alpha", "update",
				"--from-version", fromVersion,
				"--to-version", toVersionWithConflict,
				"--from-branch", "main",
			)
			cmd.Dir = kbc.Dir
			out, err := kbc.Run(cmd)
			Expect(err).To(HaveOccurred())
			Expect(string(out)).To(ContainSubstring("merge stopped due to conflicts"))

			By("validating that merge stopped with conflicts requiring manual resolution")
			validateConflictState(kbc.Dir)

			By("checking that custom code is preserved")
			validateCustomCodePreservation(kbc.Dir)

			By("checking that go module is upgraded")
			validateCommonGoModule(kbc.Dir)
		})

		It("should preserve specified paths from base when squashing (e.g., .github/workflows)", func() {
			By("adding a workflow on main branch that should be preserved")
			wfDir := filepath.Join(kbc.Dir, ".github", "workflows")
			Expect(os.MkdirAll(wfDir, 0o755)).To(Succeed())
			wf := filepath.Join(wfDir, "ci.yml")
			Expect(os.WriteFile(wf, []byte("name: KEEP_ME\n"), 0o644)).To(Succeed())

			git := func(args ...string) {
				c := exec.Command("git", args...)
				c.Dir = kbc.Dir
				o, e := c.CombinedOutput()
				Expect(e).NotTo(HaveOccurred(), string(o))
			}
			git("add", ".github/workflows/ci.yml")
			git("commit", "-m", "add ci workflow")

			By("running update (default squash) with --restore-path")
			cmd := exec.Command(
				kbc.BinaryName, "alpha", "update",
				"--from-version", fromVersion,
				"--to-version", toVersion,
				"--from-branch", "main",
				"--restore-path", ".github/workflows",
			)
			cmd.Dir = kbc.Dir
			out, err := kbc.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), string(out))

			By("workflow content is preserved on output branch")
			data, err := os.ReadFile(wf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("KEEP_ME"))
		})

		It("should succeed with no action when from-version and to-version are the same", func() {
			cmd := exec.Command(kbc.BinaryName, "alpha", "update",
				"--from-version", fromVersion,
				"--to-version", fromVersion,
				"--from-branch", "main")
			output, err := kbc.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("already uses the specified version"))
			Expect(string(output)).To(ContainSubstring("No action taken"))
		})
	})
})

func modifyMakefileControllerTools(projectDir, newVersion string) {
	makefilePath := filepath.Join(projectDir, "Makefile")
	oldLine := "CONTROLLER_TOOLS_VERSION ?= v0.17.2"
	newLine := fmt.Sprintf("CONTROLLER_TOOLS_VERSION ?= %s", newVersion)

	By("replacing the controller-tools version in the Makefile")
	Expect(util.ReplaceInFile(makefilePath, oldLine, newLine)).
		To(Succeed(), "Failed to update CONTROLLER_TOOLS_VERSION in Makefile")

	By("committing the Makefile change to simulate user customization")
	cmds := [][]string{
		{"git", "add", "Makefile"},
		{"git", "commit", "-m", fmt.Sprintf("User modified CONTROLLER_TOOLS_VERSION to %s", newVersion)},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Git command failed: %s", output))
	}
}

func validateMakefileContent(projectDir string) {
	makefilePath := filepath.Join(projectDir, "Makefile")
	content, err := os.ReadFile(makefilePath)
	Expect(err).NotTo(HaveOccurred(), "Failed to read Makefile")

	makefile := string(content)

	Expect(makefile).To(ContainSubstring(`CONTROLLER_TOOLS_VERSION ?= v0.18.0`))
	Expect(makefile).To(ContainSubstring(`GOLANGCI_LINT_VERSION ?= v2.1.0`))

	Expect(makefile).To(ContainSubstring(`.PHONY: test-e2e`))
	Expect(makefile).To(ContainSubstring(`go test ./test/e2e/ -v -ginkgo.v`))

	Expect(makefile).To(ContainSubstring(`.PHONY: cleanup-test-e2e`))
	Expect(makefile).To(ContainSubstring(`delete cluster --name $(KIND_CLUSTER)`))
}

// 4.6.0 and 4.7.0 updates include common changes that should be validated
func validateCommonGoModule(projectDir string) {
	expectModuleVersion(projectDir, "github.com/onsi/ginkgo/v2", "v2.22.0")
	expectModuleVersion(projectDir, "github.com/onsi/gomega", "v1.36.1")
	expectModuleVersion(projectDir, "k8s.io/apimachinery", "v0.33.0")
	expectModuleVersion(projectDir, "k8s.io/client-go", "v0.33.0")
	expectModuleVersion(projectDir, "sigs.k8s.io/controller-runtime", "v0.21.0")
}

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

func updateController(projectDir string) {
	controllerFile := filepath.Join(projectDir, "internal", "controller", "testoperator_controller.go")
	Expect(util.ReplaceInFile(controllerFile, "_ = logf.FromContext(ctx)", "log := logf.FromContext(ctx)")).To(Succeed())
	Expect(util.ReplaceInFile(controllerFile, "// TODO(user): your logic here", controllerImplementation)).To(Succeed())
}

func updateAPI(projectDir string) {
	typesFile := filepath.Join(projectDir, "api", "v1", "testoperator_types.go")
	err := util.ReplaceInFile(typesFile, "Foo string `json:\"foo,omitempty\"`", customField)
	Expect(err).NotTo(HaveOccurred(), "Failed to update testoperator_types.go")
}

func initializeGitRepo(projectDir string) {
	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
		{"git", "add", "-A"},
		{"git", "commit", "-m", "Initial project with custom code"},
		{"git", "checkout", "-b", "main"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = projectDir
		_, err := cmd.CombinedOutput()
		if err != nil && strings.Contains(err.Error(), "already exists") {
			Expect(exec.Command("git", "checkout", "main").Run()).To(Succeed())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

func validateCustomCodePreservation(projectDir string) {
	apiFile := filepath.Join(projectDir, "api", "v1", "testoperator_types.go")
	controllerFile := filepath.Join(projectDir, "internal", "controller", "testoperator_controller.go")

	apiContent, err := os.ReadFile(apiFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(apiContent)).To(ContainSubstring("Size int32 `json:\"size,omitempty\"`"))
	Expect(string(apiContent)).To(ContainSubstring("// +kubebuilder:validation:Minimum=0"))
	Expect(string(apiContent)).To(ContainSubstring("// +kubebuilder:validation:Maximum=3"))
	Expect(string(apiContent)).To(ContainSubstring("// +kubebuilder:default=1"))

	controllerContent, err := os.ReadFile(controllerFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(controllerContent)).To(ContainSubstring(controllerImplementation))
}

func hasConflictMarkers(projectDir string) bool {
	hasMarker := false

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil || bytes.Contains(content, []byte{0}) {
			return nil // skip unreadable or binary files
		}

		if strings.Contains(string(content), "<<<<<<<") {
			hasMarker = true
			return fmt.Errorf("conflict marker found in %s", path) // short-circuit early
		}
		return nil
	})

	if err != nil && hasMarker {
		return true
	}
	return hasMarker
}

func validateConflictState(projectDir string) {
	By("validating merge stopped with conflicts requiring manual resolution")

	// 1. Check file contents for conflict markers
	Expect(hasConflictMarkers(projectDir)).To(BeTrue())

	// 2. Check Git status for conflict-tracked files (UU = both modified)
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	lines := strings.Split(string(output), "\n")
	conflictFound := false
	for _, line := range lines {
		if strings.HasPrefix(line, "UU ") || strings.HasPrefix(line, "AA ") {
			conflictFound = true
			break
		}
	}
	Expect(conflictFound).To(BeTrue(), "Expected Git to report conflict state in files")
}

func expectModuleVersion(projectDir, module, version string) {
	goModPath := filepath.Join(projectDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	Expect(err).NotTo(HaveOccurred(), "Failed to read go.mod")

	expected := fmt.Sprintf("%s %s", module, version)
	Expect(string(content)).To(ContainSubstring(expected),
		fmt.Sprintf("Expected to find: %s", expected))
}
