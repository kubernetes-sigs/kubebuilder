/*
Copyright 2026 The Kubernetes Authors.

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

package kustomize

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser deployment selection (integration)", func() {
	var parser *Parser

	BeforeEach(func() {
		parser = NewParser("")
	})

	It("selects the single deployment and leaves ExtraDeployments empty", func() {
		yaml := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: only-deployment
`
		result, err := parser.ParseFromReader(strings.NewReader(yaml))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Deployment).NotTo(BeNil())
		Expect(result.Deployment.GetName()).To(Equal("only-deployment"))
		Expect(result.ExtraDeployments).To(BeEmpty())
	})

	It("selects the deployment with label control-plane: controller-manager", func() {
		yaml := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: other-deployment
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: manager-deployment
  labels:
    control-plane: controller-manager
`
		result, err := parser.ParseFromReader(strings.NewReader(yaml))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Deployment).NotTo(BeNil())
		Expect(result.Deployment.GetName()).To(Equal("manager-deployment"))
	})

	It("selects the deployment with pod-template annotation when no label matches", func() {
		yaml := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: other-deployment
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: annotated-deployment
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
`
		result, err := parser.ParseFromReader(strings.NewReader(yaml))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Deployment).NotTo(BeNil())
		Expect(result.Deployment.GetName()).To(Equal("annotated-deployment"))
	})

	It("selects the deployment with a container named manager when no label or annotation matches", func() {
		yaml := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: other-deployment
spec:
  template:
    spec:
      containers:
      - name: sidecar
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: named-container-deployment
spec:
  template:
    spec:
      containers:
      - name: manager
`
		result, err := parser.ParseFromReader(strings.NewReader(yaml))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Deployment).NotTo(BeNil())
		Expect(result.Deployment.GetName()).To(Equal("named-container-deployment"))
	})

	It("leaves Deployment nil and all in ExtraDeployments when multiple deployments have no signals", func() {
		yaml := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: first-deployment
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: second-deployment
`
		result, err := parser.ParseFromReader(strings.NewReader(yaml))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Deployment).To(BeNil())
		Expect(result.ExtraDeployments).To(HaveLen(2))
	})

	// The label control-plane: controller-manager is preserved across kustomize namePrefix transforms.
	It("identifies the manager when kustomize namePrefix has been applied", func() {
		yaml := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: project-v4-controller-manager
  namespace: project-v4-system
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: project-v4
    app.kubernetes.io/managed-by: kustomize
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
    spec:
      containers:
      - name: manager
        image: controller:latest
`
		result, err := parser.ParseFromReader(strings.NewReader(yaml))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Deployment).NotTo(BeNil())
		Expect(result.Deployment.GetName()).To(Equal("project-v4-controller-manager"))
		Expect(result.ExtraDeployments).To(BeEmpty())
	})

	It("places non-manager deployments in ExtraDeployments", func() {
		yaml := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: project-v4-controller-manager
  labels:
    control-plane: controller-manager
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: project-v4-some-operator
`
		result, err := parser.ParseFromReader(strings.NewReader(yaml))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Deployment).NotTo(BeNil())
		Expect(result.Deployment.GetName()).To(Equal("project-v4-controller-manager"))
		Expect(result.ExtraDeployments).To(HaveLen(1))
		Expect(result.ExtraDeployments[0].GetName()).To(Equal("project-v4-some-operator"))
	})
})
