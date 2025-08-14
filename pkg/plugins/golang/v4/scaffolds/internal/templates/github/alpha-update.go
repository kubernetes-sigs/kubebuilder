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

var _ machinery.Template = &AlphaUpdateCi{}

// AlphaUpdateCi scaffolds the GitHub Action to run kubebuilder alpha update
type AlphaUpdateCi struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *AlphaUpdateCi) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(".github", "workflows", "alpha-update.yml")
	}

	f.TemplateBody = alphaUpdateCiTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const alphaUpdateCiTemplate = `name: Alpha Update

permissions:
  contents: write
  issues: write
  pull-requests: write

on:
  workflow_dispatch:
  schedule:
    - cron: "0 0 * * 2" # Every Tuesday at 00:00 UTC

jobs:
  alpha-update:
    runs-on: ubuntu-latest
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

    steps:
    - name: Checkout repository
      uses: actions/checkout@v4
      with:
        token: ${{ secrets.GITHUB_TOKEN }}
        fetch-depth: 0
    
    - name: Configure Git
      run: |
        git config --global user.name "github-actions[bot]"
        git config --global user.email "github-actions[bot]@users.noreply.github.com"
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: stable

    - name: Install Kubebuilder
      run: |
        curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"
        chmod +x kubebuilder
        sudo mv kubebuilder /usr/local/bin/
        kubebuilder version

    - name: Run kubebuilder alpha update
      run: |
        kubebuilder alpha update \
          --force \
          --squash \
          --preserve-path .github/workflows \
          --open-gh-issue
`