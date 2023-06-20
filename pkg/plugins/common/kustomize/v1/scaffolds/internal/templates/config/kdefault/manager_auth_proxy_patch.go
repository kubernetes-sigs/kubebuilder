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

package kdefault

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &ManagerAuthProxyPatch{}

// ManagerAuthProxyPatch scaffolds a file that defines the patch that enables prometheus metrics for the manager
type ManagerAuthProxyPatch struct {
	machinery.TemplateMixin
	machinery.ComponentConfigMixin
}

// SetTemplateDefaults implements file.Template
func (f *ManagerAuthProxyPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "manager_auth_proxy_patch.yaml")
	}

	f.TemplateBody = kustomizeAuthProxyPatchTemplate

	f.IfExistsAction = machinery.Error

	return nil
}

const kustomizeAuthProxyPatchTemplate = `# This patch inject a sidecar container which is a HTTP proxy for the
# controller manager, it performs RBAC authorization against the Kubernetes API using SubjectAccessReviews.
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
spec:
  template:
    spec:
      containers:
      - name: kube-rbac-proxy
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - "ALL"
        image: gcr.io/kubebuilder/kube-rbac-proxy:v0.13.1
        args:
        - "--secure-listen-address=0.0.0.0:8443"
        - "--upstream=http://127.0.0.1:8080/"
        - "--logtostderr=true"
        - "--v=0"
        ports:
        - containerPort: 8443
          protocol: TCP
          name: https
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 5m
            memory: 64Mi
{{- if not .ComponentConfig }}
      - name: manager
        args:
        - "--health-probe-bind-address=:8081"
        - "--metrics-bind-address=127.0.0.1:8080"
        - "--leader-elect"
{{- end }}
`
