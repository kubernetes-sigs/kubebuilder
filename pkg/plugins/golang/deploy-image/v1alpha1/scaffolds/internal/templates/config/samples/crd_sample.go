/*
Copyright 2022 The Kubernetes Authors.
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

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &CRDSample{}

// CRDSample scaffolds a file that defines a sample manifest for the CRD
type CRDSample struct {
	machinery.TemplateMixin
	machinery.ResourceMixin

	// Port if informed we will create the scaffold with this spec
	Port string
}

// SetTemplateDefaults implements file.Template
func (f *CRDSample) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.Resource.Group != "" {
			f.Path = filepath.Join("config", "samples", "%[group]_%[version]_%[kind].yaml")
		} else {
			f.Path = filepath.Join("config", "samples", "%[version]_%[kind].yaml")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Println(f.Path)

	f.IfExistsAction = machinery.OverwriteFile

	f.TemplateBody = crdSampleTemplate

	return nil
}

const crdSampleTemplate = `apiVersion: {{ .Resource.QualifiedGroup }}/{{ .Resource.Version }}
kind: {{ .Resource.Kind }}
metadata:
  name: {{ lower .Resource.Kind }}-sample
spec:
  # TODO(user): edit the following value to ensure the number
  # of Pods/Instances your Operand must have on cluster
  size: 1
{{ if not (isEmptyStr .Port) }}
  # TODO(user): edit the following value to ensure the container has the right port to be initialized
  containerPort: {{ .Port }}
{{ end -}}
`
