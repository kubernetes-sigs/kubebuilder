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

package webhook

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Certificate{}

// Certificate scaffolds the Certificate for webhooks in the Helm chart
type Certificate struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// HasWebhooks is true when webhooks were found in the config
	HasWebhooks bool
}

// SetTemplateDefaults sets the default template configuration
func (f *Certificate) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", "templates", "certmanager", "certificate.yaml")
	}

	f.TemplateBody = certificateTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const certificateTemplate = `{{` + "`" + `{{- if .Values.certmanager.enable }}` + "`" + `}}
# Self-signed Issuer
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  labels:
    {{ "{{- include \"chart.labels\" . | nindent 4 }}" }}
  name: selfsigned-issuer
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  selfSigned: {}
{{- if .HasWebhooks }}
{{ "{{- if .Values.webhook.enable }}" }}
---
# Certificate for the webhook
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  annotations:
    {{ "{{- if .Values.crd.keep }}" }}
    "helm.sh/resource-policy": keep
    {{ "{{- end }}" }}
  name: serving-cert
  namespace: {{ "{{ .Release.Namespace }}" }}
  labels:
    {{ "{{- include \"chart.labels\" . | nindent 4 }}" }}
spec:
  dnsNames:
    - {{ .ProjectName }}.{{ "{{ .Release.Namespace }}" }}.svc
    - {{ .ProjectName }}.{{ "{{ .Release.Namespace }}" }}.svc.cluster.local
    - {{ .ProjectName }}-webhook-service.{{ "{{ .Release.Namespace }}" }}.svc
  issuerRef:
    kind: Issuer
    name: selfsigned-issuer
  secretName: webhook-server-cert
{{` + "`" + `{{- end }}` + "`" + `}}
{{- end }}
{{ "{{- if .Values.metrics.enable }}" }}
---
# Certificate for the metrics
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  annotations:
    {{ "{{- if .Values.crd.keep }}" }}
    "helm.sh/resource-policy": keep
    {{ "{{- end }}" }}
  labels:
    {{ "{{- include \"chart.labels\" . | nindent 4 }}" }}
  name: metrics-certs
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  dnsNames:
    - {{ .ProjectName }}.{{ "{{ .Release.Namespace }}" }}.svc
    - {{ .ProjectName }}.{{ "{{ .Release.Namespace }}" }}.svc.cluster.local
    - {{ .ProjectName }}-metrics-service.{{ "{{ .Release.Namespace }}" }}.svc
  issuerRef:
    kind: Issuer
    name: selfsigned-issuer
  secretName: metrics-server-cert
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
`
