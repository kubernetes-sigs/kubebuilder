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

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Kustomization{}

// Kustomization scaffolds a file that defines the kustomization scheme for the prometheus folder
type Kustomization struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "prometheus", "kustomization.yaml")
	}

	f.TemplateBody = kustomizationTemplate

	return nil
}

const kustomizationTemplate = `resources:
- monitor.yaml

# [PROMETHEUS-WITH-CERTS] The following patch configures the ServiceMonitor in ../prometheus
# to securely reference certificates created and managed by cert-manager.
# Additionally, ensure that you uncomment the [METRICS WITH CERTMANAGER] patch under config/default/kustomization.yaml
# to mount the "metrics-server-cert" secret in the Manager Deployment.
#patches:
#  - path: monitor_tls_patch.yaml
#    target:
#      kind: ServiceMonitor
`
