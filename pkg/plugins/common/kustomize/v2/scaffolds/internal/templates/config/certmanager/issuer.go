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

package certmanager

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Issuer{}

// Issuer scaffolds a file that defines the self-signed Issuer CR
type Issuer struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Issuer) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "certmanager", "issuer.yaml")
	}

	f.TemplateBody = issuerTemplate

	// If file exists, skip creation.
	f.IfExistsAction = machinery.SkipFile

	return nil
}

const issuerTemplate = `# The following manifest contains a self-signed issuer CR.
# More information can be found at https://docs.cert-manager.io
# WARNING: Targets CertManager v1.0. Check https://cert-manager.io/docs/installation/upgrading/ for breaking changes.
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  labels:
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: selfsigned-issuer
  namespace: system
spec:
  selfSigned: {}
`
