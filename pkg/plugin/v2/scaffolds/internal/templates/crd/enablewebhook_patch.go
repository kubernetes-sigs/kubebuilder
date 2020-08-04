/*
Copyright 2019 The Kubernetes Authors.

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

package crd

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &EnableWebhookPatch{}

// EnableWebhookPatch scaffolds a EnableWebhookPatch for a Resource
type EnableWebhookPatch struct {
	file.TemplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements input.Template
func (f *EnableWebhookPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "patches", "webhook_in_%[plural].yaml")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = enableWebhookPatchTemplate

	return nil
}

const enableWebhookPatchTemplate = `# The following patch enables conversion webhook for CRD
# CRD conversion requires k8s 1.13 or later.
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{ .Resource.Plural }}.{{ .Resource.Domain }}
spec:
  conversion:
    strategy: Webhook
    webhookClientConfig:
      # this is "\n" used as a placeholder, otherwise it will be rejected by the apiserver for being blank,
      # but we're going to set it later using the cert-manager (or potentially a patch if not using cert-manager)
      caBundle: Cg==
      service:
        namespace: system
        name: webhook-service
        path: /convert
`
