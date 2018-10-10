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

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &CRD{}

// CRDSample scaffolds a manifest for CRD sample.
type CRDSample struct {
	input.Input

	// Resource is a resource in the API group
	Resource *Resource
}

// GetInput implements input.File
func (c *CRDSample) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "samples", fmt.Sprintf(
			"%s_%s_%s.yaml", c.Resource.Group, c.Resource.Version, strings.ToLower(c.Resource.Kind)))
	}

	c.IfExistsAction = input.Error
	c.TemplateBody = crdSampleTemplate
	return c.Input, nil
}

var crdSampleTemplate = `apiVersion: {{ .Resource.Group }}.{{ .Domain }}/{{ .Resource.Version }}
kind: {{ .Resource.Kind }}
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: {{ lower .Resource.Kind }}-sample
spec:
  # Add fields here
  foo: bar
`
