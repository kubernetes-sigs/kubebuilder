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

package certmanager

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Certificate{}

// Certificate scaffolds a file that defines the issuer CR and the certificate CR
type Certificate struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements file.Template
func (f *Certificate) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "certmanager", "certificate.yaml")
	}

	f.TemplateBody = certManagerTemplate

	// If file exists (ex. because a webhook was already created), skip creation.
	f.IfExistsAction = machinery.SkipFile

	return nil
}

const certManagerTemplate = `# The following manifests contain a self-signed issuer CR and a certificate CR.
# More document can be found at https://docs.cert-manager.io
# WARNING: Targets CertManager v1.0. Check https://cert-manager.io/docs/installation/upgrading/ for breaking changes.
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned-issuer
  namespace: system
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: serving-cert  # this name should match the one appeared in kustomizeconfig.yaml
  namespace: system
spec:
  # $(SERVICE_NAME) and $(SERVICE_NAMESPACE) will be substituted by kustomize
  dnsNames:
  - $(SERVICE_NAME).$(SERVICE_NAMESPACE).svc
  - $(SERVICE_NAME).$(SERVICE_NAMESPACE).svc.cluster.local
  issuerRef:
    kind: Issuer
    name: selfsigned-issuer
  secretName: webhook-server-cert # this secret will not be prefixed, since it's not managed by kustomize
`
