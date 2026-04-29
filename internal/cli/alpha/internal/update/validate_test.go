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
	"strings"

	"github.com/h2non/gock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prepare for internal update", func() {
	var (
		tmpDir  string
		mockGit string
		logFile string
		oldPath string
		err     error
		opts    *Update
	)

	BeforeEach(func() {
		opts = &Update{
			FromVersion:    "v4.5.0",
			ToVersion:      "v4.6.0",
			FromBranch:     defaultBranch,
			OriginalBranch: "v4.6.0",
		}

		// Create temporary directory to house fake bin executables
		tmpDir, err = os.MkdirTemp("", "temp-bin")
		Expect(err).NotTo(HaveOccurred())

		// Create a common file to log the command runs from the fake bin
		logFile = filepath.Join(tmpDir, "bin.log")

		// Create fake bin executables
		mockGit = filepath.Join(tmpDir, "git")
		script := `#!/bin/bash
            echo "$@" >> "` + logFile + `"
           exit 0`
		err = mockBinResponse(script, mockGit)
		Expect(err).NotTo(HaveOccurred())

		// Prepend temp bin directory to PATH env
		oldPath = os.Getenv("PATH")
		err = os.Setenv("PATH", tmpDir+":"+oldPath)
		Expect(err).NotTo(HaveOccurred())

		gock.New("https://github.com").
			Head("/kubernetes-sigs/kubebuilder/releases/download").
			Times(2).
			Reply(200).
			Body(strings.NewReader("body"))
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
		_ = os.Setenv("PATH", oldPath)
		defer gock.Off()
	})

	Context("Validate", func() {
		It("Should scucceed", func() {
			err = opts.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should fail", func() {
			fakeBinScript := `#!/bin/bash
			    	echo "$@" >> "` + logFile + `"
					exit 1`
			err = mockBinResponse(fakeBinScript, mockGit)
			Expect(err).ToNot(HaveOccurred())

			err = opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to validate git repository"))
		})
	})

	Context("ValidateGitRepo", func() {
		It("Should scucceed", func() {
			err = opts.validateGitRepo()
			Expect(err).ToNot(HaveOccurred())
			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring("rev-parse --git-dir"))
			Expect(string(logs)).To(ContainSubstring("status --porcelain"))
		})
		It("Should fail", func() {
			fakeBinScript := `#!/bin/bash
			    	echo "$@" >> "` + logFile + `"
					exit 1`
			err = mockBinResponse(fakeBinScript, mockGit)
			Expect(err).ToNot(HaveOccurred())

			err = opts.validateGitRepo()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not in a git repository"))
		})
	})

	Context("ValidateFromBranch", func() {
		It("Should scucceed", func() {
			err = opts.validateFromBranch()
			Expect(err).ToNot(HaveOccurred())
			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring("rev-parse --verify %s", opts.FromBranch))
		})
		It("Should fail", func() {
			fakeBinScript := `#!/bin/bash
			       echo "$@" >> "` + logFile + `"
			       exit 1`
			err = mockBinResponse(fakeBinScript, mockGit)
			Expect(err).ToNot(HaveOccurred())
			err := opts.validateFromBranch()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("branch does not exist locally"))
		})
	})

	Context("ValidateSemanticVersions", func() {
		It("Should scucceed", func() {
			err := opts.validateSemanticVersions()
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should fail", func() {
			opts.FromVersion = "6"
			err := opts.validateSemanticVersions()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("has invalid semantic version. Expect: vX.Y.Z"))
		})
	})

	Context("ValidateReleaseAvailability", func() {
		It("Should scucceed", func() {
			err := validateReleaseAvailability(opts.ToVersion)
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should fail", func() {
			gock.Off()
			gock.New("https://github.com").
				Head("/kubernetes-sigs/kubebuilder/releases/download").
				Reply(401).
				Body(strings.NewReader("body"))
			err := validateReleaseAvailability(opts.FromVersion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unexpected response"))
		})
	})

	Context("Validate flag combinations", func() {
		var (
			tmpBinDir  string
			mockGh     string
			oldPathEnv string
		)

		BeforeEach(func() {
			// Create temporary directory for mock gh binary
			tmpBinDir, err = os.MkdirTemp("", "validate-test-bin")
			Expect(err).ToNot(HaveOccurred())

			// Create fake gh executable
			mockGh = filepath.Join(tmpBinDir, "gh")

			// Stub gh to return success for --version, auth status, and extension list
			script := `#!/bin/bash
if [[ "$1" == "--version" ]]; then
    echo "gh version 2.0.0 (2023-01-01)"
elif [[ "$1" == "auth" && "$2" == "status" ]]; then
    exit 0
elif [[ "$1" == "extension" && "$2" == "list" ]]; then
    echo "gh models"
fi
exit 0`
			err = os.WriteFile(mockGh, []byte(script), 0o755)
			Expect(err).ToNot(HaveOccurred())

			// Prepend temp bin directory to PATH
			oldPathEnv = os.Getenv("PATH")
			Expect(os.Setenv("PATH", tmpBinDir+string(os.PathListSeparator)+oldPathEnv)).To(Succeed())

			// Reset flags for each test
			opts.OpenGhPR = false
			opts.OpenGhIssue = false
			opts.UseGhModels = false
		})

		AfterEach(func() {
			_ = os.RemoveAll(tmpBinDir)
			_ = os.Setenv("PATH", oldPathEnv)
		})

		It("Should succeed when both --open-gh-pr and --open-gh-issue are set", func() {
			opts.OpenGhPR = true
			opts.OpenGhIssue = true
			err := opts.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail when --use-gh-models is set without --open-gh-pr", func() {
			opts.UseGhModels = true
			opts.OpenGhPR = false
			err := opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("requires --open-gh-pr"))
		})

		It("Should succeed when --use-gh-models is set with --open-gh-pr", func() {
			opts.UseGhModels = true
			opts.OpenGhPR = true
			err := opts.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Validate branch name for --open-gh-pr (security)", func() {
		It("Should fail when branch name does not start with 'kubebuilder-update-from-'", func() {
			opts.OutputBranch = "my-custom-branch"
			opts.OpenGhPR = true

			// This validation happens in Validate() before openGitHubPR is called
			err := opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("security"))
			Expect(err.Error()).To(ContainSubstring("kubebuilder-update-from-"))
		})

		It("Should use default branch name with correct prefix", func() {
			opts.FromVersion = "v1.0.0"
			opts.ToVersion = "v2.0.0"
			opts.OutputBranch = ""
			opts.OpenGhPR = true

			// Default branch name should follow the pattern
			branchName := opts.getOutputBranchName()
			Expect(branchName).To(HavePrefix("kubebuilder-update-from-"))
		})
	})
})
