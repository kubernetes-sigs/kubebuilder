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
			cfg := cfgv3.New()
			_ = cfg.SetRepository("github.com/test/repo")
			_ = cfg.SetDomain("test.io")

			scaffolder := scaffolds.NewEditScaffolder(cfg, false, false, false, "", "", customLicensePath)
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
			cfg := cfgv3.New()
			_ = cfg.SetRepository("github.com/test/repo")
			_ = cfg.SetDomain("test.io")

			scaffolder := scaffolds.NewEditScaffolder(cfg, false, false, false, "apache2", "New Owner", "")
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
			cfg := cfgv3.New()
			_ = cfg.SetRepository("github.com/test/repo")
			_ = cfg.SetDomain("test.io")

			scaffolder := scaffolds.NewEditScaffolder(cfg, false, false, false, "", "", "")
			scaffolder.InjectFS(fs)
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify boilerplate file was not created
			boilerplatePath := filepath.Join(tmpDir, "hack", "boilerplate.go.txt")
			_, err = os.Stat(boilerplatePath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})
