/*
Copyright 2024 The Kubernetes Authors.

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

package github

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &TestCi{}

// LintCi scaffolds the GitHub Action to lint the project
type LintCi struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin

	// golangci-lint version to use in the project
	GolangciLintVersion string
}

// SetTemplateDefaults implements machinery.Template
func (f *LintCi) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(".github", "workflows", "lint.yml")
	}

	f.TemplateBody = lintCiTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const lintCiTemplate = `name: Lint

on:
  push:
  pull_request:

permissions: {}

jobs:
  lint:
    permissions:
      contents: read
    name: Run on Ubuntu
    runs-on: ubuntu-latest
    steps:
      - name: Clone the code
        uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
        with:
          persist-credentials: false

      - name: Setup Go
        uses: actions/setup-go@4b73464bb391d4059bd26b0524d20df3927bd417 # v6.3.0
        with:
          go-version-file: go.mod

      - name: Check linter configuration
        run: make lint-config
      - name: Run linter
        run: make lint
`
