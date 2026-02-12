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

var _ = Describe("initSubcommand", func() {
	var (
		subCmd *initSubcommand
		cfg    config.Config
	)

	BeforeEach(func() {
		subCmd = &initSubcommand{}
		cfg = cfgv3.New()
	})

	Context("InjectConfig", func() {
		It("should set repository when provided", func() {
			subCmd.repo = "github.com/example/test"
			err := subCmd.InjectConfig(cfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.GetRepository()).To(Equal("github.com/example/test"))
		})

		It("should fail when repository cannot be detected", func() {
			originalDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Chdir(originalDir) }()

			tmpDir, err := os.MkdirTemp("", "test-init")
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.RemoveAll(tmpDir) }()

			err = os.Chdir(tmpDir)
			Expect(err).NotTo(HaveOccurred())

			subCmd.repo = ""
			err = subCmd.InjectConfig(cfg)

			Expect(err).To(HaveOccurred())
		})
	})

	Context("checkDir validation", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "test-checkdir")
			Expect(err).NotTo(HaveOccurred())

			originalDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = os.Chdir(originalDir)
				_ = os.RemoveAll(tmpDir)
			})

			err = os.Chdir(tmpDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should pass for empty directory", func() {
			Expect(checkDir()).To(Succeed())
		})

		It("should pass when only go.mod exists", func() {
			err := os.WriteFile("go.mod", []byte("module test"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			Expect(checkDir()).To(Succeed())
		})

		It("should fail when PROJECT already exists", func() {
			err := os.WriteFile("PROJECT", []byte("version: 3"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = checkDir()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already initialized"))
		})

		It("should fail when Makefile exists", func() {
			err := os.WriteFile("Makefile", []byte("all:"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = checkDir()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already initialized"))
		})

		It("should fail when cmd/main.go exists", func() {
			err := os.MkdirAll("cmd", 0o755)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(filepath.Join("cmd", "main.go"), []byte("package main"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = checkDir()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already initialized"))
		})
	})

	Context("Boilerplate customization", func() {
		var (
			fs     machinery.Filesystem
			tmpDir string
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "test-boilerplate")
			Expect(err).NotTo(HaveOccurred())

			fs = machinery.Filesystem{
				FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
			}

			DeferCleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})
		})

		It("should use custom license file when provided", func() {
			// Create a custom license header file
			customLicensePath := filepath.Join(tmpDir, "custom-header.txt")
			customContent := `/*
Copyright 2026 Test Company.

Custom License Header.
*/`
			err := os.WriteFile(customLicensePath, []byte(customContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			// Create scaffolder with custom license file
			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(testCfg, "apache2", "Test Owner", customLicensePath, "kubebuilder")
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify boilerplate file was created with custom content
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			content, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(customContent))
		})

		It("should use default license when no custom license file is provided", func() {
			// Create scaffolder without custom license file
			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(testCfg, "apache2", "Test Owner", "", "kubebuilder")
			scaffolder.InjectFS(fs)
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify boilerplate file was created with Apache 2.0 license
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			content, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Apache License"))
			Expect(string(content)).To(ContainSubstring("Test Owner"))
		})

		It("should handle none license option", func() {
			// Create scaffolder with none license
			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(testCfg, "none", "", "", "kubebuilder")
			scaffolder.InjectFS(fs)
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify boilerplate file was not created
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			_, err = os.Stat(boilerplatePath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("should fail when custom license file does not exist", func() {
			// Create scaffolder with non-existent license file
			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(testCfg, "apache2", "Test Owner", "/nonexistent/file.txt", "kubebuilder")
			scaffolder.InjectFS(fs)
			err := scaffolder.Scaffold()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read license file"))
		})

		It("should preserve exact content from license file with no modifications", func() {
			// Create a custom license file with specific formatting
			customLicensePath := filepath.Join(tmpDir, "exact-header.txt")
			// Include various formatting: multiple spaces, newlines, special characters
			exactContent := `/*
Copyright 2026 Test Company.
  Indented line here.

Double newline above.
Special chars: @#$%
*/`
			err := os.WriteFile(customLicensePath, []byte(exactContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			// Create scaffolder with custom license file
			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(testCfg, "apache2", "Test Owner", customLicensePath, "kubebuilder")
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify content is EXACTLY the same (byte-for-byte)
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			actualContent, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actualContent)).To(Equal(exactContent), "Content should be preserved exactly with no modifications")
		})

		It("should handle license file with no trailing newline", func() {
			// Create a license file without trailing newline
			customLicensePath := filepath.Join(tmpDir, "no-newline.txt")
			contentNoNewline := "/*\nCopyright 2026.\n*/"
			err := os.WriteFile(customLicensePath, []byte(contentNoNewline), 0o644)
			Expect(err).NotTo(HaveOccurred())

			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(testCfg, "apache2", "", customLicensePath, "kubebuilder")
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify exact content preservation
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			actualContent, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actualContent)).To(Equal(contentNoNewline))
		})

		It("should create boilerplate when --license-file is provided even with --license none", func() {
			// Create a custom license header file
			customLicensePath := filepath.Join(tmpDir, "custom-header.txt")
			customContent := `/*
Copyright 2026 My Company.

My Custom License.
*/`
			err := os.WriteFile(customLicensePath, []byte(customContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			// Create scaffolder with --license none BUT --license-file provided
			// Expected: --license-file should override --license none
			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(testCfg, "none", "", customLicensePath, "kubebuilder")
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify boilerplate file WAS created with custom content (--license-file overrides --license none)
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			content, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(customContent))
		})

		It("should handle empty license file", func() {
			// Create an empty license file
			customLicensePath := filepath.Join(tmpDir, "empty-header.txt")
			err := os.WriteFile(customLicensePath, []byte(""), 0o644)
			Expect(err).NotTo(HaveOccurred())

			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(testCfg, "apache2", "Test Owner", customLicensePath, "kubebuilder")
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
			// Create a custom license file without owner placeholder
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
			scaffolder := scaffolds.NewInitScaffolder(testCfg, "apache2", "Ignored Owner", customLicensePath, "kubebuilder")
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify content matches custom file exactly (owner flag was ignored)
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			content, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(customContent), "Owner flag should be ignored when license-file is provided")
			Expect(string(content)).NotTo(ContainSubstring("Ignored Owner"), "Owner from flag should not appear in boilerplate")
		})

		It("should handle license file with only whitespace content", func() {
			// Create a license file with only whitespace
			customLicensePath := filepath.Join(tmpDir, "whitespace-content.txt")
			whitespaceContent := "   \n\t\n   "
			err := os.WriteFile(customLicensePath, []byte(whitespaceContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			testCfg := cfgv3.New()
			_ = testCfg.SetRepository("github.com/test/repo")
			_ = testCfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(testCfg, "apache2", "Test Owner", customLicensePath, "kubebuilder")
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify content is preserved exactly (whitespace not trimmed)
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			content, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(whitespaceContent), "Whitespace content should be preserved as-is")
		})
	})

	Context("PreScaffold validation", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "test-prescaffold")
			Expect(err).NotTo(HaveOccurred())

			originalDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = os.Chdir(originalDir)
				_ = os.RemoveAll(tmpDir)
			})

			err = os.Chdir(tmpDir)
			Expect(err).NotTo(HaveOccurred())

			// Initialize subcommand with config
			subCmd = &initSubcommand{
				license:            "apache2",
				owner:              "Test Owner",
				repo:               "github.com/test/repo",
				skipGoVersionCheck: true,
			}
			cfg = cfgv3.New()
			err = subCmd.InjectConfig(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail early when license file does not exist", func() {
			subCmd.licenseFile = "/nonexistent/license.txt"

			// PreScaffold should fail before any files are created
			err := subCmd.PreScaffold(machinery.Filesystem{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("license file"))
			Expect(err.Error()).To(ContainSubstring("does not exist"))

			// Verify NO files were created (project not in broken state)
			entries, err := os.ReadDir(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(BeEmpty(), "No files should be created when PreScaffold fails")
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

			// PreScaffold should succeed (using built-in license)
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
			// Verify the error message shows the trimmed path (not the original with spaces)
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
			err := os.WriteFile(invalidLicensePath, []byte("/*\nCopyright 2026 Test Company\n*/\nExtra text after"), 0o644)
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
