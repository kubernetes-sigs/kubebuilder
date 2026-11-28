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
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

var _ machinery.Template = &NetworkPolicy{}

// NetworkPolicy scaffolds default NetworkPolicy manifests for the Helm chart.
type NetworkPolicy struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// Webhook generates the webhook ingress policy instead of the metrics ingress policy.
	Webhook bool
	// PodSelector holds the matchLabels lines that select the manager pods, as the chart renders the
	// selector of the manager Deployment. When empty, the policy selects the scaffolded labels.
	PodSelector []string
	// OutputDir specifies the output directory for the chart.
	OutputDir string
	// Force if true allows overwriting the scaffolded file.
	Force bool
}

// matchLabelsLines returns the lines of a matchLabels selector the chart adds: lines, or defaults
// when there are none. Each line is written as a raw string, so the Helm actions in it reach the
// chart as they are.
func matchLabelsLines(lines, defaults []string) string {
	if len(lines) == 0 {
		lines = defaults
	}
	var matchLabels strings.Builder
	for _, line := range lines {
		matchLabels.WriteString("      {{`" + line + "`}}\n")
	}
	return matchLabels.String()
}

// SetTemplateDefaults implements machinery.Template.
func (f *NetworkPolicy) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = common.DefaultOutputDir
		}
		filename := "allow-metrics-traffic.yaml"
		if f.Webhook {
			filename = "allow-webhook-traffic.yaml"
		}
		f.Path = filepath.Join(outputDir, "chart", "templates", "network-policy", filename)
	}

	chartName := f.ProjectName
	matchLabels := matchLabelsLines(f.PodSelector, []string{
		`control-plane: controller-manager`,
		`app.kubernetes.io/name: {{ include "` + chartName + `.name" . }}`,
	})
	if f.Webhook {
		f.TemplateBody = fmt.Sprintf(webhookNetworkPolicyTemplate, chartName, matchLabels)
	} else {
		f.TemplateBody = fmt.Sprintf(networkPolicyTemplate, chartName, matchLabels)
	}

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

const networkPolicyTemplate = `{{` + "`" +
	`{{- if and .Values.networkPolicy.enabled .Values.metrics.enabled }}` + "`" + `}}
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%[1]s.name\" . }}" }}
  name: {{ "{{ include \"%[1]s.resourceName\" (dict \"suffix\" \"allow-metrics-traffic\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  podSelector:
    matchLabels:
%[2]s  policyTypes:
    - Ingress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            metrics: enabled
      ports:
        - port: {{ "{{ .Values.metrics.port }}" }}
          protocol: TCP
{{` + "`" + `{{- end }}` + "`" + `}}
`

const webhookNetworkPolicyTemplate = `{{` + "`" +
	`{{- if and .Values.networkPolicy.enabled .Values.webhook.enabled }}` + "`" + `}}
---
# Allow ingress to the webhook server's pod port from all sources.
# The webhook Service forwards traffic from its Service port to the pod port.
# The Kubernetes API server uses the Service to reach admission and CRD conversion webhooks.
# To restrict admission requests by namespace, configure namespaceSelector in the
# MutatingWebhookConfiguration or ValidatingWebhookConfiguration.
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%[1]s.name\" . }}" }}
  name: {{ "{{ include \"%[1]s.resourceName\" (dict \"suffix\" \"allow-webhook-traffic\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  podSelector:
    matchLabels:
%[2]s  policyTypes:
    - Ingress
  ingress:
    - ports:
        - port: {{ "{{ .Values.webhook.port }}" }}
          protocol: TCP
{{` + "`" + `{{- end }}` + "`" + `}}
`
