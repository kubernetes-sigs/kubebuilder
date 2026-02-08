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
			cfg := cfgv3.New()
			_ = cfg.SetRepository("github.com/test/repo")
			_ = cfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(cfg, "apache2", "Test Owner", customLicensePath, "kubebuilder")
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
			cfg := cfgv3.New()
			_ = cfg.SetRepository("github.com/test/repo")
			_ = cfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(cfg, "apache2", "Test Owner", "", "kubebuilder")
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
			cfg := cfgv3.New()
			_ = cfg.SetRepository("github.com/test/repo")
			_ = cfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(cfg, "none", "", "", "kubebuilder")
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
			cfg := cfgv3.New()
			_ = cfg.SetRepository("github.com/test/repo")
			_ = cfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(cfg, "apache2", "Test Owner", "/nonexistent/file.txt", "kubebuilder")
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
			cfg := cfgv3.New()
			_ = cfg.SetRepository("github.com/test/repo")
			_ = cfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(cfg, "apache2", "Test Owner", customLicensePath, "kubebuilder")
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

			cfg := cfgv3.New()
			_ = cfg.SetRepository("github.com/test/repo")
			_ = cfg.SetDomain("test.io")

			scaffolder := scaffolds.NewInitScaffolder(cfg, "apache2", "", customLicensePath, "kubebuilder")
			scaffolder.InjectFS(fs)
			err = scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify exact content preservation
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			actualContent, err := os.ReadFile(boilerplatePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actualContent)).To(Equal(contentNoNewline))
		})
	})
})
