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

package networkpolicy

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &PolicyAllowWebhooks{}

// PolicyAllowWebhooks scaffolds a NetworkPolicy that allows traffic to the webhook server.
type PolicyAllowWebhooks struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *PolicyAllowWebhooks) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "network-policy", "allow-webhook-traffic.yaml")
	}

	f.TemplateBody = webhooksNetworkPolicyTemplate

	return nil
}

const webhooksNetworkPolicyTemplate = `# Allow ingress to the webhook server's pod port (9443) from all sources.
# The webhook Service exposes port 443 and forwards traffic to the pod port.
# The Kubernetes API server uses the Service to reach admission and CRD conversion webhooks.
# To restrict admission requests by namespace, configure namespaceSelector in the
# MutatingWebhookConfiguration or ValidatingWebhookConfiguration.
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: allow-webhook-traffic
  namespace: system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: {{ .ProjectName }}
  policyTypes:
    - Ingress
  ingress:
    - ports:
        - port: 9443
          protocol: TCP
`
