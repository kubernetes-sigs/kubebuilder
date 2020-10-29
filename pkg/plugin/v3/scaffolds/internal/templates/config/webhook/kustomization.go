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

package webhook

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/file"
)

var _ file.Template = &Kustomization{}

// Kustomization scaffolds a file that defines the kustomization scheme for the webhook folder
type Kustomization struct {
	file.TemplateMixin

	// Version of webhook the project was configured with.
	WebhookVersion string
}

// SetTemplateDefaults implements file.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "webhook", "kustomization.yaml")
	}

	f.TemplateBody = kustomizeWebhookTemplate

	// If file exists (ex. because a webhook was already created), skip creation.
	f.IfExistsAction = file.Skip

	if f.WebhookVersion == "" {
		f.WebhookVersion = "v1"
	}

	return nil
}

const kustomizeWebhookTemplate = `resources:
- manifests{{ if ne .WebhookVersion "v1" }}.{{ .WebhookVersion }}{{ end }}.yaml
- service.yaml

configurations:
- kustomizeconfig.yaml
`
