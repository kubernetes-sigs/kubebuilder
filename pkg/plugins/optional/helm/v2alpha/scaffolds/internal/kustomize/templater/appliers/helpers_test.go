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

package appliers

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sigsyaml "sigs.k8s.io/yaml"
)

var _ = Describe("FindManagerContainerRange", func() {
	It("should find manager when name is the first field", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
        args:
        - --leader-elect`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(4))
		Expect(end).To(Equal(len(lines) - 1))
		Expect(lines[start]).To(ContainSubstring("- name: manager"))
	})

	It("should find manager when fields are alphabetically sorted (yaml.Marshal)", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - args:
        - --leader-elect
        env:
        - name: BUSYBOX_IMAGE
          value: busybox:1.36.1
        image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(4))
		Expect(end).To(Equal(len(lines) - 1))
		Expect(lines[start]).To(ContainSubstring("- args:"))
	})

	It("should find manager at index 1 when sidecar is first (name-first fields)", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - name: sidecar
        image: sidecar:v1
        resources:
          limits:
            cpu: 100m
      - name: manager
        image: controller:latest
        args:
        - --leader-elect`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(9))
		Expect(end).To(Equal(len(lines) - 1))
		Expect(lines[start]).To(ContainSubstring("- name: manager"))
	})

	It("should find manager at index 1 when fields are alphabetically sorted", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - image: sidecar:v1
        name: sidecar
        resources:
          limits:
            cpu: 100m
      - args:
        - --leader-elect
        image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(9))
		Expect(end).To(Equal(len(lines) - 1))
		Expect(lines[start]).To(ContainSubstring("- args:"))
		Expect(lines[end]).To(ContainSubstring("cpu: 500m"))
	})

	It("should scope range to manager only, excluding sidecar lines", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - image: sidecar:v1
        name: sidecar
        resources:
          limits:
            cpu: 100m
            memory: 64Mi
      - args:
        - --leader-elect
        image: controller:latest
        name: manager`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")

		rangeContent := strings.Join(lines[start:end+1], "\n")
		Expect(rangeContent).To(ContainSubstring("name: manager"))
		Expect(rangeContent).To(ContainSubstring("controller:latest"))
		Expect(rangeContent).NotTo(ContainSubstring("sidecar"))
	})

	It("should not match env var named 'manager' as the container name", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - env:
        - name: manager
          value: "true"
        image: sidecar:v1
        name: sidecar
      - image: controller:latest
        name: manager`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(9))
		Expect(end).To(Equal(len(lines) - 1))
		Expect(lines[start]).To(ContainSubstring("- image:"))
	})

	It("should handle nested list fields without false container boundaries", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - args:
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
        env:
        - name: POD_NAMESPACE
          value: default
        - name: LOG_LEVEL
          value: info
        image: controller:latest
        name: manager`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(4))
		Expect(end).To(Equal(len(lines) - 1))

		rangeContent := strings.Join(lines[start:end+1], "\n")
		Expect(rangeContent).To(ContainSubstring("--metrics-bind-address"))
		Expect(rangeContent).To(ContainSubstring("POD_NAMESPACE"))
		Expect(rangeContent).To(ContainSubstring("name: manager"))
	})

	It("should use default-container annotation for custom container name", func() {
		yaml := `spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: controller-test
    spec:
      containers:
      - image: controller:latest
        name: controller-test`

		start, end := FindManagerContainerRange(yaml)
		Expect(start).To(Equal(7))
		Expect(end).To(Equal(8))
	})

	It("should return (-1, -1) when no containers section exists", func() {
		yaml := `spec:
  template:
    spec:
      serviceAccountName: test`

		start, end := FindManagerContainerRange(yaml)
		Expect(start).To(Equal(-1))
		Expect(end).To(Equal(-1))
	})

	It("should return (-1, -1) when manager container is not present", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - name: sidecar
        image: sidecar:v1
      - name: proxy
        image: proxy:v2`

		start, end := FindManagerContainerRange(yaml)
		Expect(start).To(Equal(-1))
		Expect(end).To(Equal(-1))
	})

	It("should stop the range at the next container", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m
      - image: sidecar:v1
        name: sidecar`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(4))
		Expect(end).To(Equal(8))

		rangeContent := strings.Join(lines[start:end+1], "\n")
		Expect(rangeContent).NotTo(ContainSubstring("sidecar"))
	})

	It("should work on real yaml.Marshal output with sidecar before manager", func() {
		deployment := map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name": "test-controller-manager",
			},
			"spec": map[string]any{
				"template": map[string]any{
					"metadata": map[string]any{
						"annotations": map[string]any{
							"kubectl.kubernetes.io/default-container": "manager",
						},
					},
					"spec": map[string]any{
						"containers": []any{
							map[string]any{
								"name":  "sidecar",
								"image": "sidecar:v1",
								"env": []any{
									map[string]any{"name": "SIDECAR_MODE", "value": "active"},
								},
								"resources": map[string]any{
									"limits": map[string]any{"cpu": "100m", "memory": "64Mi"},
								},
							},
							map[string]any{
								"name":  "manager",
								"image": "controller:latest",
								"args":  []any{"--leader-elect", "--health-probe-bind-address=:8081"},
								"env": []any{
									map[string]any{"name": "MANAGER_ENV", "value": "production"},
								},
								"resources": map[string]any{
									"limits":   map[string]any{"cpu": "500m", "memory": "128Mi"},
									"requests": map[string]any{"cpu": "10m", "memory": "64Mi"},
								},
							},
						},
					},
				},
			},
		}

		yamlBytes, err := sigsyaml.Marshal(deployment)
		Expect(err).NotTo(HaveOccurred())
		yamlContent := string(yamlBytes)

		start, end := FindManagerContainerRange(yamlContent)
		Expect(start).To(BeNumerically(">=", 0), "should find manager in yaml.Marshal output")

		lines := strings.Split(yamlContent, "\n")
		rangeContent := strings.Join(lines[start:end+1], "\n")

		Expect(rangeContent).To(ContainSubstring("name: manager"))
		Expect(rangeContent).To(ContainSubstring("controller:latest"))
		Expect(rangeContent).To(ContainSubstring("--leader-elect"))
		Expect(rangeContent).NotTo(ContainSubstring("name: sidecar"))
		Expect(rangeContent).NotTo(ContainSubstring("sidecar:v1"))
		Expect(rangeContent).NotTo(ContainSubstring("SIDECAR_MODE"))
	})
})
