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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/flect"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

// EnableWebhookPatch scaffolds a EnableWebhookPatch for a Resource
type EnableWebhookPatch struct {
	input.Input

	// Resource is the Resource to make the EnableWebhookPatch for
	Resource *resource.Resource
}

// GetInput implements input.File
func (p *EnableWebhookPatch) GetInput() (input.Input, error) {
	if p.Path == "" {
		plural := flect.Pluralize(strings.ToLower(p.Resource.Kind))
		p.Path = filepath.Join("config", "crd", "patches",
			fmt.Sprintf("webhook_in_%s.yaml", plural))
	}
	p.TemplateBody = enableWebhookPatchTemplate
	return p.Input, nil
}

const enableWebhookPatchTemplate = `# The following patch enables conversion webhook for CRD
# CRD conversion requires k8s 1.13 or later.
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{ .Resource.Resource }}.{{ .Resource.Group }}.{{ .Domain }}
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
