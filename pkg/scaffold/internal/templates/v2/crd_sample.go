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

package v2

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &CRDSample{}

// CRDSample scaffolds a manifest for CRD sample.
type CRDSample struct {
	file.Input
	file.ResourceMixin
}

// GetInput implements input.Template
func (f *CRDSample) GetInput() (file.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("config", "samples", fmt.Sprintf(
			"%s_%s_%s.yaml", f.Resource.Group, f.Resource.Version, strings.ToLower(f.Resource.Kind)))
	}

	f.IfExistsAction = file.Error
	f.TemplateBody = crdSampleTemplate
	return f.Input, nil
}

// Validate validates the values
func (f *CRDSample) Validate() error {
	return f.Resource.Validate()
}

const crdSampleTemplate = `apiVersion: {{ .Resource.Domain }}/{{ .Resource.Version }}
kind: {{ .Resource.Kind }}
metadata:
  name: {{ lower .Resource.Kind }}-sample
spec:
  # Add fields here
  foo: bar
`
