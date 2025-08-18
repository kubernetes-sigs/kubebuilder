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
			opts.FromVersion = "v4.5.0"
			opts.ToVersion = "v4.6.0"

			err = opts.openGitHubIssue(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create GitHub Issue"))
		})
	})
})
