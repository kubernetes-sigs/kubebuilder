/*
Copyright 2026 The Kubernetes Authors.

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

package v4

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds"
)

var _ = Describe("editSubcommand", func() {
	var (
		subCmd *editSubcommand
		cfg    config.Config
	)

	BeforeEach(func() {
		subCmd = &editSubcommand{}
		cfg = cfgv3.New()
	})

	It("should inject config successfully", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		Expect(subCmd.config).To(Equal(cfg))
	})

	Context("Boilerplate update", func() {
		var (
			fs     machinery.Filesystem
			tmpDir string
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "test-edit-boilerplate")
			Expect(err).NotTo(HaveOccurred())

			fs = machinery.Filesystem{
				FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
			}

			// Create necessary files for edit to work
			err = os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM golang:1.23"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})
		})

		It("should update boilerplate with custom license file", func() {
			// Create a custom license header file
			customLicensePath := filepath.Join(tmpDir, "new-header.txt")
			customContent := `/*
Copyright 2026 New Company.

Updated License Header.
*/`
			err := os.WriteFile(customLicensePath, []byte(customContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			// Create scaffolder with custom license file
			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewEditScaffolder(testCfg, false, false, false, "", "", customLicensePath)
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify boilerplate file was created/updated with custom content
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			content, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(customContent))
		})

		It("should update boilerplate with license flag", func() {
			// Create scaffolder with license flag
			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewEditScaffolder(testCfg, false, false, false, "apache2", "New Owner", "")
			scaffolder.InjectFS(fs)
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify boilerplate file was created with Apache 2.0 license
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			content, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Apache License"))
			Expect(string(content)).To(ContainSubstring("New Owner"))
		})

		It("should not update boilerplate when neither license nor license-file is set", func() {
			// Create scaffolder without license or license file
			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewEditScaffolder(testCfg, false, false, false, "", "", "")
			scaffolder.InjectFS(fs)
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify boilerplate file was not created
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			_, err = os.Stat(boilerplatePath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("should handle empty license file", func() {
			// Create an empty license file
			customLicensePath := filepath.Join(tmpDir, "empty-header.txt")
			err := os.WriteFile(customLicensePath, []byte(""), 0o644)
			Expect(err).NotTo(HaveOccurred())

			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewEditScaffolder(testCfg, false, false, false, "apache2", "Test Owner", customLicensePath)
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify boilerplate file was created (even if empty)
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			content, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(""), "Empty license file should create empty boilerplate")
		})

		It("should ignore owner flag when using license-file", func() {
			// Create a custom license file
			customLicensePath := filepath.Join(tmpDir, "custom-no-owner.txt")
			customContent := `/*
Copyright 2026.

Fixed License Header.
*/`
			err := os.WriteFile(customLicensePath, []byte(customContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			// Pass owner flag - it should be ignored when license-file is provided
			scaffolder := scaffolds.NewEditScaffolder(
				testCfg, false, false, false, "apache2", "Ignored Owner", customLicensePath)
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify content matches custom file exactly (owner flag was ignored)
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			content, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(customContent), "Owner flag should be ignored when license-file is provided")
			Expect(string(content)).NotTo(ContainSubstring("Ignored Owner"))
		})
	})

	Context("PreScaffold validation", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "test-edit-prescaffold")
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})

			// Initialize subcommand with config
			subCmd = &editSubcommand{
				license: "apache2",
				owner:   "Test Owner",
			}
			cfg = cfgv3.New()
			err = subCmd.InjectConfig(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail early when license file does not exist", func() {
			subCmd.licenseFile = "/nonexistent/license.txt"

			// PreScaffold should fail before any updates
			err := subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("license file"))
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})

		It("should succeed when license file exists", func() {
			customLicensePath := filepath.Join(tmpDir, "valid-license.txt")
			err := os.WriteFile(customLicensePath, []byte("/*\nValid License\n*/"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			subCmd.licenseFile = customLicensePath

			// PreScaffold should succeed
			err = subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should succeed when no license file is specified", func() {
			subCmd.licenseFile = ""

			// PreScaffold should succeed
			err := subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should treat whitespace-only license file as empty", func() {
			subCmd.licenseFile = "   "

			// PreScaffold should succeed and trim the whitespace (treating as empty)
			err := subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).NotTo(HaveOccurred())
			Expect(subCmd.licenseFile).To(Equal(""), "Whitespace-only path should be trimmed to empty string")
		})

		It("should trim whitespace from license file path", func() {
			customLicensePath := filepath.Join(tmpDir, "valid-license.txt")
			err := os.WriteFile(customLicensePath, []byte("/*\nValid License\n*/"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			subCmd.licenseFile = "  " + customLicensePath + "  "

			// PreScaffold should succeed and trim the whitespace
			err = subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).NotTo(HaveOccurred())
			Expect(subCmd.licenseFile).To(Equal(customLicensePath), "Whitespace should be trimmed from path")
		})

		It("should fail when trimmed license file path does not exist", func() {
			// Path with whitespace that doesn't exist even after trimming
			subCmd.licenseFile = "  /nonexistent/file.txt  "

			// PreScaffold should fail with trimmed path
			err := subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("license file"))
			Expect(err.Error()).To(ContainSubstring("does not exist"))
			// Verify the error message shows the trimmed path
			Expect(err.Error()).To(ContainSubstring("/nonexistent/file.txt"))
			Expect(subCmd.licenseFile).To(Equal("/nonexistent/file.txt"), "Path should be trimmed before validation")
		})

		It("should fail when license file is missing Go comment delimiters", func() {
			// Create a license file without proper Go comment delimiters
			invalidLicensePath := filepath.Join(tmpDir, "invalid-no-delimiters.txt")
			err := os.WriteFile(invalidLicensePath, []byte("Copyright 2026 Test Company"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			subCmd.licenseFile = invalidLicensePath

			// PreScaffold should fail
			err = subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be a valid Go comment block"))
			Expect(err.Error()).To(ContainSubstring("start with /* and end with */"))
		})

		It("should fail when license file does not start with opening delimiter", func() {
			// Create a license file that doesn't start with /*
			invalidLicensePath := filepath.Join(tmpDir, "invalid-no-opening.txt")
			err := os.WriteFile(invalidLicensePath, []byte("Copyright 2026 Test Company\n/*\nSome text\n*/"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			subCmd.licenseFile = invalidLicensePath

			// PreScaffold should fail
			err = subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be a valid Go comment block"))
		})

		It("should fail when license file does not end with closing delimiter", func() {
			// Create a license file that doesn't end with */
			invalidLicensePath := filepath.Join(tmpDir, "invalid-no-closing.txt")
			err := os.WriteFile(invalidLicensePath, []byte("/*\nCopyright 2026 Test Company\n*/\nExtra text"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			subCmd.licenseFile = invalidLicensePath

			// PreScaffold should fail
			err = subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be a valid Go comment block"))
		})

		It("should succeed when license file has proper Go comment delimiters", func() {
			// Create a valid license file with proper delimiters
			validLicensePath := filepath.Join(tmpDir, "valid-with-delimiters.txt")
			err := os.WriteFile(validLicensePath, []byte("/*\nCopyright 2026 Test Company\n*/"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			subCmd.licenseFile = validLicensePath

			// PreScaffold should succeed
			err = subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow empty license file without requiring delimiters", func() {
			// Create an empty license file
			emptyLicensePath := filepath.Join(tmpDir, "empty-license.txt")
			err := os.WriteFile(emptyLicensePath, []byte(""), 0o644)
			Expect(err).NotTo(HaveOccurred())

			subCmd.licenseFile = emptyLicensePath

			// PreScaffold should succeed (empty files are allowed)
			err = subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
