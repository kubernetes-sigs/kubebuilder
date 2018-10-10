/*
Copyright 2018 The Kubernetes Authors.

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

package resource

import (
	"fmt"
	"path/filepath"

	"strings"

	"github.com/markbates/inflect"
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &CRD{}

// CRD scaffolds a CRD yaml file.
type CRD struct {
	input.Input

	// Scope is Namespaced or Cluster
	Scope string

	// Plural is the plural lowercase of kind
	Plural string

	// Resource is a resource in the API group
	Resource *Resource
}

// GetInput implements input.File
func (c *CRD) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "crds", fmt.Sprintf(
			"%s_%s_%s.yaml", c.Resource.Group, c.Resource.Version, strings.ToLower(c.Resource.Kind)))
	}
	c.Scope = "Namespaced"
	if !c.Resource.Namespaced {
		c.Scope = "Cluster"
	}
	if c.Plural == "" {
		c.Plural = strings.ToLower(inflect.Pluralize(c.Resource.Kind))
	}

	c.IfExistsAction = input.Error
	c.TemplateBody = crdTemplate
	return c.Input, nil
}

var crdTemplate = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: {{ .Plural }}.{{ .Resource.Group }}.{{ .Domain }}
spec:
  group: {{ .Resource.Group }}.{{ .Domain }}
  version: "{{ .Resource.Version }}"
  names:
    kind: {{ .Resource.Kind }}
    plural: {{ .Plural }}
  scope: {{ .Scope }}
`
