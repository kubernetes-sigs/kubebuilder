/*
Copyright 2024 The Kubernetes Authors.

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

package templates

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &HelmValues{}

// HelmValues scaffolds a file that defines the values.yaml structure for the Helm chart
type HelmValues struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// DeployImages stores the images used for the DeployImage plugin
	DeployImages map[string]string
	// Force if true allows overwriting the scaffolded file
	Force bool
	// HasWebhooks is true when webhooks were found in the config
	HasWebhooks bool
	// ManagerValues contains values extracted from the manager.yaml file
	ManagerValues map[string]interface{}
}

// SetTemplateDefaults implements machinery.Template
func (f *HelmValues) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", "values.yaml")
	}
	f.TemplateBody = helmValuesTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

const helmValuesTemplate = `# [MANAGER]: Manager Deployment Configurations
controllerManager:
  replicas: {{ if and .ManagerValues (index .ManagerValues "replicas") -}}
    {{ index .ManagerValues "replicas" -}}
    {{ else -}}
    1
    {{- end }}
  container:
    image:
      repository: controller
      tag: latest
    args:
      {{- if and .ManagerValues (index .ManagerValues "args") }}
      {{- range $arg := index .ManagerValues "args" }}
      - "{{ $arg }}"
      {{- end }}
      {{- else }}
      - "--leader-elect"
      - "--metrics-bind-address=:8443"
      - "--health-probe-bind-address=:8081"
      {{- end }}
    resources:
      {{- if and .ManagerValues (index .ManagerValues "resources") }}
      limits:
        {{- range $key, $value := index (index .ManagerValues "resources") "limits" }}
        {{ $key }}: {{ $value }}
        {{- end }}
      requests:
        {{- range $key, $value := index (index .ManagerValues "resources") "requests" }}
        {{ $key }}: {{ $value }}
        {{- end }}
      {{- else }}
      limits:
        cpu: 500m
        memory: 128Mi
      requests:
        cpu: 10m
        memory: 64Mi
      {{- end }}
    livenessProbe:
      {{- if and .ManagerValues (index .ManagerValues "livenessProbe") }}
      {{- range $key, $value := index .ManagerValues "livenessProbe" }}
      {{ $key }}: {{ $value }}
      {{- end }}
      {{- else }}
      initialDelaySeconds: 15
      periodSeconds: 20
      httpGet:
        path: /healthz
        port: 8081
      {{- end }}
    readinessProbe:
      {{- if and .ManagerValues (index .ManagerValues "readinessProbe") }}
      {{- range $key, $value := index .ManagerValues "readinessProbe" }}
      {{ $key }}: {{ $value }}
      {{- end }}
      {{- else }}
      initialDelaySeconds: 5
      periodSeconds: 10
      httpGet:
        path: /readyz
        port: 8081
      {{- end }}
    {{- if .DeployImages }}
    env:
    {{- range $kind, $image := .DeployImages }}
      {{ $kind }}_IMAGE: {{ $image }}
    {{- end }}
    {{- end }}
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - "ALL"
  securityContext:
    {{- if and .ManagerValues (index .ManagerValues "securityContext") }}
    {{- range $key, $value := index .ManagerValues "securityContext" }}
    {{ $key }}: {{ $value }}
    {{- end }}
    {{- else }}
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
    {{- end }}
  terminationGracePeriodSeconds: {{ if and .ManagerValues (index .ManagerValues "terminationGracePeriodSeconds") -}}
    {{ index .ManagerValues "terminationGracePeriodSeconds" -}}
    {{ else -}}
    10
    {{- end }}
  serviceAccountName: {{ .ProjectName }}-controller-manager

# [RBAC]: To enable RBAC (Permissions) configurations
rbac:
  enable: true

# [CRDs]: To enable the CRDs
crd:
  # This option determines whether the CRDs are included
  # in the installation process.
  enable: true

  # Enabling this option adds the "helm.sh/resource-policy": keep
  # annotation to the CRD, ensuring it remains installed even when
  # the Helm release is uninstalled.
  # NOTE: Removing the CRDs will also remove all cert-manager CR(s)
  # (Certificates, Issuers, ...) due to garbage collection.
  keep: true

# [METRICS]: Set to true to generate manifests for exporting metrics.
# To disable metrics export set false, and ensure that the
# ControllerManager argument "--metrics-bind-address=:8443" is removed.
metrics:
  enable: true
{{ if .HasWebhooks }}
# [WEBHOOKS]: Webhooks configuration
# The following configuration is automatically generated from the manifests
# generated by controller-gen. To update run 'make manifests' and
# the edit command with the '--force' flag
webhook:
  enable: true
{{ end }}
# [PROMETHEUS]: To enable a ServiceMonitor to export metrics to Prometheus set true
prometheus:
  enable: false

# [CERT-MANAGER]: To enable cert-manager injection to webhooks set true
certmanager:
  enable: {{ .HasWebhooks }}

# [NETWORK POLICIES]: To enable NetworkPolicies set true
networkPolicy:
  enable: false
`
