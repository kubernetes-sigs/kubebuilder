/*
Copyright 2020 The Kubernetes Authors.

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

package prometheus

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &PrometheusServiceMonitor{}

// PrometheusMetricsService scaffolds an issuer CR and a certificate CR
type MonitorRoleBinding struct {
	input.Input
}

// GetInput implements input.File
func (f *MonitorRoleBinding) GetInput() (input.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("config", "prometheus", "monitor_binding_role.yaml")
	}
	f.TemplateBody = monitorBindingRoleTemplate
	return f.Input, nil
}

const monitorBindingRoleTemplate = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: metrics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: metrics
subjects:
- kind: ServiceAccount
  name: default
  namespace: system 
`
