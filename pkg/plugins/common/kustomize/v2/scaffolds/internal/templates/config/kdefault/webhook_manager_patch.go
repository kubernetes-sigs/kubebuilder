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

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &ManagerWebhookPatch{}

// ManagerWebhookPatch scaffolds a file that defines the patch that enables webhooks on the manager
type ManagerWebhookPatch struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *ManagerWebhookPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "manager_webhook_patch.yaml")
	}

	f.TemplateBody = managerWebhookPatchTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		// If file exists (ex. because a webhook was already created), skip creation.
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

//nolint:lll
const managerWebhookPatchTemplate = `# This patch ensures the webhook certificates are properly mounted in the manager container.
# It configures the necessary arguments, volumes, volume mounts, and container ports.

# Add the --webhook-cert-path argument for configuring the webhook certificate path
- op: add
  path: /spec/template/spec/containers/0/args/-
  value: --webhook-cert-path=/tmp/k8s-webhook-server/serving-certs

# Add the volumeMount for the webhook certificates
- op: add
  path: /spec/template/spec/containers/0/volumeMounts/-
  value:
    mountPath: /tmp/k8s-webhook-server/serving-certs
    name: webhook-certs
    readOnly: true

# Add the port configuration for the webhook server
- op: add
  path: /spec/template/spec/containers/0/ports/-
  value:
    containerPort: 9443
    name: webhook-server
    protocol: TCP

# Add the volume configuration for the webhook certificates
- op: add
  path: /spec/template/spec/volumes/-
  value:
    name: webhook-certs
    secret:
      secretName: webhook-server-cert
`
