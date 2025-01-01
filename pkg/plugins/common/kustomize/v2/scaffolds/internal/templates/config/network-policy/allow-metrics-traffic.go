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

var _ machinery.Template = &PolicyAllowMetrics{}

// PolicyAllowMetrics scaffolds a file that defines the NetworkPolicy
// to allow access to the metrics endpoint
type PolicyAllowMetrics struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *PolicyAllowMetrics) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "network-policy", "allow-metrics-traffic.yaml")
	}

	f.TemplateBody = metricsNetworkPolicyTemplate

	return nil
}

const metricsNetworkPolicyTemplate = `# This NetworkPolicy allows ingress traffic
# with Pods running on namespaces labeled with 'metrics: enabled'. Only Pods on those
# namespaces are able to gather data from the metrics endpoint.
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: allow-metrics-traffic
  namespace: system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: {{ .ProjectName }}
  policyTypes:
    - Ingress
  ingress:
    # This allows ingress traffic from any namespace with the label metrics: enabled
    - from:
      - namespaceSelector:
          matchLabels:
            metrics: enabled  # Only from namespaces with this label
      ports:
        - port: 8443
          protocol: TCP
`
