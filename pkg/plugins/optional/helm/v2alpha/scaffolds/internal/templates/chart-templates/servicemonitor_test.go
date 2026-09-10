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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("ServiceMonitor", func() {
	Context("SetTemplateDefaults", func() {
		var serviceMonitor *ServiceMonitor

		BeforeEach(func() {
			serviceMonitor = &ServiceMonitor{
				OutputDir:   helmChartOutputDir,
				ServiceName: "test-project-controller-manager-metrics-service",
			}
			serviceMonitor.InjectProjectName("test-project")
		})

		It("preserves an existing file (SkipFile) when Force is false", func() {
			serviceMonitor.Force = false
			Expect(serviceMonitor.SetTemplateDefaults()).To(Succeed())
			Expect(serviceMonitor.IfExistsAction).To(Equal(machinery.SkipFile))
		})

		It("overwrites (OverwriteFile) when Force is true", func() {
			serviceMonitor.Force = true
			Expect(serviceMonitor.SetTemplateDefaults()).To(Succeed())
			Expect(serviceMonitor.IfExistsAction).To(Equal(machinery.OverwriteFile))
		})
	})
})
