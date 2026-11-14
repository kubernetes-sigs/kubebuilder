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

package charttemplates

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("HelmLabels", func() {
	It("generates the valueOrDefault helper, named after the project", func() {
		labels := &HelmLabels{
			ProjectNameMixin: machinery.ProjectNameMixin{ProjectName: "my-chart"},
		}

		Expect(labels.SetTemplateDefaults()).To(Succeed())

		Expect(labels.Path).To(Equal(filepath.Join("dist", "chart", "templates", "_labels.tpl")))
		Expect(labels.TemplateBody).To(ContainSubstring(`{{- define "my-chart.valueOrDefault" -}}`))
		Expect(labels.TemplateBody).To(ContainSubstring(`{{- define "my-chart.stringMap" -}}`))
		Expect(labels.TemplateBody).To(ContainSubstring(`{{- if kindIs "invalid" $value }}`))
		Expect(labels.TemplateBody).To(ContainSubstring(`{{- index . 2 }}`))
		Expect(labels.TemplateBody).To(ContainSubstring(
			`{{- kindIs "string" $value | ternary $value (toJson $value) | toJson }}`))
	})

	It("always regenerates, so the helper matches the templates that call it", func() {
		labels := &HelmLabels{}

		Expect(labels.SetTemplateDefaults()).To(Succeed())

		Expect(labels.IfExistsAction).To(Equal(machinery.OverwriteFile))
	})
})
