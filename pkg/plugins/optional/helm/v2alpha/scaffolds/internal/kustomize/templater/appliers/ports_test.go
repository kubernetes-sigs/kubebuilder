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

package appliers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

func managerDeployment(name string) *unstructured.Unstructured {
	d := &unstructured.Unstructured{}
	d.SetAPIVersion("apps/v1")
	d.SetKind(common.KindDeployment)
	d.SetName(name)
	d.SetLabels(map[string]string{"control-plane": "controller-manager"})
	return d
}

const (
	managerPortFieldsYAML = `- --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
        - --webhook-port=9443
        ports:
        - containerPort: 8081
          name: health
        - containerPort: 9443
          name: webhook-server
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081`

	sidecarPortFieldsYAML = `- --metrics-bind-address=:9090
        - --health-probe-bind-address=:9091
        - --webhook-port=9092
        ports:
        - containerPort: 9091
          name: health
        - containerPort: 9092
          name: webhook-server
        livenessProbe:
          httpGet:
            path: /healthz
            port: 9091`
)

var _ = Describe("TemplatePorts manager Deployment sidecars", func() {
	const managerMetricsArg = "--metrics-bind-address=:8443"
	const sidecarMetricsArg = "--metrics-bind-address=:9090"

	deploymentWithSidecarBeforeManager := func() string {
		return `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  labels:
    control-plane: controller-manager
spec:
  template:
    spec:
      containers:
      - name: sidecar
        image: sidecar:latest
        args:
        ` + sidecarPortFieldsYAML + `
      - name: manager
        image: manager:latest
        args:
        ` + managerPortFieldsYAML
	}

	deploymentWithSidecarAfterManager := func() string {
		return `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  labels:
    control-plane: controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        image: manager:latest
        args:
        ` + managerPortFieldsYAML + `
      - name: sidecar
        image: sidecar:latest
        args:
        ` + sidecarPortFieldsYAML
	}

	assertSidecarPortFieldsUntouched := func(result string) {
		Expect(result).To(ContainSubstring(sidecarMetricsArg))
		Expect(result).To(ContainSubstring("--health-probe-bind-address=:9091"))
		Expect(result).To(ContainSubstring("--webhook-port=9092"))
		Expect(result).To(ContainSubstring("containerPort: 9091"))
		Expect(result).To(ContainSubstring("containerPort: 9092"))
		Expect(result).To(MatchRegexp(`path: /healthz\n\s+port: 9091`))
	}

	assertManagerPortFieldsTemplated := func(result string) {
		Expect(result).To(ContainSubstring("--metrics-bind-address=:{{ .Values.metrics.port }}"))
		Expect(result).To(ContainSubstring("--health-probe-bind-address=:{{ .Values.manager.healthProbe.port }}"))
		Expect(result).To(ContainSubstring("--webhook-port={{ .Values.webhook.port }}"))
		Expect(result).To(ContainSubstring("containerPort: {{ .Values.manager.healthProbe.port }}"))
		Expect(result).To(ContainSubstring("containerPort: {{ .Values.webhook.port }}"))
		Expect(result).NotTo(ContainSubstring(managerMetricsArg))
	}

	It("does not template port-like fields on a sidecar listed before the manager", func() {
		deployment := managerDeployment("test-project-controller-manager")
		result := TemplatePorts(deploymentWithSidecarBeforeManager(), deployment, "")

		assertSidecarPortFieldsUntouched(result)
		assertManagerPortFieldsTemplated(result)
	})

	It("does not template port-like fields on a sidecar listed after the manager", func() {
		deployment := managerDeployment("test-project-controller-manager")
		result := TemplatePorts(deploymentWithSidecarAfterManager(), deployment, "")

		assertSidecarPortFieldsUntouched(result)
		assertManagerPortFieldsTemplated(result)
	})

	It("does not template port fields on an extra deployment", func() {
		content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-worker
spec:
  template:
    spec:
      containers:
      - name: worker
        args:
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
        - --webhook-port=9443
`
		extra := &unstructured.Unstructured{}
		extra.SetAPIVersion("apps/v1")
		extra.SetKind(common.KindDeployment)
		extra.SetName("test-project-worker")

		result := TemplatePorts(content, extra, "")

		Expect(result).To(ContainSubstring("--metrics-bind-address=:8443"))
		Expect(result).To(ContainSubstring("--health-probe-bind-address=:8081"))
		Expect(result).To(ContainSubstring("--webhook-port=9443"))
		Expect(result).NotTo(ContainSubstring(".Values."))
	})
})

var _ = Describe("TemplatePorts NetworkPolicy", func() {
	const metricsPolicy = `spec:
  policyTypes:
    - Ingress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            metrics: enabled
      ports:
        - port: 8443
          protocol: TCP
`

	It("templates the metrics policy ingress port", func() {
		result := TemplatePorts(metricsPolicy, networkPolicy("test-project-allow-metrics-traffic"), "test-project")

		Expect(result).To(ContainSubstring("port: {{ .Values.metrics.port }}"))
		Expect(result).NotTo(ContainSubstring("port: 8443"))
	})

	It("templates the webhook policy ingress port", func() {
		webhookPolicy := `spec:
  policyTypes:
    - Ingress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            webhook: enabled
      ports:
        - port: 9443
          protocol: TCP
`
		result := TemplatePorts(webhookPolicy, networkPolicy("test-project-allow-webhook-traffic"), "test-project")

		Expect(result).To(ContainSubstring("port: {{ .Values.webhook.port }}"))
		Expect(result).NotTo(ContainSubstring("port: 9443"))
	})

	It("templates the webhook policy ingress port when the rule has no source selector", func() {
		webhookPolicy := `spec:
  policyTypes:
    - Ingress
  ingress:
    - ports:
        - port: 9443
          protocol: TCP
`
		result := TemplatePorts(webhookPolicy, networkPolicy("test-project-allow-webhook-traffic"), "test-project")

		Expect(result).To(ContainSubstring("port: {{ .Values.webhook.port }}"))
		Expect(result).NotTo(ContainSubstring("port: 9443"))
	})

	It("leaves the ports of a policy whose name only resembles the scaffolded ones untouched", func() {
		policy := `spec:
  ingress:
    - ports:
        - port: 8443
          protocol: TCP
`
		result := TemplatePorts(policy, networkPolicy("test-project-disallow-metrics-traffic"), "test-project")

		Expect(result).To(ContainSubstring("port: 8443"))
		Expect(result).NotTo(ContainSubstring(".Values."))
	})

	It("leaves a custom policy's ports untouched", func() {
		dnsPolicy := `spec:
  policyTypes:
    - Ingress
  ingress:
    - ports:
        - port: 5353
          protocol: UDP
`
		result := TemplatePorts(dnsPolicy, networkPolicy("test-project-allow-dns-traffic"), "test-project")

		Expect(result).To(ContainSubstring("port: 5353"))
		Expect(result).NotTo(ContainSubstring(".Values."))
	})

	It("leaves a custom policy with an unrelated prefix untouched", func() {
		policy := `spec:
  policyTypes:
    - Ingress
  ingress:
    - ports:
        - port: 8443
          protocol: TCP
`
		result := TemplatePorts(policy, networkPolicy("team-allow-metrics-traffic"), "test-project")

		Expect(result).To(ContainSubstring("port: 8443"))
		Expect(result).NotTo(ContainSubstring(".Values.metrics.port"))
	})

	It("does not rewrite a port that follows the ingress block", func() {
		policy := `spec:
  ingress:
    - ports:
        - port: 8443
          protocol: TCP
  someField:
    port: 9999
`
		result := TemplatePorts(policy, networkPolicy("test-project-allow-metrics-traffic"), "test-project")

		Expect(result).To(ContainSubstring("port: {{ .Values.metrics.port }}"))
		Expect(result).To(ContainSubstring("port: 9999"), "a port outside the ingress block must not be rewritten")
	})

	It("templates only the ingress port and preserves egress ports of a metrics policy", func() {
		mixedPolicy := `spec:
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - ports:
        - port: 8443
          protocol: TCP
  egress:
    - ports:
        - port: 53
          protocol: UDP
`
		result := TemplatePorts(mixedPolicy, networkPolicy("test-project-allow-metrics-traffic"), "test-project")

		Expect(result).To(ContainSubstring("port: {{ .Values.metrics.port }}"))
		Expect(result).NotTo(ContainSubstring("port: 8443"))
		Expect(result).To(ContainSubstring("port: 53"), "egress port must not be rewritten")
	})
})
