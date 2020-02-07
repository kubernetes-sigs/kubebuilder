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
	"time"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Boilerplate{}

// Boilerplate scaffolds a boilerplate header file.
type Boilerplate struct {
	file.TemplateMixin
	file.BoilerplateMixin

	// License is the License type to write
	License string

	// Owner is the copyright owner - e.g. "The Kubernetes Authors"
	Owner string

	// Year is the copyright year
	Year string
}

// SetTemplateDefaults implements input.Template
func (f *Boilerplate) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("hack", "boilerplate.go.txt")
	}

	if f.Year == "" {
		f.Year = fmt.Sprintf("%v", time.Now().Year())
	}

	// Boilerplate given
	if len(f.Boilerplate) > 0 {
		f.TemplateBody = f.Boilerplate
		return nil
	}

	// Pick a template boilerplate option
	switch f.License {
	case "", "apache2":
		f.TemplateBody = apache
	case "none":
		f.TemplateBody = none
	}

	return nil
}

const apache = `/*
{{ if .Owner -}}
Copyright {{ .Year }} {{ .Owner }}.
{{- end }}

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/`

const none = `/*
{{ if .Owner -}}
Copyright {{ .Year }} {{ .Owner }}.
{{- end }}
*/`
