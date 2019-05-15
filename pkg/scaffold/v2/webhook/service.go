/*
Copyright 2019 The Kubernetes Authors.

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

package webhook

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &Service{}

// Service scaffolds the Service file in manager folder.
type Service struct {
	input.Input
}

// GetInput implements input.File
func (c *Service) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "webhook", "service.yaml")
	}
	c.TemplateBody = ServiceTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var ServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: webhook-service
  namespace: system
spec:
  ports:
    - port: 443
      targetPort: 443
`
