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

package kdefault

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &CertManagerMetricsPatch{}

// CertManagerMetricsPatch scaffolds a file that defines the patch that enables webhooks on the manager
type CertManagerMetricsPatch struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *CertManagerMetricsPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "cert_metrics_manager_patch.yaml")
	}

	f.TemplateBody = metricsManagerPatchTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		// If file exists (ex. because a webhook was already created), skip creation.
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

// nolint:lll
const metricsManagerPatchTemplate = `# This patch adds the args and volumes to allow the manager to use the metrics-server certs
# Ensure the volumeMounts field exists by creating it if missing
- op: add
  path: /spec/template/spec/containers/0/volumeMounts
  value: []
# Add the volume mount for the serving certificates
- op: add
  path: /spec/template/spec/containers/0/volumeMounts/-
  value:
    mountPath: /tmp/k8s-metrics-server/serving-certs
    name: metrics-certs
    readOnly: true
# Add the cert-dir argument
- op: add
  path: /spec/template/spec/containers/0/args/-
  value: --cert-dir=/tmp/k8s-metrics-server/serving-certs
# Ensure the volumes field exists by creating it if missing
- op: add
  path: /spec/template/spec/volumes
  value: []
# Add the volume for the serving certificates
- op: add
  path: /spec/template/spec/volumes/-
  value:
    name: metrics-certs
    secret:
      secretName: metrics-server-cert
      optional: false
      items:
        - key: ca.crt
          path: ca.crt
        - key: tls.crt
          path: tls.crt
        - key: tls.key
          path: tls.key
`
