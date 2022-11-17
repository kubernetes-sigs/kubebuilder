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
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Kustomization{}

// Kustomization scaffolds a file that defines the kustomization scheme for the prometheus folder
type Kustomization struct {
	machinery.TemplateMixin

	CRDManifests []string
}

// SetTemplateDefaults implements file.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "samples", "kustomization.yaml")
	}

	defaultTemplate, err := f.createTemplate()
	if err != nil {
		return err
	}

	f.TemplateBody = defaultTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

func (f *Kustomization) createTemplate() (string, error) {
	t := template.Must(template.New("customResourcesConfig").Parse(kustomizationTemplate))

	outputTmpl := &bytes.Buffer{}
	if err := t.Execute(outputTmpl, f.CRDManifests); err != nil {
		return "", fmt.Errorf("error when generating sample kustomization manifest: %w", err)
	}

	return outputTmpl.String(), nil

}

const kustomizationTemplate = `---
resources:{{ range $i ,$e := . }}
  - {{ . }}{{end}}
`
