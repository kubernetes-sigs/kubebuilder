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

package manager

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Deployment{}

// Deployment scaffolds the manager Deployment for the Helm chart
type Deployment struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// DeployImages if true will scaffold the env with the images
	DeployImages bool
	// Force if true allow overwrite the scaffolded file
	Force bool
	// HasWebhooks is true when webhooks were found in the config
	HasWebhooks bool
}

// SetTemplateDefaults sets the default template configuration
func (f *Deployment) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", "templates", "manager", "manager.yaml")
	}

	f.TemplateBody = managerDeploymentTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

//nolint:lll
const managerDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .ProjectName }}-controller-manager
  namespace: {{ "{{ .Release.Namespace }}" }}
  labels:
    {{ "{{- include \"chart.labels\" . | nindent 4 }}" }}
    control-plane: controller-manager
spec:
  replicas:  {{ "{{ .Values.controllerManager.replicas }}" }}
  selector:
    matchLabels:
      {{ "{{- include \"chart.selectorLabels\" . | nindent 6 }}" }}
      control-plane: controller-manager
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        {{ "{{- include \"chart.labels\" . | nindent 8 }}" }}
        control-plane: controller-manager
        {{ "{{- if and .Values.controllerManager.pod .Values.controllerManager.pod.labels }}" }}
        {{ "{{- range $key, $value := .Values.controllerManager.pod.labels }}" }}
        {{ "{{ $key }}" }}: {{ "{{ $value }}" }}
        {{ "{{- end }}" }}
        {{ "{{- end }}" }}
    spec:
      containers:
        - name: manager
          args:
            {{ "{{- range .Values.controllerManager.container.args }}" }}
            - {{ "{{ . }}" }}
            {{ "{{- end }}" }}
          command:
            - /manager
          image: {{ "{{ .Values.controllerManager.container.image.repository }}" }}:{{ "{{ .Values.controllerManager.container.image.tag }}" }}
          {{ "{{- if .Values.controllerManager.container.env }}" }}
          env:
            {{ "{{- range $key, $value := .Values.controllerManager.container.env }}" }}
            - name: {{ "{{ $key }}" }}
              value: {{ "{{ $value }}" }}
            {{ "{{- end }}" }}
          {{ "{{- end }}" }}
          livenessProbe:
            {{ "{{- toYaml .Values.controllerManager.container.livenessProbe | nindent 12 }}" }}
          readinessProbe:
            {{ "{{- toYaml .Values.controllerManager.container.readinessProbe | nindent 12 }}" }}
{{- if .HasWebhooks }}
          {{ "{{- if .Values.webhook.enable }}" }}
          ports:
            - containerPort: 9443
              name: webhook-server
              protocol: TCP
          {{ "{{- end }}" }}
{{- end }}
          resources:
            {{ "{{- toYaml .Values.controllerManager.container.resources | nindent 12 }}" }}
          securityContext:
            {{ "{{- toYaml .Values.controllerManager.container.securityContext | nindent 12 }}" }}
{{- if .HasWebhooks }}
          {{ "{{- if and .Values.certmanager.enable (or .Values.webhook.enable .Values.metrics.enable) }}" }}
{{- else }}
          {{ "{{- if and .Values.certmanager.enable .Values.metrics.enable }}" }}
{{- end }}
          volumeMounts:
{{- if .HasWebhooks }}
            {{ "{{- if and .Values.webhook.enable .Values.certmanager.enable }}" }}
            - name: webhook-cert
              mountPath: /tmp/k8s-webhook-server/serving-certs
              readOnly: true
            {{ "{{- end }}" }}
{{- end }}
            {{ "{{- if and .Values.metrics.enable .Values.certmanager.enable }}" }}
            - name: metrics-certs
              mountPath: /tmp/k8s-metrics-server/metrics-certs
              readOnly: true
            {{ "{{- end }}" }}
          {{ "{{- end }}" }}
      securityContext:
        {{ "{{- toYaml .Values.controllerManager.securityContext | nindent 8 }}" }}
      serviceAccountName: {{ "{{ .Values.controllerManager.serviceAccountName }}" }}
      terminationGracePeriodSeconds: {{ "{{ .Values.controllerManager.terminationGracePeriodSeconds }}" }}
{{- if .HasWebhooks }}
      {{ "{{- if and .Values.certmanager.enable (or .Values.webhook.enable .Values.metrics.enable) }}" }}
{{- else }}
      {{ "{{- if and .Values.certmanager.enable .Values.metrics.enable }}" }}
{{- end }}
      volumes:
{{- if .HasWebhooks }}
        {{ "{{- if and .Values.webhook.enable .Values.certmanager.enable }}" }}
        - name: webhook-cert
          secret:
            secretName: webhook-server-cert
        {{ "{{- end }}" }}
{{- end }}
        {{ "{{- if and .Values.metrics.enable .Values.certmanager.enable }}" }}
        - name: metrics-certs
          secret:
            secretName: metrics-server-cert
        {{ "{{- end }}" }}
      {{ "{{- end }}" }}
`
