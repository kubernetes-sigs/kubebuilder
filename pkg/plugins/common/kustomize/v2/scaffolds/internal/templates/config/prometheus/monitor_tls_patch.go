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

package prometheus

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &ServiceMonitorPatch{}

// ServiceMonitorPatch scaffolds a file that defines the patch for the ServiceMonitor
// to use cert-manager managed certificates for secure TLS configuration.
type ServiceMonitorPatch struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *ServiceMonitorPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "prometheus", "monitor_tls_patch.yaml")
	}

	f.TemplateBody = serviceMonitorPatchTemplate

	return nil
}

const serviceMonitorPatchTemplate = `# Patch for Prometheus ServiceMonitor to enable secure TLS configuration
# using certificates managed by cert-manager
- op: replace
  path: /spec/endpoints/0/tlsConfig
  value:
    # SERVICE_NAME and SERVICE_NAMESPACE will be substituted by kustomize
    serverName: SERVICE_NAME.SERVICE_NAMESPACE.svc
    insecureSkipVerify: false
    ca:
      secret:
        name: metrics-server-cert
        key: ca.crt
    cert:
      secret:
        name: metrics-server-cert
        key: tls.crt
    keySecret:
      name: metrics-server-cert
      key: tls.key
`
