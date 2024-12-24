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

package manager

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Config{}

// Config scaffolds a file that defines the namespace and the manager deployment
type Config struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// Image is controller manager image name
	Image string
}

// SetTemplateDefaults implements machinery.Template
func (f *Config) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "manager", "manager.yaml")
	}

	f.TemplateBody = configTemplate

	return nil
}

const configTemplate = `apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: {{ .ProjectName }}
  replicas: 1
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager
        app.kubernetes.io/name: {{ .ProjectName }}
    spec:
      # TODO(user): Uncomment the following code to configure the nodeAffinity expression
      # according to the platforms which are supported by your solution.
      # It is considered best practice to support multiple architectures. You can
      # build your manager image using the makefile target docker-buildx.
      # affinity:
      #   nodeAffinity:
      #     requiredDuringSchedulingIgnoredDuringExecution:
      #       nodeSelectorTerms:
      #         - matchExpressions:
      #           - key: kubernetes.io/arch
      #             operator: In
      #             values:
      #               - amd64
      #               - arm64
      #               - ppc64le
      #               - s390x
      #           - key: kubernetes.io/os
      #             operator: In
      #             values:
      #               - linux
      securityContext:
        # Projects are configured by default to adhere to the "restricted" Pod Security Standards.
        # This ensures that deployments meet the highest security requirements for Kubernetes.
        # For more details, see: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      containers:
      - command:
        - /manager
        args:
          - --leader-elect
          - --health-probe-bind-address=:8081
        image: {{ .Image }}
        name: manager
        ports: []
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - "ALL"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
        # TODO(user): Configure the resources accordingly based on the project requirements.
        # More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
        volumeMounts: []
      volumes: []
      serviceAccountName: controller-manager
      terminationGracePeriodSeconds: 10
`
