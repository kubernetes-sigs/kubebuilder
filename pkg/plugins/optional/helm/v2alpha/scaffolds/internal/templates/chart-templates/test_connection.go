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
)

var _ machinery.Template = &TestConnection{}

// TestConnection scaffolds a Helm test that verifies the manager deployment is healthy
type TestConnection struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// OutputDir specifies the output directory for the chart
	OutputDir string

	// HasWebhooks determines if webhook verification code is included in the generated chart
	// When true, adds webhook service checks wrapped in Helm's .Values.webhook.enable conditional
	HasWebhooks bool

	// Force if true allows overwriting the scaffolded file
	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *TestConnection) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = defaultOutputDir
		}
		f.Path = filepath.Join(outputDir, "chart", "templates", "tests", "test-connection.yaml")
	}

	prefix := f.ProjectName
	f.TemplateBody = fmt.Sprintf(testConnectionTemplate,
		prefix, prefix, // ServiceAccount
		prefix, prefix, // Role
		prefix, prefix, prefix, prefix, // RoleBinding
		prefix, prefix, prefix, // Pod
		prefix, // Webhook service
		prefix, // Metrics service
	)

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

//nolint:lll
const testConnectionTemplate = `---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-sa\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-role\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
rules:
  - apiGroups:
      - ""
    resources:
      - services
    verbs:
      - get
      - list
  - apiGroups:
      - apps
    resources:
      - deployments
    verbs:
      - get
      - list
      - watch
  {{ "{{- if .Values.certManager.enable }}" }}
  - apiGroups:
      - cert-manager.io
    resources:
      - certificates
    verbs:
      - get
      - list
      - watch
  {{ "{{- end }}" }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-rolebinding\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-role\" \"context\" $) }}" }}
subjects:
  - kind: ServiceAccount
    name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-sa\" \"context\" $) }}" }}
    namespace: {{ "{{ .Release.Namespace }}" }}
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
  annotations:
    "helm.sh/hook": test
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-connection\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  restartPolicy: Never
  serviceAccountName: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-sa\" \"context\" $) }}" }}
  containers:
    - name: test
      image: bitnami/kubectl:latest
      command:
        - /bin/sh
        - -c
        - |
          set -ex
          echo "Testing Helm chart deployment..."
          echo "Namespace: {{ "{{ .Release.Namespace }}" }}"

          {{ "{{- if .Values.certManager.enable }}" }}
          # Wait for cert-manager certificates
          echo "Waiting for certificates..."
          kubectl wait --for=condition=Ready certificate --all -n {{ "{{ .Release.Namespace }}" }} --timeout=3m || {
            echo "Warning: Certificate wait failed or no certificates found"
          }
          {{ "{{- end }}" }}

          # Wait for manager deployment to be ready
          echo "Waiting for manager deployment..."
          if ! kubectl wait --for=condition=Available deployment \
            -l control-plane=controller-manager \
            -n {{ "{{ .Release.Namespace }}" }} --timeout=5m; then
            echo "ERROR: Manager deployment failed to become available"
            kubectl get deployments -n {{ "{{ .Release.Namespace }}" }}
            kubectl get pods -n {{ "{{ .Release.Namespace }}" }}
            exit 1
          fi

          echo "Manager deployment is ready"
{{ if .HasWebhooks }}
          {{ "{{- if .Values.webhook.enable }}" }}
          echo "Verifying Webhook Service..."
          if ! kubectl get svc {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"webhook-service\" \"context\" $) }}" }} \
            -n {{ "{{ .Release.Namespace }}" }}; then
            echo "ERROR: Webhook service not found"
            kubectl get svc -n {{ "{{ .Release.Namespace }}" }}
            exit 1
          fi
          {{ "{{- end }}" }}
{{ end }}
          {{ "{{- if .Values.metrics.enable }}" }}
          echo "Verifying Metrics Service..."
          if ! kubectl get svc {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"controller-manager-metrics-service\" \"context\" $) }}" }} \
            -n {{ "{{ .Release.Namespace }}" }}; then
            echo "ERROR: Metrics service not found"
            kubectl get svc -n {{ "{{ .Release.Namespace }}" }}
            exit 1
          fi
          {{ "{{- end }}" }}

          echo "All tests passed successfully!"
`
