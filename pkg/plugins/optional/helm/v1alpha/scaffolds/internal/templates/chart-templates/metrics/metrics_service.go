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

package metrics

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Service{}

// Service scaffolds the Service for metrics in the Helm chart
type Service struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults sets the default template configuration
func (f *Service) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", "templates", "metrics", "metrics-service.yaml")
	}

	f.TemplateBody = metricsServiceTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const metricsServiceTemplate = `{{` + "`" + `{{- if .Values.metrics.enable }}` + "`" + `}}
apiVersion: v1
kind: Service
metadata:
  name: {{ .ProjectName }}-controller-manager-metrics-service
  namespace: {{ "{{ .Release.Namespace }}" }}
  labels:
    {{ "{{- include \"chart.labels\" . | nindent 4 }}" }}
spec:
  ports:
    - port: 8443
      targetPort: 8443
      protocol: TCP
      name: https
  selector:
    control-plane: controller-manager
{{` + "`" + `{{- end }}` + "`" + `}}
`
