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

var _ input.File = &EnableCAInjectionPatch{}

// EnableCAInjectionPatch scaffolds a EnableCAInjectionPatch for a Resource
type EnableCAInjectionPatch struct {
	input.Input

	// Resource is the Resource to make the EnableCAInjectionPatch for
	Resource *resource.Resource
}

// GetInput implements input.File
func (f *EnableCAInjectionPatch) GetInput() (input.Input, error) {
	if f.Path == "" {
		plural := flect.Pluralize(strings.ToLower(f.Resource.Kind))
		f.Path = filepath.Join("config", "crd", "patches",
			fmt.Sprintf("cainjection_in_%s.yaml", plural))
	}
	f.TemplateBody = EnableCAInjectionPatchTemplate
	return f.Input, nil
}

// Validate validates the values
func (f *EnableCAInjectionPatch) Validate() error {
	return f.Resource.Validate()
}

const EnableCAInjectionPatchTemplate = `# The following patch adds a directive for certmanager to inject CA into the CRD
# CRD conversion requires k8s 1.13 or later.
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    cert-manager.io/inject-ca-from: $(CERTIFICATE_NAMESPACE)/$(CERTIFICATE_NAME)
  name: {{ .Resource.Resource }}.{{ .Resource.Group }}.{{ .Domain }}
`
