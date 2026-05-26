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
	// OutputDir specifies the output directory for the chart.
	OutputDir string
	// Force if true allows overwriting the scaffolded file.
	Force bool
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
	if f.Webhook {
		f.TemplateBody = fmt.Sprintf(webhookNetworkPolicyTemplate, chartName, chartName, chartName)
	} else {
		f.TemplateBody = fmt.Sprintf(networkPolicyTemplate, chartName, chartName, chartName)
	}

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

const networkPolicyTemplate = `{{` + "`" + `{{- if .Values.networkPolicy.enable }}` + "`" + `}}
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"allow-metrics-traffic\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
  policyTypes:
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
	`{{- if and .Values.networkPolicy.enable .Values.webhook.enable }}` + "`" + `}}
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"allow-webhook-traffic\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
  policyTypes:
    - Ingress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            webhook: enabled
      ports:
        - port: {{ "{{ .Values.webhook.port }}" }}
          protocol: TCP
{{` + "`" + `{{- end }}` + "`" + `}}
`
