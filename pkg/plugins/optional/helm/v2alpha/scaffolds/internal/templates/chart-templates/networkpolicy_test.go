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
	"github.com/spf13/afero"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

const helmChartOutputDir = "dist"

var _ = Describe("NetworkPolicy", func() {
	Context("SetTemplateDefaults", func() {
		var networkPolicy *NetworkPolicy

		BeforeEach(func() {
			networkPolicy = &NetworkPolicy{
				OutputDir: helmChartOutputDir,
				Force:     true,
			}
			networkPolicy.InjectProjectName("test-project")
		})

		It("should set the correct path", func() {
			err := networkPolicy.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(networkPolicy.Path).To(Equal("dist/chart/templates/network-policy/allow-metrics-traffic.yaml"))
		})

		It("should use default output dir when not specified", func() {
			networkPolicy.OutputDir = ""
			err := networkPolicy.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(networkPolicy.Path).To(Equal("dist/chart/templates/network-policy/allow-metrics-traffic.yaml"))
		})

		It("should set OverwriteFile action when Force is true", func() {
			err := networkPolicy.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(networkPolicy.IfExistsAction).To(Equal(machinery.OverwriteFile))
		})

		It("should set SkipFile action when Force is false", func() {
			networkPolicy.Force = false
			err := networkPolicy.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
			Expect(networkPolicy.IfExistsAction).To(Equal(machinery.SkipFile))
		})

		It("should generate a metrics NetworkPolicy guarded by networkPolicy.enabled", func() {
			err := networkPolicy.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())

			Expect(networkPolicy.TemplateBody).To(ContainSubstring("{{`{{- if .Values.networkPolicy.enabled }}`}}"))
			Expect(networkPolicy.TemplateBody).To(ContainSubstring("kind: NetworkPolicy"))
			Expect(networkPolicy.TemplateBody).To(ContainSubstring(
				`name: {{ "{{ include \"test-project.resourceName\" ` +
					`(dict \"suffix\" \"allow-metrics-traffic\" \"context\" $) }}" }}`))
			Expect(networkPolicy.TemplateBody).To(ContainSubstring("metrics: enabled"))
			Expect(networkPolicy.TemplateBody).To(ContainSubstring(`port: {{ "{{ .Values.metrics.port }}" }}`))
			Expect(networkPolicy.TemplateBody).To(ContainSubstring(`{{ "{{ include \"test-project.name\" . }}" }}`))
			Expect(networkPolicy.TemplateBody).NotTo(ContainSubstring("allow-webhook-traffic"))
		})

		It("should add a webhook NetworkPolicy when webhooks are included", func() {
			networkPolicy.Webhook = true
			err := networkPolicy.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())

			Expect(networkPolicy.Path).To(Equal("dist/chart/templates/network-policy/allow-webhook-traffic.yaml"))
			Expect(networkPolicy.TemplateBody).To(ContainSubstring(
				"{{`{{- if and .Values.networkPolicy.enabled .Values.webhook.enabled }}`}}"))
			Expect(networkPolicy.TemplateBody).To(ContainSubstring(
				`name: {{ "{{ include \"test-project.resourceName\" ` +
					`(dict \"suffix\" \"allow-webhook-traffic\" \"context\" $) }}" }}`))
			Expect(networkPolicy.TemplateBody).To(ContainSubstring("webhook: enabled"))
			Expect(networkPolicy.TemplateBody).To(ContainSubstring(`port: {{ "{{ .Values.webhook.port }}" }}`))
			Expect(networkPolicy.TemplateBody).NotTo(ContainSubstring("allow-metrics-traffic"))
		})

		It("should render Helm template syntax through machinery", func() {
			cfg := cfgv3.New()
			Expect(cfg.SetProjectName("test-project")).To(Succeed())

			fs := afero.NewMemMapFs()
			scaffold := machinery.NewScaffold(machinery.Filesystem{FS: fs}, machinery.WithConfig(cfg))
			err := scaffold.Execute(&NetworkPolicy{
				OutputDir: helmChartOutputDir,
			}, &NetworkPolicy{
				Webhook:   true,
				OutputDir: helmChartOutputDir,
			})
			Expect(err).NotTo(HaveOccurred())

			content, err := afero.ReadFile(fs, "dist/chart/templates/network-policy/allow-metrics-traffic.yaml")
			Expect(err).NotTo(HaveOccurred())
			metricsPolicy := string(content)

			content, err = afero.ReadFile(fs, "dist/chart/templates/network-policy/allow-webhook-traffic.yaml")
			Expect(err).NotTo(HaveOccurred())
			webhookPolicy := string(content)

			Expect(metricsPolicy).To(ContainSubstring("{{- if .Values.networkPolicy.enabled }}"))
			Expect(metricsPolicy).To(ContainSubstring(`{{ include "test-project.name" . }}`))
			Expect(metricsPolicy).To(ContainSubstring(
				`name: {{ include "test-project.resourceName" (dict "suffix" "allow-metrics-traffic" "context" $) }}`))
			Expect(metricsPolicy).To(ContainSubstring("port: {{ .Values.metrics.port }}"))

			Expect(webhookPolicy).To(ContainSubstring(
				"{{- if and .Values.networkPolicy.enabled .Values.webhook.enabled }}"))
			Expect(webhookPolicy).To(ContainSubstring(`{{ include "test-project.name" . }}`))
			Expect(webhookPolicy).To(ContainSubstring(
				`name: {{ include "test-project.resourceName" (dict "suffix" "allow-webhook-traffic" "context" $) }}`))
			Expect(webhookPolicy).To(ContainSubstring("port: {{ .Values.webhook.port }}"))
		})
	})
})
