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

// NamespaceTransformer scaffolds a file that defines the NamespaceTransformer
// for the default overlay folder
type NamespaceTransformer struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *NamespaceTransformer) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "namespace_transformer.yaml")
	}

	f.TemplateBody = namespaceTransformerTemplate

	return nil
}

const namespaceTransformerTemplate = `# This NamespaceTransformer is an alternative to the
# "namespace:" field in kustomization.yaml.
# While the "namespace:" field overrides ALL namespaces
# (potentially breaking multi-namespace setups), this transformer
# with "unsetOnly: true" only sets namespace on resources
# that don't already have one.
#
# This is useful when using namespace-scoped RBAC markers such as:
#   //+kubebuilder:rbac:groups=apps,namespace=infrastructure,...
# which generate Roles with explicit namespaces that should
# be preserved.
#
# TIP: When using multiple namespaces, also consider using the
# "roleName" parameter in your RBAC markers to give each Role
# a unique name, e.g.:
#   //+kubebuilder:rbac:namespace=infra,roleName=infra-role,...
# This avoids ID conflicts even before switching to the
# NamespaceTransformer.
#
# Usage:
#   1. Comment out the "namespace:" field in kustomization.yaml
#   2. Uncomment the "transformers:" section in kustomization.yaml
#   3. Remove "namespace: system" from any resource files under
#      config/ that should receive the default namespace from
#      this transformer instead
#
# More info:
#   https://kubectl.docs.kubernetes.io/references/kustomize/builtins/
#   #_namespacetransformer_
apiVersion: builtin
kind: NamespaceTransformer
metadata:
  name: namespace-transformer
  namespace: {{ .ProjectName }}-system
setRoleBindingSubjects: allServiceAccounts
unsetOnly: true
fieldSpecs:
  - path: metadata/namespace
    create: true
`
