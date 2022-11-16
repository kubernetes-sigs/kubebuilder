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

package templates

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

var _ machinery.Template = &ConfigManifest{}

// Kustomization scaffolds a file that defines the kustomization scheme for the prometheus folder
type ConfigManifest struct {
	machinery.TemplateMixin

	// Custom Resource to be instrumented
	GVKs []resource.GVK
}

// SetTemplateDefaults implements file.Template
func (f *ConfigManifest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("kube-state-metrics", "cr-config.yaml")
	}

	defaultTemplate, err := f.createTemplate()
	if err != nil {
		return err
	}

	f.TemplateBody = defaultTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

func (f *ConfigManifest) createTemplate() (string, error) {
	t := template.Must(template.New("customResourcesConfig").Parse(customResourcesConfigTemplate))

	outputTmpl := &bytes.Buffer{}
	if err := t.Execute(outputTmpl, f.GVKs); err != nil {
		return "", fmt.Errorf("error when generating kube-state-metrics config: %w", err)
	}

	return outputTmpl.String(), nil
}

// nolint: lll
const customResourcesConfigTemplate = `---
kind: CustomResourceStateMetrics
spec:
  resources:{{ range $i, $e := . }}
    - groupVersionKind:
        group: "{{.Group}}.{{.Domain}}"
        kind: "{{.Kind}}"
        version: "{{.Version}}"
      labelsFromPath:
        name:
          - metadata
          - name
        namespace:
          - metadata
          - namespace
        uid:
          - metadata
          - uid
      metrics:
        - name: "info"
          each:
            type: Info
            info:
              labelsFromPath:
                version:
                  - spec
                  - version{{ end }}
`
