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

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Certificate{}

// Certificate scaffolds a file that defines the issuer CR and the certificate CR
type Certificate struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Certificate) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "certmanager", "certificate-webhook.yaml")
	}

	f.TemplateBody = certManagerTemplate

	// If file exists (ex. because a webhook was already created), skip creation.
	f.IfExistsAction = machinery.SkipFile

	return nil
}

const certManagerTemplate = `# The following manifests contain a self-signed issuer CR and a certificate CR.
# More document can be found at https://docs.cert-manager.io
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  labels:
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: serving-cert  # this name should match the one appeared in kustomizeconfig.yaml
  namespace: system
spec:
  # SERVICE_NAME and SERVICE_NAMESPACE will be substituted by kustomize
  # replacements in the config/default/kustomization.yaml file.
  dnsNames:
  - SERVICE_NAME.SERVICE_NAMESPACE.svc
  - SERVICE_NAME.SERVICE_NAMESPACE.svc.cluster.local
  issuerRef:
    kind: Issuer
    name: selfsigned-issuer
  secretName: webhook-server-cert
`
