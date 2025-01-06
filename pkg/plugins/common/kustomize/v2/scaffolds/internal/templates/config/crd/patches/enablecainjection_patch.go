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

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &EnableCAInjectionPatch{}

// EnableCAInjectionPatch scaffolds a file that defines the patch that injects CA into the CRD
type EnableCAInjectionPatch struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *EnableCAInjectionPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("config", "crd", "patches", "cainjection_in_%[group]_%[plural].yaml")
		} else {
			f.Path = filepath.Join("config", "crd", "patches", "cainjection_in_%[plural].yaml")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = enableCAInjectionPatchTemplate

	return nil
}

const enableCAInjectionPatchTemplate = `# The following patch adds a directive for certmanager to inject CA into the CRD
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    cert-manager.io/inject-ca-from: CERTIFICATE_NAMESPACE/CERTIFICATE_NAME
  name: {{ .Resource.Plural }}.{{ .Resource.QualifiedGroup }}
`
