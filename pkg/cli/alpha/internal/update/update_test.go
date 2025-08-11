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

package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/gock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Mock response for binary executables
func mockBinResponse(script, mockBin string) error {
	err := os.WriteFile(mockBin, []byte(script), 0o755)
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return fmt.Errorf("Error Mocking bin response: %w", err)
	}
	return nil
}

// Mock response from an url
func mockURLResponse(body, url string, times, reply int) {
	urlStrings := strings.Split(url, "/")
	gockNew := strings.Join(urlStrings[0:3], "/")
	get := "/" + strings.Join(urlStrings[3:], "/")
	gock.New(gockNew).
		Get(get).
		Times(times).
		Reply(reply).
		Body(strings.NewReader(body))
}

var _ = Describe("Prepare for internal update", func() {
	var (
		tmpDir   string
		mockGit  string
		mockMake string
		mocksh   string
		logFile  string
		oldPath  string
		err      error
		opts     Update
	)

	BeforeEach(func() {
		opts = Update{
			FromVersion: "v4.5.0",
			ToVersion:   "v4.6.0",
			FromBranch:  "main",
		}

		// Create temporary directory to house fake bin executables
		tmpDir, err = os.MkdirTemp("", "temp-bin")
		Expect(err).NotTo(HaveOccurred())

		// Create a common file to log the command runs from the fake bin
		logFile = filepath.Join(tmpDir, "bin.log")

		// Create fake bin executables
		mockGit = filepath.Join(tmpDir, "git")
		mockMake = filepath.Join(tmpDir, "make")
		mocksh = filepath.Join(tmpDir, "sh")
		script := `#!/bin/bash
            echo "$@" >> "` + logFile + `"
           exit 0`
		err = mockBinResponse(script, mockGit)
		Expect(err).NotTo(HaveOccurred())
		err = mockBinResponse(script, mockMake)
		Expect(err).NotTo(HaveOccurred())
		err = mockBinResponse(script, mocksh)
		Expect(err).NotTo(HaveOccurred())

		// Prepend temp bin directory to PATH env
		oldPath = os.Getenv("PATH")
		err = os.Setenv("PATH", tmpDir+":"+oldPath)
		Expect(err).NotTo(HaveOccurred())

		// Mock response from "https://github.com/kubernetes-sigs/kubebuilder/releases/download"
		mockURLResponse(script, "https://github.com/kubernetes-sigs/kubebuilder/releases/download", 2, 200)
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
		_ = os.Setenv("PATH", oldPath)
		defer gock.Off()
	})

	Context("Update", func() {
		It("Should scucceed updating project using a default three-way Git merge", func() {
			err = opts.Update()
			Expect(err).ToNot(HaveOccurred())
			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring(fmt.Sprintf("checkout %s", opts.FromBranch)))
		})
		It("Should fail when git command fails", func() {
			fakeBinScript := `#!/bin/bash
			       echo "$@" >> "` + logFile + `"
			       exit 1`
			err = mockBinResponse(fakeBinScript, mockGit)
			Expect(err).ToNot(HaveOccurred())
			err = opts.Update()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to checkout base branch %s", opts.FromBranch))

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring(fmt.Sprintf("checkout %s", opts.FromBranch)))
		})
		It("Should fail when kubebuilder binary could not be downloaded", func() {
			gock.Off()

			// mockURLResponse(fakeBinScript, "https://github.com/kubernetes-sigs/kubebuilder/releases/download", 2, 401)
			gock.New("https://github.com").
				Get("/kubernetes-sigs/kubebuilder/releases/download").
				Times(2).
				Reply(401).
				Body(strings.NewReader(""))

			err = opts.Update()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to prepare ancestor branch"))
			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring(fmt.Sprintf("checkout %s", opts.FromBranch)))
		})
	})

	Context("RegenerateProjectWithVersion", func() {
		It("Should scucceed downloading release binary and running `alpha generate`", func() {
			err = regenerateProjectWithVersion(opts.FromBranch)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail downloading release binary", func() {
			// mockURLResponse(fakeBinScript, "https://github.com/kubernetes-sigs/kubebuilder/releases/download", 2, 401)
			gock.Off()
			gock.New("https://github.com").
				Get("/kubernetes-sigs/kubebuilder/releases/download").
				Times(2).
				Reply(401).
				Body(strings.NewReader(""))

			err = regenerateProjectWithVersion(opts.FromBranch)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to download release %s binary", opts.FromBranch))
		})

		It("Should fail running alpha generate", func() {
			// mockURLResponse(fakeBinScript, "https://github.com/kubernetes-sigs/kubebuilder/releases/download", 2, 200)
			fakeBinScript := `#!/bin/bash
			       echo "$@" >> "` + logFile + `"
			       exit 1`
			gock.Off()
			gock.New("https://github.com").
				Get("/kubernetes-sigs/kubebuilder/releases/download").
				Times(2).
				Reply(200).
				Body(strings.NewReader(fakeBinScript))

			err = regenerateProjectWithVersion(opts.FromBranch)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to run alpha generate on ancestor branch"))
		})
	})

	verifyLogs := func(newBranch, oldBranch, fromVersion string) {
		logs, readErr := os.ReadFile(logFile)
		Expect(readErr).NotTo(HaveOccurred())
		Expect(string(logs)).To(ContainSubstring("checkout -b %s %s", newBranch, oldBranch))
		Expect(string(logs)).To(ContainSubstring("checkout %s", newBranch))
		Expect(string(logs)).To(ContainSubstring(
			"-c find . -mindepth 1 -maxdepth 1 ! -name '.git' ! -name 'PROJECT' -exec rm -rf {}"))
		Expect(string(logs)).To(ContainSubstring("alpha generate"))
		Expect(string(logs)).To(ContainSubstring("add --all"))
		Expect(string(logs)).To(ContainSubstring("commit -m Clean scaffolding from release version: %s", fromVersion))
	}

	Context("PrepareAncestorBranch", func() {
		It("Should scucceed to prepare the ancestor branch", func() {
			err = opts.prepareAncestorBranch()
			Expect(err).ToNot(HaveOccurred())
			verifyLogs(opts.AncestorBranch, opts.FromBranch, opts.FromVersion)
		})

		It("Should fail to prepare the ancestor branch", func() {
			fakeBinScript := `#!/bin/bash
			       echo "$@" >> "` + logFile + `"
			       exit 1`
			err = mockBinResponse(fakeBinScript, mockGit)
			Expect(err).ToNot(HaveOccurred())
			err = opts.prepareAncestorBranch()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create %s from %s", opts.AncestorBranch, opts.FromBranch))
		})
	})

	Context("PrepareUpgradeBranch", func() {
		It("Should scucceed PrepareUpgradeBranch", func() {
			err = opts.prepareUpgradeBranch()
			Expect(err).ToNot(HaveOccurred())
			verifyLogs(opts.UpgradeBranch, opts.AncestorBranch, opts.ToVersion)
		})

		It("Should fail PrepareUpgradeBranch", func() {
			fakeBinScript := `#!/bin/bash
							echo "$@" >> "` + logFile + `"
							exit 1`
			err = mockBinResponse(fakeBinScript, mockGit)
			Expect(err).ToNot(HaveOccurred())
			err = opts.prepareUpgradeBranch()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"failed to checkout %s branch off %s", opts.UpgradeBranch, opts.AncestorBranch))
		})
	})

	Context("BinaryWithVersion", func() {
		It("Should scucceed to download the specified released version from GitHub releases", func() {
			_, err = binaryWithVersion(opts.FromVersion)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail to download the specified released version from GitHub releases", func() {
			// mockURLResponse(fakeBinScript, "https://github.com/kubernetes-sigs/kubebuilder/releases/download", 2, 401)
			gock.Off()
			gock.New("https://github.com").
				Get("/kubernetes-sigs/kubebuilder/releases/download").
				Times(2).
				Reply(401).
				Body(strings.NewReader(""))

			_, err = binaryWithVersion(opts.FromVersion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to download the binary: HTTP 401"))
		})
	})

	Context("CleanupBranch", func() {
		It("Should scucceed executing cleanup command", func() {
			err = cleanupBranch()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail executing cleanup command", func() {
			fakeBinScript := `#!/bin/bash
			       echo "$@" >> "` + logFile + `"
			       exit 1`
			err = mockBinResponse(fakeBinScript, mocksh)
			Expect(err).ToNot(HaveOccurred())
			err = cleanupBranch()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to clean up files"))
		})
	})

	Context("RunMakeTargets", func() {
		It("Should fail to run make commands", func() {
			fakeBinScript := `#!/bin/bash
			       echo "$@" >> "` + logFile + `"
			       exit 1`
			err = mockBinResponse(fakeBinScript, mockMake)
			Expect(err).ToNot(HaveOccurred())

			runMakeTargets()
		})
	})

	Context("RunAlphaGenerate", func() {
		It("Should scucceed runAlphaGenerate", func() {
			mockKubebuilder := filepath.Join(tmpDir, "kubebuilder")
			KubebuilderScript := `#!/bin/bash
			       echo "$@" >> "` + logFile + `"
			       exit 0`
			err = mockBinResponse(KubebuilderScript, mockKubebuilder)
			Expect(err).NotTo(HaveOccurred())

			err = runAlphaGenerate(tmpDir, opts.FromBranch)
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring("alpha generate"))
		})

		It("Should fail runAlphaGenerate", func() {
			mockKubebuilder := filepath.Join(tmpDir, "kubebuilder")
			KubebuilderScript := `#!/bin/bash
			       echo "$@" >> "` + logFile + `"
			       exit 1`
			err = mockBinResponse(KubebuilderScript, mockKubebuilder)
			Expect(err).NotTo(HaveOccurred())

			err = runAlphaGenerate(tmpDir, opts.FromBranch)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to run alpha generate"))
		})
	})

	Context("PrepareOriginalBranch", func() {
		It("Should scucceed prepareOriginalBranch", func() {
			err = opts.prepareOriginalBranch()
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring("checkout -b %s", opts.OriginalBranch))
			Expect(string(logs)).To(ContainSubstring("checkout %s -- .", opts.FromBranch))
			Expect(string(logs)).To(ContainSubstring("add --all"))
			Expect(string(logs)).To(ContainSubstring(
				"commit -m Add code from %s into %s", opts.FromBranch, opts.OriginalBranch))
		})

		It("Should fail prepareOriginalBranch", func() {
			fakeBinScript := `#!/bin/bash
							echo "$@" >> "` + logFile + `"
							exit 1`
			err = mockBinResponse(fakeBinScript, mockGit)
			Expect(err).ToNot(HaveOccurred())
			err = opts.prepareOriginalBranch()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to checkout branch %s", opts.OriginalBranch))
		})
	})

	Context("MergeOriginalToUpgrade", func() {
		It("Should scucceed MergeOriginalToUpgrade", func() {
			err = opts.mergeOriginalToUpgrade()
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring("checkout -b %s %s", opts.MergeBranch, opts.UpgradeBranch))
			Expect(string(logs)).To(ContainSubstring("checkout %s", opts.MergeBranch))
			Expect(string(logs)).To(ContainSubstring("merge --no-edit --no-commit %s", opts.OriginalBranch))
			Expect(string(logs)).To(ContainSubstring("add --all"))
			Expect(string(logs)).To(ContainSubstring("Merge from %s to %s.", opts.FromVersion, opts.ToVersion))
			Expect(string(logs)).To(ContainSubstring("Merge happened without conflicts"))
		})

		It("Should fail MergeOriginalToUpgrade", func() {
			fakeBinScript := `#!/bin/bash
							echo "$@" >> "` + logFile + `"
							exit 1`
			err = mockBinResponse(fakeBinScript, mockGit)
			Expect(err).ToNot(HaveOccurred())
			err = opts.mergeOriginalToUpgrade()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"failed to create merge branch %s from %s", opts.MergeBranch, opts.OriginalBranch))
		})
	})

	Context("SquashToOutputBranch", func() {
		BeforeEach(func() {
			opts.FromBranch = "main"
			opts.ToVersion = "v4.6.0"
			if opts.MergeBranch == "" {
				opts.MergeBranch = "tmp-merge-test"
			}
		})

		It("should create/reset the output branch and commit one squashed snapshot", func() {
			opts.OutputBranch = ""
			opts.PreservePath = []string{".github/workflows"} // exercise the restore call

			err = opts.squashToOutputBranch()
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			s := string(logs)

			Expect(s).To(ContainSubstring(fmt.Sprintf("checkout %s", opts.FromBranch)))
			Expect(s).To(ContainSubstring(fmt.Sprintf(
				"checkout -B kubebuilder-alpha-update-to-%s %s",
				opts.ToVersion, opts.FromBranch,
			)))
			Expect(s).To(ContainSubstring(
				"-c find . -mindepth 1 -maxdepth 1 ! -name '.git' -exec rm -rf {} +",
			))
			Expect(s).To(ContainSubstring(fmt.Sprintf("checkout %s -- .", opts.MergeBranch)))
			Expect(s).To(ContainSubstring(fmt.Sprintf(
				"restore --source %s --staged --worktree .github/workflows",
				opts.FromBranch,
			)))
			Expect(s).To(ContainSubstring("add --all"))

			msg := fmt.Sprintf(
				"[kubebuilder-automated-update]: update scaffold from %s to %s; (squashed 3-way merge)",
				opts.FromVersion, opts.ToVersion,
			)
			Expect(s).To(ContainSubstring(msg))

			Expect(s).To(ContainSubstring("commit --no-verify -m"))
		})

		It("should respect a custom output branch name", func() {
			opts.OutputBranch = "my-custom-branch"
			err = opts.squashToOutputBranch()
			Expect(err).ToNot(HaveOccurred())

			logs, _ := os.ReadFile(logFile)
			Expect(string(logs)).To(ContainSubstring(
				fmt.Sprintf("checkout -B %s %s", "my-custom-branch", opts.FromBranch),
			))
		})

		It("squash: no changes -> commit exits 1 but returns nil", func() {
			fake := `#!/bin/bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "commit" ]]; then exit 1; fi
exit 0`
			Expect(mockBinResponse(fake, mockGit)).To(Succeed())

			opts.PreservePath = nil
			Expect(opts.squashToOutputBranch()).To(Succeed())

			s, _ := os.ReadFile(logFile)
			Expect(string(s)).To(ContainSubstring("commit --no-verify -m"))
		})

		It("squash: trims preserve-path and skips blanks", func() {
			opts.PreservePath = []string{" .github/workflows ", "", "docs"}
			Expect(opts.squashToOutputBranch()).To(Succeed())
			s, _ := os.ReadFile(logFile)
			Expect(string(s)).To(ContainSubstring("restore --source main --staged --worktree .github/workflows"))
			Expect(string(s)).To(ContainSubstring("restore --source main --staged --worktree docs"))
		})

		It("update: runs squash when --squash is set", func() {
			opts.Squash = true
			Expect(opts.Update()).To(Succeed())
			s, _ := os.ReadFile(logFile)
			Expect(string(s)).To(ContainSubstring("checkout -B kubebuilder-alpha-update-to-" + opts.ToVersion + " main"))
			Expect(string(s)).To(ContainSubstring("-c find . -mindepth 1"))
			Expect(string(s)).To(ContainSubstring("checkout " + opts.MergeBranch + " -- ."))
		})
	})
})
