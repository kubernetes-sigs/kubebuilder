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

var _ machinery.Template = &E2eTestCi{}

// E2eTestCi scaffolds the GitHub Action to call make test-e2e
type E2eTestCi struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *E2eTestCi) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(".github", "workflows", "test-e2e.yml")
	}

	f.TemplateBody = e2eTestCiTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const e2eTestCiTemplate = `name: E2E Tests

on:
  push:
  pull_request:

jobs:
  test-e2e:
    name: Run on Ubuntu
    runs-on: ubuntu-latest
    steps:
      - name: Clone the code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Install the latest version of kind
        run: |
          curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
          chmod +x ./kind
          sudo mv ./kind /usr/local/bin/kind

      - name: Verify kind installation
        run: kind version

      - name: Create kind cluster
        run: kind create cluster

      - name: Running Test e2e
        run: |
          go mod tidy
          make test-e2e
`
