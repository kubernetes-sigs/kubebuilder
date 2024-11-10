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

// SetTemplateDefaults implements file.Template
func (f *CertManagerMetricsPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "certmanager_metrics_manager_patch.yaml")
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

const metricsManagerPatchTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
  labels:
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
spec:
  template:
    spec:
      containers:
      - name: manager
        volumeMounts:
        - mountPath: /tmp/k8s-metrics-server/metrics-certs
          name: metrics-certs
          readOnly: true
      volumes:
      - name: metrics-certs
        secret:
          secretName: metrics-server-cert
`
