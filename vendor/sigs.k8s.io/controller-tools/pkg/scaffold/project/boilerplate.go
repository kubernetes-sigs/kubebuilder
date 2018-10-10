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

package project

import (
	"fmt"
	"path/filepath"
	"time"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &Boilerplate{}

// Boilerplate scaffolds a boilerplate header file.
type Boilerplate struct {
	input.Input

	// License is the License type to write
	License string

	// Owner is the copyright owner - e.g. "The Kubernetes Authors"
	Owner string

	// Year is the copyright year
	Year string
}

// GetInput implements input.File
func (c *Boilerplate) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("hack", "boilerplate.go.txt")
	}

	// Boilerplate given
	if len(c.Boilerplate) > 0 {
		c.TemplateBody = c.Boilerplate
		return c.Input, nil
	}

	// Pick a template boilerplate option
	if c.Year == "" {
		c.Year = fmt.Sprintf("%v", time.Now().Year())
	}
	switch c.License {
	case "", "apache2":
		c.TemplateBody = apache
	case "none":
		c.TemplateBody = none
	}
	return c.Input, nil
}

var apache = `/*
{{ if .Owner }}Copyright {{ .Year }} {{ .Owner }}.
{{ end }}
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

var none = `/*
{{ if .Owner }}Copyright {{ .Year }} {{ .Owner }}{{ end }}.
*/`
