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

package hack

import (
	"fmt"
	"path/filepath"
	"time"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// DefaultBoilerplatePath is the default path to the boilerplate file
var DefaultBoilerplatePath = filepath.Join("hack", "boilerplate.go.txt")

var _ machinery.Template = &Boilerplate{}

// Boilerplate scaffolds a file that defines the common header for the rest of the files
type Boilerplate struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin

	// License is the License type to write
	License string

	// Licenses maps License types to their actual string
	Licenses map[string]string

	// Owner is the copyright owner - e.g. "The Kubernetes Authors"
	Owner string

	// Year is the copyright year
	Year string
}

// Validate implements file.RequiresValidation
func (f Boilerplate) Validate() error {
	if f.License != "" {
		if _, found := knownLicenses[f.License]; !found {
			if _, found := f.Licenses[f.License]; !found {
				return fmt.Errorf("unknown specified license %s", f.License)
			}
		}
	}
	return nil
}

// SetTemplateDefaults implements machinery.Template
func (f *Boilerplate) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = DefaultBoilerplatePath
	}

	if f.License == "" {
		f.License = "apache2"
	}

	if f.Licenses == nil {
		f.Licenses = make(map[string]string, len(knownLicenses))
	}

	for key, value := range knownLicenses {
		if _, hasLicense := f.Licenses[key]; !hasLicense {
			f.Licenses[key] = value
		}
	}

	if f.Year == "" {
		f.Year = fmt.Sprintf("%v", time.Now().Year())
	}

	// Boilerplate given
	if len(f.Boilerplate) > 0 {
		f.TemplateBody = f.Boilerplate
		return nil
	}

	f.TemplateBody = boilerplateTemplate

	return nil
}

const boilerplateTemplate = `/*
{{ if .Owner -}}
Copyright {{ .Year }} {{ .Owner }}.
{{- else -}}
Copyright {{ .Year }}.
{{- end }}
{{ index .Licenses .License }}*/`

var knownLicenses = map[string]string{
	"apache2":   apache2,
	"copyright": "",
}

const apache2 = `
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
`
