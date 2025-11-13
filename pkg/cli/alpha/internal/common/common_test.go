//go:build integration
// +build integration

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

package common

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	v3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("LoadProjectConfig", func() {
	var (
		kbc         *utils.TestContext
		projectFile string
	)

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext("kubebuilder", "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
		projectFile = filepath.Join(kbc.Dir, yaml.DefaultPath)
	})

	AfterEach(func() {
		By("cleaning up test artifacts")
		kbc.Destroy()
	})

	Context("when PROJECT file exists and is valid", func() {
		It("should load the project config successfully", func() {
			config.Register(config.Version{Number: 3}, func() config.Config {
				return &v3.Cfg{Version: config.Version{Number: 3}}
			})

			const version = `version: "3"
`
			Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

			cfg, err := LoadProjectConfig(kbc.Dir)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
		})
	})

	Context("when PROJECT file does not exist", func() {
		It("should return an error", func() {
			_, err := LoadProjectConfig(kbc.Dir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load PROJECT file"))
		})
	})

	Context("when PROJECT file is invalid", func() {
		It("should return an error", func() {
			Expect(os.WriteFile(projectFile, []byte(":?!"), 0o644)).To(Succeed())

			_, err := LoadProjectConfig(kbc.Dir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load PROJECT file"))
		})
	})
})

var _ = Describe("GetInputPath", func() {
	var (
		kbc         *utils.TestContext
		projectFile string
	)

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext("kubebuilder", "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
		projectFile = filepath.Join(kbc.Dir, yaml.DefaultPath)
	})

	AfterEach(func() {
		By("cleaning up test artifacts")
		kbc.Destroy()
	})

	Context("when inputPath has trailing slash", func() {
		It("should handle trailing slash and find PROJECT file", func() {
			Expect(os.WriteFile(projectFile, []byte("test"), 0o644)).To(Succeed())

			inputPath, err := GetInputPath(kbc.Dir + "/")
			Expect(err).NotTo(HaveOccurred())
			Expect(inputPath).To(Equal(kbc.Dir + "/"))
		})
	})

	Context("when inputPath is empty", func() {
		It("should return error if PROJECT file does not exist in CWD", func() {
			inputPath, err := GetInputPath("")
			Expect(err).To(HaveOccurred())
			Expect(inputPath).To(Equal(""))
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	Context("when inputPath is valid and PROJECT file exists", func() {
		It("should return the inputPath", func() {
			Expect(os.WriteFile(projectFile, []byte("test"), 0o644)).To(Succeed())

			inputPath, err := GetInputPath(kbc.Dir)
			Expect(err).NotTo(HaveOccurred())
			Expect(inputPath).To(Equal(kbc.Dir))
		})
	})

	Context("when inputPath is valid but PROJECT file does not exist", func() {
		It("should return an error", func() {
			inputPath, err := GetInputPath(kbc.Dir)
			Expect(err).To(HaveOccurred())
			Expect(inputPath).To(Equal(""))
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	Context("when inputPath does not exist", func() {
		It("should return an error", func() {
			invalidPath := filepath.Join(kbc.Dir, "nonexistent")
			inputPath, err := GetInputPath(invalidPath)
			Expect(err).To(HaveOccurred())
			Expect(inputPath).To(Equal(""))
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})
})
