/*
Copyright 2025 The Kubernetes Authors.

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

var _ machinery.Template = &NamespaceTransformer{}

// NamespaceTransformer scaffolds a file that defines the namespace transformer for the default overlay folder
type NamespaceTransformer struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *NamespaceTransformer) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "namespace-transformer.yaml")
	}

	f.TemplateBody = namespaceTransformerTemplate

	f.IfExistsAction = machinery.Error

	return nil
}

const namespaceTransformerTemplate = `# This namespace transformer adds a namespace to resources that don't
# already have one. Resources with an explicitly set namespace (e.g., from RBAC markers) will
# preserve their namespace. This allows controllers to manage resources across multiple namespaces
# while applying a default namespace to resources that don't specify one.
#
# More info: https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_namespacetransformer_
apiVersion: builtin
kind: NamespaceTransformer
metadata:
  name: namespace-transformer
  namespace: {{ .ProjectName }}-system
# Only add namespace to resources that don't have one set
# This preserves namespaces from RBAC markers (e.g., +kubebuilder:rbac:namespace=infrastructure)
unsetOnly: true
`
