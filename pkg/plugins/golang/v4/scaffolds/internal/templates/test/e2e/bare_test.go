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

package e2e

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &BareTest{}

// BareTest defines the basic e2e test for bare init projects (no APIs/controllers)
type BareTest struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults set defaults for this template
func (f *BareTest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("test", "e2e", "bare_test.go")
	}

	f.TemplateBody = bareTestTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const bareTestTemplate = `//go:build e2e
// +build e2e

{{ .Boilerplate }}

package e2e

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Describe bare init project tests that run before any APIs are added
var _ = Describe("Project Structure", func() {
	It("should have the required project files", func() {
		By("checking for PROJECT file")
		_, err := os.Stat("PROJECT")
		Expect(err).NotTo(HaveOccurred(), "PROJECT file should exist")

		By("checking for Makefile")
		_, err = os.Stat("Makefile")
		Expect(err).NotTo(HaveOccurred(), "Makefile should exist")

		By("checking for go.mod")
		_, err = os.Stat("go.mod")
		Expect(err).NotTo(HaveOccurred(), "go.mod should exist")

		By("checking for main.go")
		_, err = os.Stat(filepath.Join("cmd", "main.go"))
		Expect(err).NotTo(HaveOccurred(), "cmd/main.go should exist")

		By("checking for config directory")
		_, err = os.Stat("config")
		Expect(err).NotTo(HaveOccurred(), "config directory should exist")

		By("checking for test directory")
		_, err = os.Stat("test")
		Expect(err).NotTo(HaveOccurred(), "test directory should exist")

		By("checking for GitHub workflows")
		_, err = os.Stat(filepath.Join(".github", "workflows", "test.yml"))
		Expect(err).NotTo(HaveOccurred(), "test workflow should exist")

		_, err = os.Stat(filepath.Join(".github", "workflows", "lint.yml"))
		Expect(err).NotTo(HaveOccurred(), "lint workflow should exist")

		_, err = os.Stat(filepath.Join(".github", "workflows", "test-e2e.yml"))
		Expect(err).NotTo(HaveOccurred(), "e2e test workflow should exist")
	})
})
`