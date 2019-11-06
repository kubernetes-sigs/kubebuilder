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

package v1

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &AuthProxyService{}

// AuthProxyService scaffolds the config/rbac/auth_proxy_service.yaml file
type AuthProxyService struct {
	input.Input
}

// GetInput implements input.File
func (f *AuthProxyService) GetInput() (input.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "auth_proxy_service.yaml")
	}
	f.TemplateBody = AuthProxyServiceTemplate
	return f.Input, nil
}

const AuthProxyServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  annotations:
    prometheus.io/port: "8443"
    prometheus.io/scheme: https
    prometheus.io/scrape: "true"
  labels:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
  name: controller-manager-metrics-service
  namespace: system
spec:
  ports:
  - name: https
    port: 8443
    targetPort: https
  selector:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
`
