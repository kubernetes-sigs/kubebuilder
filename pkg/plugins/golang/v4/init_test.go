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

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
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
})
