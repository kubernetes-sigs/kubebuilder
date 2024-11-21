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

package samples

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &CRDSample{}

// CRDSample scaffolds a file that defines a sample manifest for the CRD
type CRDSample struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
	machinery.ProjectNameMixin

	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *CRDSample) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.Resource.Group != "" {
			f.Path = filepath.Join("config", "samples", "%[group]_%[version]_%[kind].yaml")
		} else {
			f.Path = filepath.Join("config", "samples", "%[version]_%[kind].yaml")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.Error
	}

	f.TemplateBody = crdSampleTemplate

	return nil
}

const crdSampleTemplate = `apiVersion: {{ .Resource.QualifiedGroup }}/{{ .Resource.Version }}
kind: {{ .Resource.Kind }}
metadata:
  labels:
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: {{ lower .Resource.Kind }}-sample
spec:
  # TODO(user): Add fields here
`
