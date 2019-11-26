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

var _ input.File = &KustomizeAuthProxyPatch{}

// KustomizeAuthProxyPatch scaffolds the patch file for enabling
// prometheus metrics for manager Pod.
type KustomizeAuthProxyPatch struct {
	input.Input
}

// GetInput implements input.File
func (c *KustomizeAuthProxyPatch) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "default", "manager_auth_proxy_patch.yaml")
	}
	c.TemplateBody = kustomizeAuthProxyPatchTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

const kustomizeAuthProxyPatchTemplate = `# This patch inject a sidecar container which is a HTTP proxy for the controller manager,
# it performs RBAC authorization against the Kubernetes API using SubjectAccessReviews.
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
        image: gcr.io/kubebuilder/kube-rbac-proxy:v0.4.1
        args:
        - "--secure-listen-address=0.0.0.0:8443"
        - "--upstream=http://127.0.0.1:8080/"
        - "--logtostderr=true"
        - "--v=10"
        ports:
        - containerPort: 8443
          name: https
      - name: manager
        args:
        - "--metrics-addr=127.0.0.1:8080"
        - "--enable-leader-election"
`
