//go:build integration

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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GitHub PR Creation", func() {
	var (
		tmpDir  string
		mockGh  string
		logFile string
		oldPath string
		err     error
		opts    *Update
	)

	BeforeEach(func() {
		opts = &Update{
			FromVersion: "v4.6.0",
			ToVersion:   "v4.7.0",
			FromBranch:  "main",
		}

		// Create temporary directory to house fake bin executables
		tmpDir, err = os.MkdirTemp("", "temp-bin-pr")
		Expect(err).NotTo(HaveOccurred())

		// Common file to log command runs from the fake bin
		logFile = filepath.Join(tmpDir, "bin.log")

		// Create fake gh executable
		mockGh = filepath.Join(tmpDir, "gh")

		// Prepend temp bin directory to PATH env
		oldPath = os.Getenv("PATH")
		Expect(os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)).To(Succeed())
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
		_ = os.Setenv("PATH", oldPath)
	})

	Context("openGitHubPR without conflicts", func() {
		It("Should create PR with basic description", func() {
			bodyFile := filepath.Join(tmpDir, "pr-body-basic.txt")
			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo 'owner/repo'
elif [[ "$1" == "pr" && "$2" == "create" ]]; then
    echo "$@" >> "` + logFile + `"
    # Capture stdin (PR body) to separate file
    cat - > "` + bodyFile + `"
    echo "https://github.com/owner/repo/pull/123"
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			_, err = opts.openGitHubPR(false)
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			logStr := string(logs)

			// Verify pr create was called with correct args
			Expect(logStr).To(ContainSubstring("pr create"))
			Expect(logStr).To(ContainSubstring("--repo owner/repo"))
			Expect(logStr).To(ContainSubstring("--base main"))
			Expect(logStr).To(ContainSubstring("--title chore: upgrade scaffold from v4.6.0 to v4.7.0"))

			// Verify body content
			body, bodyErr := os.ReadFile(bodyFile)
			Expect(bodyErr).ToNot(HaveOccurred())
			bodyStr := string(body)

			Expect(bodyStr).To(ContainSubstring("Upgrade project to use the scaffold changes"))
			Expect(bodyStr).To(ContainSubstring("Kubebuilder [v4.7.0]"))

			// Should NOT contain conflict warning
			Expect(bodyStr).ToNot(ContainSubstring("Conflicts were detected"))
		})

		It("Should create PR with conflict warning when conflicts exist", func() {
			bodyFile := filepath.Join(tmpDir, "pr-body-conflict.txt")
			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo 'owner/repo'
elif [[ "$1" == "pr" && "$2" == "create" ]]; then
    echo "$@" >> "` + logFile + `"
    # Capture stdin (PR body) to separate file
    cat - > "` + bodyFile + `"
    echo "https://github.com/owner/repo/pull/124"
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			_, err = opts.openGitHubPR(true)
			Expect(err).ToNot(HaveOccurred())

			// Verify conflict warning is present in PR body
			body, bodyErr := os.ReadFile(bodyFile)
			Expect(bodyErr).ToNot(HaveOccurred())
			bodyStr := string(body)

			Expect(bodyStr).To(ContainSubstring("Conflicts were detected during the merge"))
			Expect(bodyStr).To(ContainSubstring("alpha update"))
			Expect(bodyStr).To(ContainSubstring("https://kubebuilder.io/reference/commands/alpha_update"))
		})
	})

	Context("openGitHubPR with AI models", func() {
		BeforeEach(func() {
			opts.UseGhModels = true
		})

		It("Should append AI summary to PR description", func() {
			bodyFile := filepath.Join(tmpDir, "pr-body-ai.txt")
			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo 'owner/repo'
elif [[ "$1" == "models" && "$2" == "run" ]]; then
    echo "## ( 🤖 AI generate ) Scaffold Changes Overview"
    echo "This is a Kubebuilder scaffold update..."
    echo "- Go toolchain bumped to 1.24.0"
elif [[ "$1" == "pr" && "$2" == "create" ]]; then
    echo "$@" >> "` + logFile + `"
    # Capture stdin (PR body) to separate file
    cat - > "` + bodyFile + `"
    echo "https://github.com/owner/repo/pull/125"
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			_, err = opts.openGitHubPR(false)
			Expect(err).ToNot(HaveOccurred())

			// Verify AI summary is appended to PR body
			body, bodyErr := os.ReadFile(bodyFile)
			Expect(bodyErr).ToNot(HaveOccurred())
			bodyStr := string(body)

			Expect(bodyStr).To(ContainSubstring("AI generate"))
			Expect(bodyStr).To(ContainSubstring("Scaffold Changes Overview"))
		})

		It("Should handle AI generation failure gracefully", func() {
			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo 'owner/repo'
elif [[ "$1" == "models" && "$2" == "run" ]]; then
    echo "Error: AI generation failed" >&2
    exit 1
elif [[ "$1" == "pr" && "$2" == "create" ]]; then
    echo "$@" >> "` + logFile + `"
    echo "https://github.com/owner/repo/pull/126"
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			// Should not fail even if AI generation fails
			_, err = opts.openGitHubPR(false)
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			logStr := string(logs)

			// PR should still be created
			Expect(logStr).To(ContainSubstring("pr create"))
		})
	})

	Context("openGitHubPR error handling", func() {
		It("Should fail when gh repo view fails", func() {
			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo "Error: not a git repository" >&2
    exit 1
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			_, err = opts.openGitHubPR(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to detect GitHub repository"))
		})

		It("Should fail when gh pr create fails", func() {
			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo 'owner/repo'
elif [[ "$1" == "pr" && "$2" == "create" ]]; then
    echo "Error: head ref must be a branch" >&2
    exit 1
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			_, err = opts.openGitHubPR(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create GitHub pull request"))
		})
	})

	Context("PR description format", func() {
		It("Should include release notes links", func() {
			bodyFile := filepath.Join(tmpDir, "pr-body-links.txt")
			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo 'owner/repo'
elif [[ "$1" == "pr" && "$2" == "create" ]]; then
    echo "$@" >> "` + logFile + `"
    # Capture stdin (PR body) to separate file
    cat - > "` + bodyFile + `"
    echo "https://github.com/owner/repo/pull/127"
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			_, err = opts.openGitHubPR(false)
			Expect(err).ToNot(HaveOccurred())

			// Verify release notes links in PR body
			body, bodyErr := os.ReadFile(bodyFile)
			Expect(bodyErr).ToNot(HaveOccurred())
			bodyStr := string(body)
			Expect(bodyStr).To(ContainSubstring("https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v4.6.0"))
			Expect(bodyStr).To(ContainSubstring("https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v4.7.0"))
		})

		It("Should use custom output branch name if specified", func() {
			opts.OutputBranch = "kubebuilder-update-from-v1.0.0-to-v2.0.0-custom"

			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo 'owner/repo'
elif [[ "$1" == "pr" && "$2" == "create" ]]; then
    echo "$@" >> "` + logFile + `"
    echo "https://github.com/owner/repo/pull/128"
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			_, err = opts.openGitHubPR(false)
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			logStr := string(logs)

			// Verify custom branch is used
			Expect(logStr).To(ContainSubstring("--head kubebuilder-update-from-v1.0.0-to-v2.0.0-custom"))
		})
	})

	Context("Branch name security validation", func() {
		It("Should reject branch names without required prefix", func() {
			opts.OutputBranch = "my-custom-branch"

			_, err = opts.openGitHubPR(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("security"))
			Expect(err.Error()).To(ContainSubstring("kubebuilder-update-from-"))
		})

		It("Should accept branch names with required prefix", func() {
			opts.OutputBranch = "kubebuilder-update-from-v4.0.0-to-v4.1.0"

			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo 'owner/repo'
elif [[ "$1" == "pr" && "$2" == "create" ]]; then
    echo "https://github.com/owner/repo/pull/129"
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			_, err = opts.openGitHubPR(false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should use default branch name format which has required prefix", func() {
			opts.FromVersion = "v4.0.0"
			opts.ToVersion = "v4.1.0"
			opts.OutputBranch = "" // Use default

			script := `#!/bin/bash
if [[ "$1" == "repo" && "$2" == "view" ]]; then
    echo 'owner/repo'
elif [[ "$1" == "pr" && "$2" == "create" ]]; then
    echo "https://github.com/owner/repo/pull/130"
fi
exit 0`
			err = mockBinResponse(script, mockGh)
			Expect(err).NotTo(HaveOccurred())

			_, err = opts.openGitHubPR(false)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
