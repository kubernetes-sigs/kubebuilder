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

package golang

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("golang:repository", func() {
	var (
		tmpDir string
		oldDir string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "repo-test")
		Expect(err).NotTo(HaveOccurred())
		oldDir, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(tmpDir)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.Chdir(oldDir)).To(Succeed())
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	When("go.mod exists", func() {
		BeforeEach(func() {
			// Simulate `go mod edit -json` output by writing a go.mod file and using go commands
			Expect(os.WriteFile("go.mod", []byte("module github.com/example/repo\n"), 0o644)).To(Succeed())
		})

		It("findGoModulePath returns the module path", func() {
			path, err := findGoModulePath()
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal("github.com/example/repo"))
		})

		It("FindCurrentRepo returns the module path", func() {
			path, err := FindCurrentRepo()
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal("github.com/example/repo"))
		})
	})

	When("go.mod does not exist", func() {
		It("findGoModulePath returns error", func() {
			got, err := findGoModulePath()
			Expect(err).To(HaveOccurred())
			Expect(got).To(Equal(""))
		})

		It("FindCurrentRepo tries to init a module and returns the path or a helpful error", func() {
			path, err := FindCurrentRepo()
			if err != nil {
				Expect(path).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("could not determine repository path"))
			} else {
				Expect(path).NotTo(BeEmpty())
			}
		})
	})

	When("go mod command fails with exec.ExitError", func() {
		var origPath string

		BeforeEach(func() {
			// Move go binary out of PATH to force exec error
			origPath = os.Getenv("PATH")
			// Set PATH to empty so "go" cannot be found
			Expect(os.Setenv("PATH", "")).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Setenv("PATH", origPath)).To(Succeed())
		})

		It("findGoModulePath returns error with stderr message", func() {
			got, err := findGoModulePath()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(BeEmpty())
			Expect(got).To(Equal(""))
		})

		It("FindCurrentRepo returns error with stderr message", func() {
			got, err := FindCurrentRepo()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not determine repository path"))
			Expect(got).To(Equal(""))
		})
	})
})
