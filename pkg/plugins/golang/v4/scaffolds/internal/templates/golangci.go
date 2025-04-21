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
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Golangci{}

// Golangci scaffolds a file which define Golangci rules
type Golangci struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Golangci) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = ".golangci.yml"
	}

	f.TemplateBody = golangciTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const golangciTemplate = `version: "2"
run:
  allow-parallel-runners: true
linters:
  default: none
  enable:
    - copyloopvar
    - dupl
    - errcheck
    - ginkgolinter
    - goconst
    - gocyclo
    - govet
    - ineffassign
    - lll
    - misspell
    - nakedret
    - prealloc
    - revive
    - staticcheck
    - unconvert
    - unparam
    - unused
  settings:
    revive:
      rules:
        - name: comment-spacings
        - name: import-shadowing
  exclusions:
    generated: lax
    rules:
      - linters:
          - lll
        path: api/*
      - linters:
          - dupl
          - lll
        path: internal/*
    paths:
      - third_party$
      - builtin$
      - examples$
formatters:
  enable:
    - gofmt
    - goimports
  exclusions:
    generated: lax
    paths:
      - third_party$
      - builtin$
      - examples$
`
