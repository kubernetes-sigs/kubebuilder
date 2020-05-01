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

package templates

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &ManagerWebhookPatch{}

// ManagerWebhookPatch scaffolds a ManagerWebhookPatch for a Resource
type ManagerWebhookPatch struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *ManagerWebhookPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "manager_webhook_patch.yaml")
	}

	f.TemplateBody = managerWebhookPatchTemplate

	return nil
}

const managerWebhookPatchTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
spec:
  template:
    spec:
      containers:
      - name: manager
        ports:
        - containerPort: 9443
          name: webhook-server
          protocol: TCP
        volumeMounts:
        - mountPath: /tmp/k8s-webhook-server/serving-certs
          name: cert
          readOnly: true
      volumes:
      - name: cert
        secret:
          defaultMode: 420
          secretName: webhook-server-cert
`
