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

var _ machinery.Template = &HelmChartCI{}

// HelmChartCI scaffolds the GitHub Action for testing Helm charts
type HelmChartCI struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *HelmChartCI) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(".github", "workflows", "test-chart.yml")
	}

	f.TemplateBody = testChartTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

//nolint:lll
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
          curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
          chmod +x ./kind
          sudo mv ./kind /usr/local/bin/kind

      - name: Verify kind installation
        run: kind version

      - name: Create kind cluster
        run: kind create cluster

      - name: Prepare {{ .ProjectName }}
        run: |
          go mod tidy
          make docker-build IMG={{ .ProjectName }}:v0.1.0
          kind load docker-image {{ .ProjectName }}:v0.1.0

      - name: Install Helm
        run: |
          curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

      - name: Verify Helm installation
        run: helm version

      - name: Lint Helm Chart
        run: |
          helm lint ./dist/chart

# TODO: Uncomment if cert-manager is enabled
#      - name: Install cert-manager via Helm
#        run: |
#          helm repo add jetstack https://charts.jetstack.io
#          helm repo update
#          helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set installCRDs=true
#
#      - name: Wait for cert-manager to be ready
#        run: |
#          kubectl wait --namespace cert-manager --for=condition=available --timeout=300s deployment/cert-manager
#          kubectl wait --namespace cert-manager --for=condition=available --timeout=300s deployment/cert-manager-cainjector
#          kubectl wait --namespace cert-manager --for=condition=available --timeout=300s deployment/cert-manager-webhook

# TODO: Uncomment if Prometheus is enabled
#      - name: Install Prometheus Operator CRDs
#        run: |
#          helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
#          helm repo update
#          helm install prometheus-crds prometheus-community/prometheus-operator-crds
#
#      - name: Install Prometheus via Helm
#        run: |
#          helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
#          helm repo update
#          helm install prometheus prometheus-community/prometheus --namespace monitoring --create-namespace
#
#      - name: Wait for Prometheus to be ready
#        run: |
#          kubectl wait --namespace monitoring --for=condition=available --timeout=300s deployment/prometheus-server

      - name: Install Helm chart for project
        run: |
          helm install my-release ./dist/chart --create-namespace --namespace {{ .ProjectName }}-system

      - name: Check Helm release status
        run: |
          helm status my-release --namespace {{ .ProjectName }}-system

# TODO: Uncomment if prometheus.enabled is set to true to confirm that the ServiceMonitor gets created
#      - name: Check Presence of ServiceMonitor
#        run: |
#          kubectl wait --namespace {{ .ProjectName }}-system --for=jsonpath='{.kind}'=ServiceMonitor servicemonitor/{{ .ProjectName }}-controller-manager-metrics-monitor
`
