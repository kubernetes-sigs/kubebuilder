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

var _ = Describe("InitCmd", func() {
	var (
		request  *external.PluginRequest
		response external.PluginResponse
	)

	BeforeEach(func() {
		request = &external.PluginRequest{
			APIVersion: "v1alpha1",
			Args:       []string{},
			Command:    "init",
			Universe:   map[string]string{},
			Config: map[string]interface{}{
				"projectName": "testproject",
			},
		}
	})

	Context("when scaffolding during init", func() {
		It("should generate Prometheus instance manifest", func() {
			response = InitCmd(request)

			Expect(response.Error).To(BeFalse())
			Expect(response.ErrorMsgs).To(BeEmpty())
			Expect(response.Universe).To(HaveKey("config/prometheus/prometheus.yaml"))

			content := response.Universe["config/prometheus/prometheus.yaml"]
			Expect(content).To(ContainSubstring("kind: Prometheus"))
			Expect(content).To(ContainSubstring("apiVersion: monitoring.coreos.com/v1"))
		})

		It("should generate kustomization for Prometheus resources", func() {
			response = InitCmd(request)

			Expect(response.Error).To(BeFalse())
			Expect(response.Universe).To(HaveKey("config/prometheus/kustomization.yaml"))

			content := response.Universe["config/prometheus/kustomization.yaml"]
			Expect(content).To(ContainSubstring("resources:"))
			Expect(content).To(ContainSubstring("- prometheus.yaml"))
		})

		It("should generate setup instructions", func() {
			response = InitCmd(request)

			Expect(response.Error).To(BeFalse())
			Expect(response.Universe).To(HaveKey("config/default/kustomization_prometheus_patch.yaml"))

			content := response.Universe["config/default/kustomization_prometheus_patch.yaml"]
			Expect(content).To(ContainSubstring("Prometheus Setup Instructions"))
			Expect(content).To(ContainSubstring("Install Prometheus Operator"))
		})

		It("should use project name from config when available", func() {
			request.Config = map[string]interface{}{
				"projectName": "myoperator",
			}

			response = InitCmd(request)

			Expect(response.Error).To(BeFalse())
			content := response.Universe["config/prometheus/prometheus.yaml"]
			Expect(content).To(ContainSubstring("myoperator"))
		})

		It("should fail when config is nil (no project name)", func() {
			request.Config = nil

			response = InitCmd(request)

			Expect(response.Error).To(BeTrue())
			Expect(response.ErrorMsgs).To(ContainElement(ContainSubstring("project name not found")))
		})

		It("should fail when projectName is empty", func() {
			request.Config = map[string]interface{}{
				"projectName": "",
			}

			response = InitCmd(request)

			Expect(response.Error).To(BeTrue())
			Expect(response.ErrorMsgs).To(ContainElement(ContainSubstring("project name not found")))
		})
	})

	Context("when validating Prometheus instance manifest", func() {
		It("should configure serviceMonitorSelector", func() {
			response = InitCmd(request)

			content := response.Universe["config/prometheus/prometheus.yaml"]
			Expect(content).To(ContainSubstring("serviceMonitorSelector"))
			Expect(content).To(ContainSubstring("control-plane: controller-manager"))
		})

		It("should set scrape interval", func() {
			response = InitCmd(request)

			content := response.Universe["config/prometheus/prometheus.yaml"]
			Expect(content).To(ContainSubstring("scrapeInterval"))
		})

		It("should configure security context", func() {
			response = InitCmd(request)

			content := response.Universe["config/prometheus/prometheus.yaml"]
			Expect(content).To(ContainSubstring("securityContext"))
			Expect(content).To(ContainSubstring("runAsNonRoot: true"))
		})
	})
})
