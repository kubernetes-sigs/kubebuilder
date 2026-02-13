/*
Copyright 2025 The Kubernetes Authors.

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

var _ machinery.Template = &HelmChartCI{}

// HelmChartCI scaffolds the GitHub Action for testing Helm charts
type HelmChartCI struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// Force if true allows overwriting the scaffolded file
	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *HelmChartCI) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(".github", "workflows", "test-chart.yml")
	}

	f.TemplateBody = testChartTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

const testChartTemplate = `name: Test Chart

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
          curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-$(go env GOARCH)
          chmod +x ./kind
          sudo mv ./kind /usr/local/bin/kind

      - name: Verify kind installation
        run: kind version

      - name: Create kind cluster
        run: kind create cluster

      - name: Prepare {{ .ProjectName }}
        run: |
          go mod tidy
          make docker-build IMG=controller:latest
          kind load docker-image controller:latest

      - name: Install Helm
        run: make install-helm

      - name: Lint Helm Chart
        run: |
          helm lint ./dist/chart

# TODO: Uncomment if cert-manager is enabled
#      - name: Install cert-manager via Helm (wait for readiness)
#        run: |
#          helm repo add jetstack https://charts.jetstack.io
#          helm repo update
#          helm install cert-manager jetstack/cert-manager \
#            --namespace cert-manager \
#            --create-namespace \
#            --set crds.enabled=true \
#            --wait \
#            --timeout 300s

# TODO: Uncomment if Prometheus is enabled
#      - name: Install Prometheus Operator CRDs
#        run: |
#          helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
#          helm repo update
#          helm install prometheus-crds prometheus-community/prometheus-operator-crds

      - name: Deploy manager via Helm
        run: |
          make helm-deploy IMG={{ .ProjectName }}:v0.1.0

      - name: Check Helm release status
        run: |
          make helm-status

      - name: Run Helm tests
        run: |
          helm test {{ .ProjectName }} --namespace {{ .ProjectName }}-system
`
