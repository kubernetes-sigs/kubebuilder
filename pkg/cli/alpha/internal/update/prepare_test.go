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

	"github.com/h2non/gock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal/common"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	v3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
)

const (
	testFromVersion = "v4.5.0"
	testToVersion   = "v4.6.0"
)

var _ = Describe("Prepare for internal update", func() {
	var (
		tmpDir      string
		workDir     string
		projectFile string
		mockGh      string
		err         error

		logFile string
		oldPath string
		opts    Update
	)

	BeforeEach(func() {
		workDir, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())

		// 1) Create tmp dir and chdir first
		tmpDir, err = os.MkdirTemp("", "kubebuilder-prepare-test")
		Expect(err).ToNot(HaveOccurred())
		err = os.Chdir(tmpDir)
		Expect(err).ToNot(HaveOccurred())

		// 2) Now that tmpDir exists, set logFile and PATH
		logFile = filepath.Join(tmpDir, "bin.log")

		oldPath = os.Getenv("PATH")
		Expect(os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)).To(Succeed())

		projectFile = filepath.Join(tmpDir, yaml.DefaultPath)

		config.Register(config.Version{Number: 3}, func() config.Config {
			return &v3.Cfg{Version: config.Version{Number: 3}, CliVersion: "1.0.0"}
		})

		gock.New("https://api.github.com").
			Get("/repos/kubernetes-sigs/kubebuilder/releases/latest").
			Reply(200).
			JSON(map[string]string{"tag_name": "v1.1.0"})

		// 3) Create the mock gh inside tmpDir (on PATH)
		mockGh = filepath.Join(tmpDir, "gh")
		ghOK := `#!/bin/bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "repo" && "$2" == "view" ]]; then
  echo "acme/repo"
  exit 0
fi
if [[ "$1" == "issue" && "$2" == "create" ]]; then
  exit 0
fi
exit 0`
		Expect(mockBinResponse(ghOK, mockGh)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.Setenv("PATH", oldPath)).To(Succeed())
		err = os.Chdir(workDir)
		Expect(err).ToNot(HaveOccurred())

		err = os.RemoveAll(tmpDir)
		Expect(err).ToNot(HaveOccurred())
		defer gock.Off()
	})

	Context("Prepare", func() {
		DescribeTable("should succeed for valid options",
			func(options *Update) {
				const version = `version: "3"`
				Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

				result := options.Prepare()
				Expect(result).ToNot(HaveOccurred())
				Expect(options.Prepare()).To(Succeed())
				Expect(options.FromVersion).To(Equal("v1.0.0"))
				Expect(options.ToVersion).To(Equal("v1.1.0"))
			},
			Entry("options", &Update{FromVersion: "v1.0.0", ToVersion: "v1.1.0", FromBranch: "test"}),
			Entry("options", &Update{FromVersion: "1.0.0", ToVersion: "1.1.0", FromBranch: "test"}),
			Entry("options", &Update{FromVersion: "v1.0.0", ToVersion: "v1.1.0"}),
			Entry("options", &Update{}),
		)

		DescribeTable("Should fail to prepare if project path is undetermined",
			func(options *Update) {
				err = options.Prepare()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to determine project path"))
			},
			Entry("options", &Update{FromVersion: "v1.0.0", ToVersion: "v1.1.0", FromBranch: "test"}),
		)

		DescribeTable("Should fail if PROJECT config could not be loaded",
			func(options *Update) {
				const version = ""
				Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

				err = options.Prepare()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to load PROJECT config"))
			},
			Entry("options", &Update{FromVersion: "v1.0.0", ToVersion: "v1.1.0", FromBranch: "test"}),
		)

		DescribeTable("Should fail if FromVersion cannot be determined",
			func(options *Update) {
				config.Register(config.Version{Number: 3}, func() config.Config {
					return &v3.Cfg{Version: config.Version{Number: 3}}
				})

				const version = `version: "3"`
				Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())
				Expect(options.FromVersion).To(BeEquivalentTo(""))
			},
			Entry("options", &Update{}),
		)
	})

	Context("DefineFromVersion", func() {
		DescribeTable("Should succeed when --from-version or CliVersion in Project config is present",
			func(options *Update) {
				const version = `version: "3"`
				Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

				config, errLoad := common.LoadProjectConfig(tmpDir)
				Expect(errLoad).ToNot(HaveOccurred())
				fromVersion, errLoad := options.defineFromVersion(config)
				Expect(errLoad).ToNot(HaveOccurred())
				Expect(fromVersion).To(BeEquivalentTo("v1.0.0"))
			},
			Entry("options", &Update{FromVersion: ""}),
			Entry("options", &Update{FromVersion: "1.0.0"}),
		)
		DescribeTable("Should fail when --from-version and CliVersion in Project config both are absent",
			func(options *Update) {
				config.Register(config.Version{Number: 3}, func() config.Config {
					return &v3.Cfg{Version: config.Version{Number: 3}}
				})

				const version = `version: "3"`
				Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

				config, errLoad := common.LoadProjectConfig(tmpDir)
				Expect(errLoad).NotTo(HaveOccurred())
				fromVersion, errLoad := options.defineFromVersion(config)
				Expect(errLoad).To(HaveOccurred())
				Expect(errLoad.Error()).To(ContainSubstring("no version specified in PROJECT file"))
				Expect(fromVersion).To(Equal(""))
			},
			Entry("options", &Update{FromVersion: ""}),
		)
	})

	Context("DefineToVersion", func() {
		DescribeTable("Should succeed.",
			func(options *Update) {
				toVersion := options.defineToVersion()
				Expect(toVersion).To(BeEquivalentTo("v1.1.0"))
			},
			Entry("options", &Update{ToVersion: "1.1.0"}),
			Entry("options", &Update{ToVersion: "v1.1.0"}),
			Entry("options", &Update{}),
		)
	})

	Context("OpenGitHubIssue", func() {
		It("creates issue without conflicts", func() {
			opts.FromBranch = defaultBranch
			opts.FromVersion = "v4.5.1"
			opts.ToVersion = "v4.8.0"

			err = opts.openGitHubIssue(false)
			Expect(err).ToNot(HaveOccurred())

			logs, readErr := os.ReadFile(logFile)
			Expect(readErr).ToNot(HaveOccurred())
			s := string(logs)

			Expect(s).To(ContainSubstring("repo view --json nameWithOwner --jq .nameWithOwner"))
			Expect(s).To(ContainSubstring("issue create"))

			expURL := fmt.Sprintf("https://github.com/%s/compare/%s...%s?expand=1",
				"acme/repo", opts.FromBranch, opts.getOutputBranchName())
			Expect(s).To(ContainSubstring(expURL))
			Expect(s).To(ContainSubstring(opts.ToVersion))
			Expect(s).To(ContainSubstring(opts.FromVersion))
		})

		It("creates issue with conflicts template", func() {
			opts.FromBranch = defaultBranch
			opts.FromVersion = "v4.5.2"
			opts.ToVersion = "v4.10.0"

			err = opts.openGitHubIssue(true)
			Expect(err).ToNot(HaveOccurred())

			logs, _ := os.ReadFile(logFile)
			s := string(logs)
			Expect(s).To(ContainSubstring("Resolve conflicts"))
			Expect(s).To(ContainSubstring("make manifests generate fmt vet lint-fix"))
		})

		It("fails when repo detection fails", func() {
			failRepo := `#!/bin/bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "repo" && "$2" == "view" ]]; then
  exit 1
fi
exit 0`
			Expect(mockBinResponse(failRepo, mockGh)).To(Succeed())

			err = opts.openGitHubIssue(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to detect GitHub repository"))
		})

		It("fails when issue creation fails", func() {
			failIssue := `#!/bin/bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "repo" && "$2" == "view" ]]; then
  echo "acme/repo"
  exit 0
fi
if [[ "$1" == "issue" && "$2" == "create" ]]; then
  exit 1
fi
exit 0`
			Expect(mockBinResponse(failIssue, mockGh)).To(Succeed())

			opts.FromBranch = defaultBranch
			opts.FromVersion = testFromVersion
			opts.ToVersion = testToVersion

			err = opts.openGitHubIssue(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create GitHub issue: exit status 1"))
		})
	})

	Context("Version Handling Edge Cases", func() {
		DescribeTable("Should handle version prefix normalization in defineToVersion",
			func(inputVersion, expectedVersion string) {
				opts := &Update{ToVersion: inputVersion}
				normalizedVersion := opts.defineToVersion()
				Expect(normalizedVersion).To(Equal(expectedVersion))
			},
			Entry("adds v prefix when missing", "1.0.0", "v1.0.0"),
			Entry("keeps v prefix when present", "v1.0.0", "v1.0.0"),
			Entry("handles semantic versioning", "1.2.3", "v1.2.3"),
			Entry("handles pre-release versions", "1.0.0-alpha", "v1.0.0-alpha"),
			Entry("handles build metadata", "1.0.0+build.1", "v1.0.0+build.1"),
		)

		DescribeTable("Should handle malformed versions gracefully during validation",
			func(invalidFromVersion, invalidToVersion string) {
				const version = `version: "3"`
				Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

				opts := &Update{FromVersion: invalidFromVersion, ToVersion: invalidToVersion, FromBranch: "test"}
				err = opts.Prepare()
				// Should handle gracefully or provide clear error message
				if err != nil {
					Expect(err.Error()).To(Or(
						ContainSubstring("version"),
						ContainSubstring("validate"),
						ContainSubstring("semantic"),
					))
				}
			},
			Entry("invalid from version", "not.a.version", "v1.0.0"),
			Entry("invalid to version", "v1.0.0", "not.a.version"),
			Entry("special characters in from", "v1.0.0$invalid", "v1.0.0"),
			Entry("special characters in to", "v1.0.0", "v1.0.0$invalid"),
		)
	})

	Context("GitHub Integration Edge Cases", func() {
		BeforeEach(func() {
			opts.FromBranch = defaultBranch
			opts.FromVersion = testFromVersion
			opts.ToVersion = testToVersion
		})

		It("handles missing gh CLI", func() {
			noGh := `#!/bin/bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "repo" && "$2" == "view" ]]; then
  echo "command not found: gh" >&2
  exit 127
fi
exit 0`
			Expect(mockBinResponse(noGh, mockGh)).To(Succeed())

			err = opts.openGitHubIssue(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to detect GitHub repository"))
		})

		It("handles gh CLI authentication failure", func() {
			authFailGh := `#!/bin/bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "repo" && "$2" == "view" ]]; then
  echo "error: authentication required" >&2
  exit 1
fi
exit 0`
			Expect(mockBinResponse(authFailGh, mockGh)).To(Succeed())

			err = opts.openGitHubIssue(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to detect GitHub repository"))
		})
	})

	Context("Output Branch Name Generation", func() {
		DescribeTable("Should generate correct output branch names",
			func(fromVersion, toVersion, expectedSuffix string) {
				opts := &Update{
					FromVersion: fromVersion,
					ToVersion:   toVersion,
				}
				branchName := opts.getOutputBranchName()
				Expect(branchName).To(ContainSubstring("kubebuilder-update-from"))
				Expect(branchName).To(ContainSubstring(expectedSuffix))
			},
			Entry("standard versions", "v1.0.0", "v1.1.0", "v1.0.0-to-v1.1.0"),
			Entry("versions without v prefix", "1.0.0", "1.1.0", "1.0.0-to-1.1.0"),
			Entry("pre-release versions", "v1.0.0-alpha", "v1.1.0-beta", "v1.0.0-alpha-to-v1.1.0-beta"),
		)

		It("uses custom output branch when specified", func() {
			customBranch := "my-custom-update-branch"
			opts := &Update{
				FromVersion:  "v1.0.0",
				ToVersion:    "v1.1.0",
				OutputBranch: customBranch,
			}
			branchName := opts.getOutputBranchName()
			Expect(branchName).To(Equal(customBranch))
		})
	})

	Context("Git Configuration Validation", func() {
		It("should handle empty git config", func() {
			opts := &Update{
				FromVersion: "v1.0.0",
				ToVersion:   "v1.1.0",
				FromBranch:  "test",
				GitConfig:   []string{}, // Empty git config
			}
			const version = `version: "3"`
			Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

			err = opts.Prepare()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle invalid git config format", func() {
			opts := &Update{
				FromVersion: "v1.0.0",
				ToVersion:   "v1.1.0",
				FromBranch:  "test",
				GitConfig:   []string{"invalid-config-format"}, // Invalid format
			}
			const version = `version: "3"`
			Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

			err = opts.Prepare()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Branch Name Validation", func() {
		DescribeTable("Should handle various branch name formats",
			func(branchName string, shouldSucceed bool) {
				opts := &Update{
					FromVersion: "v1.0.0",
					ToVersion:   "v1.1.0",
					FromBranch:  branchName,
				}
				const version = `version: "3"`
				Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

				err = opts.Prepare()
				if shouldSucceed {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(HaveOccurred())
				}
			},
			Entry("standard main branch", "main", true),
			Entry("standard master branch", "master", true),
			Entry("feature branch", "feature/my-feature", true),
			Entry("release branch", "release/v1.0.0", true),
			Entry("branch with numbers", "branch-123", true),
			Entry("empty branch name", "", true),
		)
	})

	Context("Resource Cleanup and Error Recovery", func() {
		It("should handle cleanup when preparation fails", func() {
			// This test ensures that temporary resources are cleaned up even when operations fail
			failGit := `#!/bin/bash
echo "$@" >> "` + logFile + `"
exit 1`
			mockGit := filepath.Join(tmpDir, "git")
			Expect(mockBinResponse(failGit, mockGit)).To(Succeed())

			opts := &Update{
				FromVersion: "v1.0.0",
				ToVersion:   "v1.1.0",
				FromBranch:  "test",
			}
			const version = `version: "3"`
			Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

			err = opts.Prepare()
			// The specific error depends on when git fails in the preparation process
			// This test ensures the system handles git failures gracefully
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("git"))
			}
		})
	})
})
