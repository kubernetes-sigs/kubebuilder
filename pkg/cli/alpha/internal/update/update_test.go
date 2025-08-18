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

	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal/update/helpers"
)

// Helpers to keep lines short and consistent with production messages.
func expNormalMsg(from, to string) string {
	return fmt.Sprintf("(chore) scaffold update: %s -> %s", from, to)
}

func expConflictMsg(from, to string) string {
	return fmt.Sprintf(":warning: (chore) [with conflicts] scaffold update: %s -> %s", from, to)
}

// Mock response for binary executables.
func mockBinResponse(script, mockBin string) error {
	err := os.WriteFile(mockBin, []byte(script), 0o755)
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return fmt.Errorf("error Mocking bin response: %w", err)
	}
	return nil
}

// Mock response from an URL.
func mockURLResponse(body, url string, times, reply int) {
	parts := strings.Split(url, "/")
	host := strings.Join(parts[0:3], "/")
	path := "/" + strings.Join(parts[3:], "/")
	gock.New(host).Get(path).Times(times).Reply(reply).Body(strings.NewReader(body))
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
			FromBranch:  defaultBranch,
		}

		// Create temporary directory to house fake bin executables.
		tmpDir, err = os.MkdirTemp("", "temp-bin")
		Expect(err).NotTo(HaveOccurred())

		// Common file to log command runs from the fake bin.
		logFile = filepath.Join(tmpDir, "bin.log")

		// Create fake bin executables.
		mockGit = filepath.Join(tmpDir, "git")
		mockMake = filepath.Join(tmpDir, "make")
		mocksh = filepath.Join(tmpDir, "sh")
		script := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 0`
		Expect(mockBinResponse(script, mockGit)).To(Succeed())
		Expect(mockBinResponse(script, mockMake)).To(Succeed())
		Expect(mockBinResponse(script, mocksh)).To(Succeed())

		// Prepend temp bin directory to PATH env.
		oldPath = os.Getenv("PATH")
		Expect(os.Setenv("PATH", tmpDir+":"+oldPath)).To(Succeed())

		// Mock GitHub release download.
		mockURLResponse(script,
			"https://github.com/kubernetes-sigs/kubebuilder/releases/download", 2, 200)
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
		_ = os.Setenv("PATH", oldPath)
		defer gock.Off()
	})

	// Helper that formats the expectations properly.
	verifyLogs := func(newBranch, oldBranch, fromVersion string) {
		logs, readErr := os.ReadFile(logFile)
		Expect(readErr).NotTo(HaveOccurred())
		s := string(logs)

		Expect(s).To(ContainSubstring(
			fmt.Sprintf("checkout -b %s %s", newBranch, oldBranch),
		))
		Expect(s).To(ContainSubstring(fmt.Sprintf("checkout %s", newBranch)))
		Expect(s).To(ContainSubstring(
			"-c find . -mindepth 1 -maxdepth 1 ! -name '.git' ! -name 'PROJECT' -exec rm -rf {}",
		))
		Expect(s).To(ContainSubstring("alpha generate"))
		Expect(s).To(ContainSubstring("add --all"))
		Expect(s).To(ContainSubstring(
			fmt.Sprintf("initial scaffold from release version: %s", fromVersion),
		))
	}

	Context("Update", func() {
		It("succeeds using a default three-way Git merge", func() {
			err = opts.Update()
			Expect(err).ToNot(HaveOccurred())
			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring(
				fmt.Sprintf("checkout %s", opts.FromBranch),
			))
		})

		It("fails when git command fails", func() {
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			Expect(mockBinResponse(fail, mockGit)).To(Succeed())

			err = opts.Update()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("failed to checkout base branch %s", opts.FromBranch),
			))

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring(
				fmt.Sprintf("checkout %s", opts.FromBranch),
			))
		})

		It("fails when kubebuilder binary cannot be downloaded", func() {
			gock.Off()
			gock.New("https://github.com").
				Get("/kubernetes-sigs/kubebuilder/releases/download").
				Times(2).Reply(401).Body(strings.NewReader(""))

			err = opts.Update()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to prepare ancestor branch"))

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring(
				fmt.Sprintf("checkout %s", opts.FromBranch),
			))
		})
	})

	Context("RegenerateProjectWithVersion", func() {
		It("succeeds downloading binary and running `alpha generate`", func() {
			err = regenerateProjectWithVersion(opts.FromVersion)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails downloading binary", func() {
			gock.Off()
			gock.New("https://github.com").
				Get("/kubernetes-sigs/kubebuilder/releases/download").
				Times(2).Reply(401).Body(strings.NewReader(""))

			err = regenerateProjectWithVersion(opts.FromVersion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("failed to download release %s binary", opts.FromVersion),
			))
		})

		It("fails running alpha generate", func() {
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			gock.Off()
			gock.New("https://github.com").
				Get("/kubernetes-sigs/kubebuilder/releases/download").
				Times(2).Reply(200).Body(strings.NewReader(fail))

			err = regenerateProjectWithVersion(opts.FromVersion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"failed to run alpha generate on ancestor branch",
			))
		})
	})

	Context("PrepareAncestorBranch", func() {
		It("succeeds", func() {
			err = opts.prepareAncestorBranch()
			Expect(err).ToNot(HaveOccurred())
			verifyLogs(opts.AncestorBranch, opts.FromBranch, opts.FromVersion)
		})

		It("fails creating branch", func() {
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			Expect(mockBinResponse(fail, mockGit)).To(Succeed())

			err = opts.prepareAncestorBranch()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("failed to create %s from %s",
					opts.AncestorBranch, opts.FromBranch),
			))
		})
	})

	Context("PrepareUpgradeBranch", func() {
		It("succeeds", func() {
			err = opts.prepareUpgradeBranch()
			Expect(err).ToNot(HaveOccurred())
			verifyLogs(opts.UpgradeBranch, opts.AncestorBranch, opts.ToVersion)
		})

		It("fails creating branch", func() {
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			Expect(mockBinResponse(fail, mockGit)).To(Succeed())

			err = opts.prepareUpgradeBranch()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("failed to checkout %s branch off %s",
					opts.UpgradeBranch, opts.AncestorBranch),
			))
		})
	})

	Context("BinaryWithVersion", func() {
		It("succeeds to download the specified released version", func() {
			_, err = helpers.DownloadReleaseVersionWith(opts.FromVersion)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails to download the specified released version", func() {
			gock.Off()
			gock.New("https://github.com").
				Get("/kubernetes-sigs/kubebuilder/releases/download").
				Times(2).Reply(401).Body(strings.NewReader(""))

			_, err = helpers.DownloadReleaseVersionWith(opts.FromVersion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to download the binary: HTTP 401"))
		})
	})

	Context("CleanupBranch", func() {
		It("succeeds executing cleanup command", func() {
			err = cleanupBranch()
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails executing cleanup command", func() {
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			Expect(mockBinResponse(fail, mocksh)).To(Succeed())

			err = cleanupBranch()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to clean up files"))
		})
	})

	Context("RunMakeTargets", func() {
		It("logs warning when make fails", func() {
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			Expect(mockBinResponse(fail, mockMake)).To(Succeed())

			// Should not panic even if make fails; just logs a warning.
			runMakeTargets()
		})
	})

	Context("RunAlphaGenerate", func() {
		It("succeeds", func() {
			mockKB := filepath.Join(tmpDir, "kubebuilder")
			script := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 0`
			Expect(mockBinResponse(script, mockKB)).To(Succeed())

			err = runAlphaGenerate(tmpDir, opts.FromVersion)
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring("alpha generate"))
		})

		It("fails", func() {
			mockKB := filepath.Join(tmpDir, "kubebuilder")
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			Expect(mockBinResponse(fail, mockKB)).To(Succeed())

			err = runAlphaGenerate(tmpDir, opts.FromVersion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to run alpha generate"))
		})
	})

	Context("PrepareOriginalBranch", func() {
		It("succeeds", func() {
			err = opts.prepareOriginalBranch()
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			s := string(logs)

			Expect(s).To(ContainSubstring(
				fmt.Sprintf("checkout -b %s", opts.OriginalBranch),
			))
			Expect(s).To(ContainSubstring(
				fmt.Sprintf("checkout %s -- .", opts.FromBranch),
			))
			Expect(s).To(ContainSubstring("add --all"))
			Expect(s).To(ContainSubstring(
				fmt.Sprintf("original code from %s to keep changes", opts.FromBranch),
			))
		})

		It("fails", func() {
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			Expect(mockBinResponse(fail, mockGit)).To(Succeed())

			err = opts.prepareOriginalBranch()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("failed to checkout branch %s", opts.OriginalBranch),
			))
		})
	})

	Context("MergeOriginalToUpgrade", func() {
		BeforeEach(func() {
			// deterministic names for merge test
			opts.UpgradeBranch = "tmp-upgrade-X"
			opts.MergeBranch = "tmp-merge-X"
			opts.OriginalBranch = "tmp-original-X"
		})

		It("succeeds and commits with normal message", func() {
			_, err = opts.mergeOriginalToUpgrade()
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			s := string(logs)

			Expect(s).To(ContainSubstring(
				fmt.Sprintf("checkout -b %s %s", opts.MergeBranch, opts.UpgradeBranch),
			))
			Expect(s).To(ContainSubstring(fmt.Sprintf("checkout %s", opts.MergeBranch)))
			Expect(s).To(ContainSubstring(
				fmt.Sprintf("merge --no-edit --no-commit %s", opts.OriginalBranch),
			))
			Expect(s).To(ContainSubstring("add --all"))
			Expect(s).To(ContainSubstring(expNormalMsg(opts.FromVersion, opts.ToVersion)))
		})

		It("fails when branch creation fails", func() {
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			Expect(mockBinResponse(fail, mockGit)).To(Succeed())

			_, err = opts.mergeOriginalToUpgrade()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("failed to create merge branch %s from %s",
					opts.MergeBranch, opts.UpgradeBranch),
			))
		})

		It("stops on conflicts when Force=false", func() {
			failOnMerge := `#!/bin/bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "merge" ]]; then exit 1; fi
exit 0`
			Expect(mockBinResponse(failOnMerge, mockGit)).To(Succeed())

			opts.Force = false
			_, err = opts.mergeOriginalToUpgrade()
			Expect(err).To(HaveOccurred())

			s, _ := os.ReadFile(logFile)
			Expect(string(s)).NotTo(ContainSubstring("commit --no-verify -m"))
		})

		It("commits with conflict message when Force=true", func() {
			failOnMerge := `#!/bin/bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "merge" ]]; then exit 1; fi
exit 0`
			Expect(mockBinResponse(failOnMerge, mockGit)).To(Succeed())

			opts.Force = true
			_, err = opts.mergeOriginalToUpgrade()
			Expect(err).ToNot(HaveOccurred())

			s, _ := os.ReadFile(logFile)
			Expect(string(s)).To(ContainSubstring(
				expConflictMsg(opts.FromVersion, opts.ToVersion),
			))
		})
	})

	Context("SquashToOutputBranch", func() {
		BeforeEach(func() {
			opts.FromBranch = defaultBranch
			opts.FromVersion = "v4.5.0"
			opts.ToVersion = "v4.6.0"
			if opts.MergeBranch == "" {
				opts.MergeBranch = "tmp-merge-test"
			}
		})

		It("creates/resets output branch and commits one squashed snapshot", func() {
			opts.OutputBranch = "" // default naming
			opts.PreservePath = []string{".github/workflows"}
			opts.ShowCommits = false

			err = opts.squashToOutputBranch(false) // no conflicts
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			s := string(logs)

			Expect(s).To(ContainSubstring(fmt.Sprintf("checkout %s", opts.FromBranch)))

			expOut := fmt.Sprintf(
				"checkout -B %s %s",
				fmt.Sprintf("kubebuilder-update-from-%s-to-%s",
					opts.FromVersion, opts.ToVersion),
				opts.FromBranch,
			)
			Expect(s).To(ContainSubstring(expOut))

			Expect(s).To(ContainSubstring(fmt.Sprintf("checkout %s -- .", opts.MergeBranch)))
			Expect(s).To(ContainSubstring("add --all"))
			Expect(s).To(ContainSubstring(expNormalMsg(opts.FromVersion, opts.ToVersion)))
			Expect(s).To(ContainSubstring("commit --no-verify -m"))
		})

		It("respects a custom output branch name", func() {
			opts.OutputBranch = "my-custom-branch"

			err = opts.squashToOutputBranch(false)
			Expect(err).ToNot(HaveOccurred())

			logs, _ := os.ReadFile(logFile)
			Expect(string(logs)).To(ContainSubstring(
				fmt.Sprintf("checkout -B %s %s", "my-custom-branch", opts.FromBranch),
			))
		})

		It("no changes -> commit exits 1 but helper returns nil", func() {
			fake := `#!/bin/bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "commit" ]]; then exit 1; fi
exit 0`
			Expect(mockBinResponse(fake, mockGit)).To(Succeed())

			opts.PreservePath = nil
			Expect(opts.squashToOutputBranch(false)).To(Succeed())

			s, _ := os.ReadFile(logFile)
			Expect(string(s)).To(ContainSubstring("commit --no-verify -m"))
		})

		It("trims preserve-path and skips blanks", func() {
			opts.PreservePath = []string{" .github/workflows ", "", "docs"}
			Expect(opts.squashToOutputBranch(false)).To(Succeed())

			s, _ := os.ReadFile(logFile)
			Expect(string(s)).To(ContainSubstring("checkout main -- docs"))
			Expect(string(s)).To(ContainSubstring("checkout main -- .github/workflows"))
		})
	})

	Context("getOutputBranchName", func() {
		It("returns default name when OutputBranch is empty", func() {
			const fromVersion = "v4.5.0"
			const toVersion = "v4.6.0"
			opts.FromVersion = fromVersion
			opts.ToVersion = toVersion
			opts.OutputBranch = ""

			want := fmt.Sprintf("kubebuilder-update-from-%s-to-%s", fromVersion, toVersion)
			Expect(opts.getOutputBranchName()).To(Equal(want))
		})

		It("returns custom name when OutputBranch is set", func() {
			opts.OutputBranch = "my-custom"
			Expect(opts.getOutputBranchName()).To(Equal("my-custom"))
		})
	})

	Context("runAlphaGenerate PATH restoration", func() {
		It("does not mutate process PATH (same even on failure)", func() {
			tmp := filepath.Join(tmpDir, "kubebuilder")
			fail := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			Expect(mockBinResponse(fail, tmp)).To(Succeed())

			orig := os.Getenv("PATH")
			err := runAlphaGenerate(tmpDir, "v4.5.0")
			Expect(err).To(HaveOccurred())
			Expect(os.Getenv("PATH")).To(Equal(orig))
		})
	})
})
