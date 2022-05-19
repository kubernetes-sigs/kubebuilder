/*
Copyright 2020 The Kubernetes Authors.

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

package patches

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &EnableWebhookPatch{}

// EnableWebhookPatch scaffolds a file that defines the patch that enables conversion webhook for the CRD
type EnableWebhookPatch struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements file.Template
func (f *EnableWebhookPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup {
			f.Path = filepath.Join("config", "crd", "patches", "webhook_in_%[group]_%[plural].yaml")
		} else {
			f.Path = filepath.Join("config", "crd", "patches", "webhook_in_%[plural].yaml")
		}

	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = enableWebhookPatchTemplate

	return nil
}

const enableWebhookPatchTemplate = `# The following patch enables a conversion webhook for the CRD
{{- if ne .Resource.API.CRDVersion "v1" }}
# CRD conversion requires k8s 1.13 or later.
{{- end }}
apiVersion: apiextensions.k8s.io/{{ .Resource.API.CRDVersion }}
kind: CustomResourceDefinition
metadata:
  name: {{ .Resource.Plural }}.{{ .Resource.QualifiedGroup }}
spec:
  conversion:
    strategy: Webhook
    {{- if ne .Resource.API.CRDVersion "v1" }}
    webhookClientConfig:
      service:
        namespace: system
        name: webhook-service
        path: /convert
    {{- else }}
    webhook:
      clientConfig:
        service:
          namespace: system
          name: webhook-service
          path: /convert
      conversionReviewVersions:
      - {{ .Resource.API.CRDVersion }}
    {{- end }}
`
