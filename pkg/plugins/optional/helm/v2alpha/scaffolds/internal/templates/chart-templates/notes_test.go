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

package charttemplates

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("Notes", func() {
	Context("SetTemplateDefaults", func() {
		var notes *Notes

		BeforeEach(func() {
			notes = &Notes{
				OutputDir: "dist",
				Force:     true,
			}
			notes.InjectProjectName("test-project")
		})

		It("should set the correct path", func() {
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(notes.Path).To(Equal("dist/chart/templates/NOTES.txt"))
		})

		It("should use default output dir when not specified", func() {
			notes.OutputDir = ""
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(notes.Path).To(Equal("dist/chart/templates/NOTES.txt"))
		})

		It("should set OverwriteFile action when Force is true", func() {
			notes.Force = true
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(notes.IfExistsAction).To(Equal(machinery.OverwriteFile))
		})

		It("should set SkipFile action when Force is false", func() {
			notes.Force = false
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(notes.IfExistsAction).To(Equal(machinery.SkipFile))
		})

		It("should generate template with Helm template syntax", func() {
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(notes.TemplateBody).To(ContainSubstring("{{ .Chart.Name }}"))
			Expect(notes.TemplateBody).To(ContainSubstring("{{ .Release.Name }}"))
			Expect(notes.TemplateBody).To(ContainSubstring("{{ .Release.Namespace }}"))
		})

		It("should include basic installation info", func() {
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(notes.TemplateBody).To(ContainSubstring("Thank you for installing"))
			Expect(notes.TemplateBody).To(ContainSubstring("release is named"))
			Expect(notes.TemplateBody).To(ContainSubstring("controller and CRDs have been installed"))
		})

		It("should include kubectl commands for verification", func() {
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(notes.TemplateBody).To(ContainSubstring("kubectl get pods"))
			Expect(notes.TemplateBody).To(ContainSubstring("kubectl get customresourcedefinitions"))
		})

		It("should include helm status commands", func() {
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(notes.TemplateBody).To(ContainSubstring("helm status"))
			Expect(notes.TemplateBody).To(ContainSubstring("helm get all"))
		})

		It("should not contain line numbers or metadata", func() {
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			// Template should be clean without any Go code artifacts
			Expect(notes.TemplateBody).NotTo(ContainSubstring("LINE_NUMBER"))
			Expect(notes.TemplateBody).NotTo(MatchRegexp(`(?m)^\s*\d+\|`))
		})

		It("should use proper Helm template delimiters", func() {
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			// Check for balanced template delimiters
			openCount := strings.Count(notes.TemplateBody, "{{")
			closeCount := strings.Count(notes.TemplateBody, "}}")
			Expect(openCount).To(Equal(closeCount), "Template should have balanced {{ and }} delimiters")
		})

		It("should be concise and generic", func() {
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			// Should be simple and not overly verbose (reasonable limit for helpful content)
			Expect(len(notes.TemplateBody)).To(BeNumerically("<", 800), "NOTES.txt should be concise")
		})

		It("should generate valid Helm template syntax when processed", func() {
			err := notes.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())

			// The template body should use backtick-wrapped syntax for Helm templates
			Expect(notes.TemplateBody).To(ContainSubstring("{{`{{ .Chart.Name }}`}}"))
			Expect(notes.TemplateBody).To(ContainSubstring("{{`{{ .Release.Name }}`}}"))
			Expect(notes.TemplateBody).To(ContainSubstring("{{`{{ .Release.Namespace }}`}}"))
		})
	})
})
