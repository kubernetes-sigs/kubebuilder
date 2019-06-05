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

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &KustomizePrometheusMetricsPatch{}

// KustomizePrometheusMetricsPatch scaffolds the patch file for enabling
// prometheus metrics for manager Pod.
type KustomizePrometheusMetricsPatch struct {
	input.Input
}

// GetInput implements input.File
func (c *KustomizePrometheusMetricsPatch) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "default", "manager_prometheus_metrics_patch.yaml")
	}
	c.TemplateBody = kustomizePrometheusMetricsPatchTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var kustomizePrometheusMetricsPatchTemplate = `# This patch enables Prometheus scraping for the manager pod.
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
