/*
Copyright 2022 The Kubernetes Authors.

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

package scaffolds

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

var _ = Describe("EditCmd", func() {
	var (
		request  *external.PluginRequest
		response external.PluginResponse
	)

	BeforeEach(func() {
		request = &external.PluginRequest{
			APIVersion: "v1alpha1",
			Args:       []string{},
			Command:    "edit",
			Universe:   map[string]string{},
			Config: map[string]interface{}{
				"projectName": "testproject",
			},
		}
	})

	Context("when adding Prometheus to existing project", func() {
		It("should generate all required files", func() {
			response = EditCmd(request)

			Expect(response.Error).To(BeFalse())
			Expect(response.ErrorMsgs).To(BeEmpty())

			// Verify all expected files are present
			Expect(response.Universe).To(HaveKey("config/prometheus/prometheus.yaml"))
			Expect(response.Universe).To(HaveKey("config/prometheus/kustomization.yaml"))
			Expect(response.Universe).To(HaveKey("config/default/kustomization_prometheus_patch.yaml"))
		})

		It("should generate Prometheus instance manifest", func() {
			response = EditCmd(request)

			Expect(response.Error).To(BeFalse())
			content := response.Universe["config/prometheus/prometheus.yaml"]
			Expect(content).To(ContainSubstring("kind: Prometheus"))
			Expect(content).To(ContainSubstring("apiVersion: monitoring.coreos.com/v1"))
		})

		It("should use project name from config", func() {
			response = EditCmd(request)

			Expect(response.Error).To(BeFalse())
			content := response.Universe["config/prometheus/prometheus.yaml"]
			Expect(content).To(ContainSubstring("testproject"))
		})

		It("should fail when config is nil (PROJECT file not found)", func() {
			request.Config = nil

			response = EditCmd(request)

			Expect(response.Error).To(BeTrue())
			Expect(response.ErrorMsgs).To(ContainElement(ContainSubstring("failed to read project name")))
		})

		It("should fail when projectName is empty", func() {
			request.Config = map[string]interface{}{
				"projectName": "",
			}

			response = EditCmd(request)

			Expect(response.Error).To(BeTrue())
			Expect(response.ErrorMsgs).To(ContainElement(ContainSubstring("failed to read project name")))
		})
	})

	Context("when generating manifests", func() {
		It("should include installation instructions", func() {
			response = EditCmd(request)

			content := response.Universe["config/default/kustomization_prometheus_patch.yaml"]
			Expect(content).To(ContainSubstring("Prometheus Setup Instructions"))
			Expect(content).To(ContainSubstring("prometheus-operator"))
		})

		It("should reference prometheus.yaml in kustomization", func() {
			response = EditCmd(request)

			content := response.Universe["config/prometheus/kustomization.yaml"]
			Expect(content).To(ContainSubstring("- prometheus.yaml"))
		})
	})
})
