/*
Copyright 2023 The Kubernetes Authors.

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
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &UnitTest{}

// UnitTest scaffolds a file that defines Golangci GitHub Actions
type UnitTest struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements file.Template
func (f *UnitTest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = ".github/workflows/unit-test.yml"
	}

	f.TemplateBody = golangciGitHubActionTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

//nolint:lll
const golangciGitHubActionTemplate = `name: Unit tests

# Trigger the workflow on pull requests and direct pushes to any branch
on:
  push:
  pull_request:

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    # Pull requests from the same repository won't trigger this checks as they were already triggered by the push
    if: (github.event_name == 'push' || github.event.pull_request.head.repo.full_name != github.repository)
    steps:
      - name: Clone the code
        uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '~1.20'
      # This step is needed as the following one tries to remove
      # kustomize for each test but has no permission to do so
      - name: Remove pre-installed kustomize
        run: sudo rm -f /usr/local/bin/kustomize
      - name: Perform the test
        run: make test
`
