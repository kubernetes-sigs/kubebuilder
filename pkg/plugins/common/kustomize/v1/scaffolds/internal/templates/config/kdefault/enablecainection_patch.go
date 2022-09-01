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

package kdefault

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &WebhookCAInjectionPatch{}

// WebhookCAInjectionPatch scaffolds a file that defines the patch that adds annotation to webhooks
type WebhookCAInjectionPatch struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements file.Template
func (f *WebhookCAInjectionPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "webhookcainjection_patch.yaml")
	}

	f.TemplateBody = injectCAPatchTemplate

	// If file exists (ex. because a webhook was already created), skip creation.
	f.IfExistsAction = machinery.SkipFile

	return nil
}

const injectCAPatchTemplate = `# This patch add annotation to admission webhook config and
# the variables $(CERTIFICATE_NAMESPACE) and $(CERTIFICATE_NAME) will be substituted by kustomize.
apiVersion: admissionregistration.k8s.io/{{ .Resource.Webhooks.WebhookVersion }}
kind: MutatingWebhookConfiguration
metadata:
  labels:
    app.kubernetes.io/name: mutatingwebhookconfiguration
    app.kubernetes.io/instance: mutating-webhook-configuration
    app.kubernetes.io/component: webhook
    app.kubernetes.io/created-by: {{ .ProjectName }}
    app.kubernetes.io/part-of: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: mutating-webhook-configuration
  annotations:
    cert-manager.io/inject-ca-from: $(CERTIFICATE_NAMESPACE)/$(CERTIFICATE_NAME)
---
apiVersion: admissionregistration.k8s.io/{{ .Resource.Webhooks.WebhookVersion }}
kind: ValidatingWebhookConfiguration
metadata:
  labels:
    app.kubernetes.io/name: validatingwebhookconfiguration
    app.kubernetes.io/instance: validating-webhook-configuration
    app.kubernetes.io/component: webhook
    app.kubernetes.io/created-by: {{ .ProjectName }}
    app.kubernetes.io/part-of: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: validating-webhook-configuration
  annotations:
    cert-manager.io/inject-ca-from: $(CERTIFICATE_NAMESPACE)/$(CERTIFICATE_NAME)
`
