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

package manager

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &Deploy{}

// Deploy scaffolds yaml deploy and configurations for the manager.
type Deploy struct {
	input.Input
	// Image is controller manager image name
	Image string
}

// GetInput implements input.File
func (c *Deploy) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("deploy", "manager", "manager.yaml")
	}
	c.TemplateBody = deployTemplate
	return c.Input, nil
}

var deployTemplate = `apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
  labels:
    control-plane: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  replicas: 1
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - command:
        - /manager
        args:
        - --enable-leader-election
        image: {{ .Image }}
        name: manager
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
      terminationGracePeriodSeconds: 10
`
