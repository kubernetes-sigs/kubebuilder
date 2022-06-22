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

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &KustomizeConfig{}

// KustomizeConfig scaffolds a file that configures the kustomization for the webhook folder
type KustomizeConfig struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements file.Template
func (f *KustomizeConfig) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "webhook", "kustomizeconfig.yaml")
	}

	f.TemplateBody = kustomizeConfigWebhookTemplate

	// If file exists (ex. because a webhook was already created), skip creation.
	f.IfExistsAction = machinery.SkipFile

	return nil
}

//nolint:lll
const kustomizeConfigWebhookTemplate = `# the following config is for teaching kustomize where to look at when substituting vars.
# It requires kustomize v2.1.0 or newer to work properly.
nameReference:
- kind: Service
  version: v1
  fieldSpecs:
  - kind: MutatingWebhookConfiguration
    group: admissionregistration.k8s.io
    path: webhooks/clientConfig/service/name
  - kind: ValidatingWebhookConfiguration
    group: admissionregistration.k8s.io
    path: webhooks/clientConfig/service/name

namespace:
- kind: MutatingWebhookConfiguration
  group: admissionregistration.k8s.io
  path: webhooks/clientConfig/service/namespace
  create: true
- kind: ValidatingWebhookConfiguration
  group: admissionregistration.k8s.io
  path: webhooks/clientConfig/service/namespace
  create: true

varReference:
- path: metadata/annotations
`
