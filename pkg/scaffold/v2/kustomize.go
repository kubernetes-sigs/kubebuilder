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

package v2

import (
	"os"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &Kustomize{}

// Kustomize scaffolds the Kustomization file for the default overlay
type Kustomize struct {
	input.Input

	// Prefix to use for name prefix customization
	Prefix string
}

// GetInput implements input.File
func (c *Kustomize) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("deploy", "default", "kustomization.yaml")
	}
	if c.Prefix == "" {
		// use directory name as prefix
		dir, err := os.Getwd()
		if err != nil {
			return input.Input{}, err
		}
		c.Prefix = filepath.Base(dir)
	}
	c.TemplateBody = kustomizeTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var kustomizeTemplate = `# Adds namespace to all resources.
namespace: {{.Prefix}}-system

# Value of this field is prepended to the
# names of all resources, e.g. a deployment named
# "wordpress" becomes "alices-wordpress".
# Note that it should also match with the prefix (text before '-') of the namespace
# field above.
namePrefix: {{.Prefix}}-

# Labels to add to all resources and selectors.
#commonLabels:
#  someName: someValue

bases:
- ../crd
- ../rbac
- ../manager
# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix including the one in crd/kustomization.yaml
#- ../webhook
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'. 'WEBHOOK' components are required.
#- ../certmanager

patchesStrategicMerge:
  # Protect the /metrics endpoint by putting it behind auth.
  # Only one of manager_auth_proxy_patch.yaml and
  # manager_prometheus_metrics_patch.yaml should be enabled.
- manager_auth_proxy_patch.yaml
  # If you want your controller-manager to expose the /metrics
  # endpoint w/o any authn/z, uncomment the following line and
  # comment manager_auth_proxy_patch.yaml.
  # Only one of manager_auth_proxy_patch.yaml and
  # manager_prometheus_metrics_patch.yaml should be enabled.
#- manager_prometheus_metrics_patch.yaml

# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix including the one in crd/kustomization.yaml
#- manager_webhook_patch.yaml

# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'.
# Uncomment 'CERTMANAGER' sections in crd/kustomization.yaml to enable the CA injection in the admission webhooks.
# 'CERTMANAGER' needs to be enabled to use ca injection
#- webhookcainjection_patch.yaml

# the following config is for teaching kustomize how to do var substitution
vars:
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER' prefix.
#- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
#  objref:
#    kind: Certificate
#    group: certmanager.k8s.io
#    version: v1alpha1
#    name: serving-cert # this name should match the one in certificate.yaml
#  fieldref:
#    fieldpath: metadata.namespace
#- name: CERTIFICATE_NAME
#  objref:
#    kind: Certificate
#    group: certmanager.k8s.io
#    version: v1alpha1
#    name: serving-cert # this name should match the one in certificate.yaml
#- name: SERVICE_NAMESPACE # namespace of the service
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service
#  fieldref:
#    fieldpath: metadata.namespace
#- name: SERVICE_NAME
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service
`
