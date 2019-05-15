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

var _ input.File = &InjectCAPatch{}

// InjectCAPatch scaffolds the InjectCAPatch file in manager folder.
type InjectCAPatch struct {
	input.Input
}

// GetInput implements input.File
func (c *InjectCAPatch) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "default", "webhookcainjection_patch.yaml")
	}
	c.TemplateBody = injectCAPatchTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var injectCAPatchTemplate = `# This patch add annotation to admission webhook config and
# the variables $(NAMESPACE) and $(CERTIFICATENAME) will be substituted by kustomize.  
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: mutating-webhook-configuration
  annotations:
    certmanager.k8s.io/inject-ca-from: $(CERTIFICATENAMESPACE)/$(CERTIFICATENAME)
---
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: validating-webhook-configuration
  annotations:
    certmanager.k8s.io/inject-ca-from: $(CERTIFICATENAMESPACE)/$(CERTIFICATENAME)
`
