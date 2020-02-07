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

package metricsauth

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &KustomizePrometheusMetricsPatch{}

// KustomizePrometheusMetricsPatch scaffolds the patch file for enabling
// prometheus metrics for manager Pod.
type KustomizePrometheusMetricsPatch struct {
	file.Input
}

// GetInput implements input.Template
func (f *KustomizePrometheusMetricsPatch) GetInput() (file.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "manager_prometheus_metrics_patch.yaml")
	}
	f.TemplateBody = kustomizePrometheusMetricsPatchTemplate
	f.IfExistsAction = file.Error
	return f.Input, nil
}

const kustomizePrometheusMetricsPatchTemplate = `# This patch enables Prometheus scraping for the manager pod.
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: controller-manager
  namespace: system
spec:
  template:
    metadata:
      annotations:
        prometheus.io/scrape: 'true'
    spec:
      containers:
      # Expose the prometheus metrics on default port
      - name: manager
        ports:
        - containerPort: 8080
          name: metrics
          protocol: TCP
`
