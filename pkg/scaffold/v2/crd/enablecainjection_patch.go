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

	"github.com/markbates/inflect"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
)

// EnableCAInjectionPatch scaffolds a EnableCAInjectionPatch for a Resource
type EnableCAInjectionPatch struct {
	input.Input

	// Resource is the Resource to make the EnableCAInjectionPatch for
	Resource *resource.Resource
}

// GetInput implements input.File
func (p *EnableCAInjectionPatch) GetInput() (input.Input, error) {
	if p.Path == "" {
		rs := inflect.NewDefaultRuleset()
		plural := rs.Pluralize(strings.ToLower(p.Resource.Kind))
		p.Path = filepath.Join("config", "crd", "patches",
			fmt.Sprintf("cainjection_in_%s.yaml", plural))
	}
	p.TemplateBody = EnableCAInjectionPatchTemplate
	return p.Input, nil
}

// Validate validates the values
func (g *EnableCAInjectionPatch) Validate() error {
	return g.Resource.Validate()
}

var EnableCAInjectionPatchTemplate = `# The following patch adds a directive for certmanager to inject CA into the CRD
# CRD conversion requires k8s 1.13 or later.
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    certmanager.k8s.io/inject-ca-from: $(NAMESPACE)/$(CERTIFICATENAME)
  name: {{ .Resource.Resource }}.{{ .Resource.Group }}.{{ .Domain }}
`
