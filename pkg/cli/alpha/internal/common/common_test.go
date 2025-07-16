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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	v3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
)

func TestCommon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Common Pkg Suite")
}

var _ = Describe("LoadProjectConfig", func() {
	var (
		tmpDir      string
		projectFile string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "kubebuilder-common-test")
		Expect(err).NotTo(HaveOccurred())
		projectFile = filepath.Join(tmpDir, yaml.DefaultPath)
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when PROJECT file exists and is valid", func() {
		It("should load the project config successfully", func() {
			// Register version 3 config
			config.Register(config.Version{Number: 3}, func() config.Config {
				return &v3.Cfg{Version: config.Version{Number: 3}}
			})

			const version = `version: "3"
`
			Expect(os.WriteFile(projectFile, []byte(version), 0o644)).To(Succeed())

			config, err := LoadProjectConfig(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).NotTo(BeNil())
		})
	})

	Context("when PROJECT file does not exist", func() {
		It("should return an error", func() {
			_, err := LoadProjectConfig(tmpDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load PROJECT file"))
		})
	})

	Context("when PROJECT file is invalid", func() {
		It("should return an error", func() {
			// Write an invalid YAML content
			Expect(os.WriteFile(projectFile, []byte(":?!"), 0o644)).To(Succeed())

			_, err := LoadProjectConfig(tmpDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load PROJECT file"))
		})
	})
})

var _ = Describe("GetInputPath", func() {
	var (
		tmpDir      string
		projectFile string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "kubebuilder-common-test")
		Expect(err).NotTo(HaveOccurred())
		projectFile = filepath.Join(tmpDir, yaml.DefaultPath)
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when inputPath is empty", func() {
		It("should return current working directory if PROJECT file exists", func() {
			// Create PROJECT file in tmpDir
			Expect(os.WriteFile(projectFile, []byte("test-data"), 0o644)).To(Succeed())

			// Change working directory to tmpDir
			Expect(os.Chdir(tmpDir)).To(Succeed())

			currWd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())

			inputPath, err := GetInputPath("")
			Expect(err).NotTo(HaveOccurred())
			Expect(inputPath).To(Equal(currWd))
		})

		It("should return error if PROJECT file does not exist in cwd", func() {
			// Change working directory to tmpDir (no PROJECT file)
			Expect(os.Chdir(tmpDir)).To(Succeed())

			inputPath, err := GetInputPath("")
			Expect(err).To(HaveOccurred())
			Expect(inputPath).To(Equal(""))
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	Context("when inputPath is provided", func() {
		It("should return inputPath if PROJECT file exists", func() {
			Expect(os.WriteFile(projectFile, []byte("test"), 0o644)).To(Succeed())

			inputPath, err := GetInputPath(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(inputPath).To(Equal(tmpDir))
		})

		It("should return error if PROJECT file does not exist at provided inputPath", func() {
			inputPath, err := GetInputPath(tmpDir)
			Expect(err).To(HaveOccurred())
			Expect(inputPath).To(Equal(""))
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	Context("when inputPath is invalid", func() {
		It("should return error if inputPath does not exist", func() {
			invalidPath := filepath.Join(tmpDir, "nonexistent")
			inputPath, err := GetInputPath(invalidPath)
			Expect(err).To(HaveOccurred())
			Expect(inputPath).To(Equal(""))
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})
})
