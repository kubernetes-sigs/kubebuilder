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

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Service{}

// Service scaffolds the Service file in manager folder.
type Service struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Service) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "webhook", "service.yaml")
	}

	f.TemplateBody = serviceTemplate

	f.IfExistsAction = file.Error

	return nil
}

const serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: webhook-service
  namespace: system
spec:
  ports:
    - port: 443
      targetPort: 9443
  selector:
    control-plane: controller-manager
`
