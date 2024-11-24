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

// TestCi scaffolds the GitHub Action to call make test
type TestCi struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *TestCi) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(".github", "workflows", "test.yml")
	}

	f.TemplateBody = testCiTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const testCiTemplate = `name: Tests

on:
  push:
  pull_request:

jobs:
  test:
    name: Run on Ubuntu
    runs-on: ubuntu-latest
    steps:
      - name: Clone the code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Running Tests
        run: |
          go mod tidy
          make test
`
