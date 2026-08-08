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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	sigsyaml "sigs.k8s.io/yaml"
)

const (
	nameKey       = "name"
	cpuKey        = "cpu"
	memoryKey     = "memory"
	metadataKey   = "metadata"
	specKey       = k8sObjectSpecField
	containersKey = "containers"
	imageKey      = "image"
	managerVal    = "manager"
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

	It("should stop at sibling fields like volumes:", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m
      volumes:
      - name: data
        emptyDir: {}
      serviceAccountName: controller-manager`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(4))
		Expect(end).To(Equal(8))

		rangeContent := strings.Join(lines[start:end+1], "\n")
		Expect(rangeContent).To(ContainSubstring("name: manager"))
		Expect(rangeContent).NotTo(ContainSubstring("volumes:"))
		Expect(rangeContent).NotTo(ContainSubstring("serviceAccountName"))
	})

	It("should not match a volume named manager", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - name: sidecar
        image: sidecar:v1
      volumes:
      - name: manager
        configMap:
          name: manager-config`

		start, end := FindManagerContainerRange(yaml)
		Expect(start).To(Equal(-1))
		Expect(end).To(Equal(-1))
	})

	It("should not match a port or volumeMount named manager", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - args:
        - --leader-elect
        name: manager
        ports:
        - containerPort: 8080
          name: manager
        volumeMounts:
        - mountPath: /data
          name: manager
      - image: sidecar:v1
        name: sidecar
        ports:
        - containerPort: 9090
          name: manager`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(4))
		Expect(end).To(Equal(12))

		rangeContent := strings.Join(lines[start:end+1], "\n")
		Expect(rangeContent).To(ContainSubstring("name: manager"))
		Expect(rangeContent).NotTo(ContainSubstring("sidecar"))
	})

	It("should handle Helm directives inside container block", func() {
		yaml := `spec:
  template:
    spec:
      containers:
      - args:
        - --leader-elect
        env:
        {{- if or .Values.manager.env }}
          {{- toYaml .Values.manager.env | nindent 8 }}
        {{- end }}
        image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m`

		start, end := FindManagerContainerRange(yaml)
		lines := strings.Split(yaml, "\n")
		Expect(start).To(Equal(4))
		Expect(end).To(Equal(len(lines) - 1))

		rangeContent := strings.Join(lines[start:end+1], "\n")
		Expect(rangeContent).To(ContainSubstring("name: manager"))
		Expect(rangeContent).To(ContainSubstring("--leader-elect"))
	})

	It("should work on real yaml.Marshal output with sidecar before manager", func() {
		deployment := map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			metadataKey: map[string]any{
				nameKey: "test-controller-manager",
			},
			specKey: map[string]any{
				"template": map[string]any{
					metadataKey: map[string]any{
						"annotations": map[string]any{
							"kubectl.kubernetes.io/default-container": managerVal,
						},
					},
					specKey: map[string]any{
						containersKey: []any{
							map[string]any{
								nameKey:  "sidecar",
								imageKey: "sidecar:v1",
								"env": []any{
									map[string]any{nameKey: "SIDECAR_MODE", "value": "active"},
								},
								"resources": map[string]any{
									"limits": map[string]any{cpuKey: "100m", memoryKey: "64Mi"},
								},
							},
							map[string]any{
								nameKey:  managerVal,
								imageKey: "controller:latest",
								"args":   []any{"--leader-elect", "--health-probe-bind-address=:8081"},
								"env": []any{
									map[string]any{nameKey: "MANAGER_ENV", "value": "production"},
								},
								"resources": map[string]any{
									"limits":   map[string]any{cpuKey: "500m", memoryKey: "128Mi"},
									"requests": map[string]any{cpuKey: "10m", memoryKey: "64Mi"},
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

	It("should not false-positive on nested name fields in yaml.Marshal output", func() {
		deployment := map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			metadataKey: map[string]any{
				nameKey: "test-controller-manager",
			},
			specKey: map[string]any{
				"template": map[string]any{
					metadataKey: map[string]any{
						"annotations": map[string]any{
							"kubectl.kubernetes.io/default-container": managerVal,
						},
					},
					specKey: map[string]any{
						containersKey: []any{
							map[string]any{
								nameKey:  "sidecar",
								imageKey: "sidecar:v1",
								"ports": []any{
									map[string]any{"containerPort": 9090, nameKey: managerVal},
								},
							},
							map[string]any{
								nameKey:  managerVal,
								imageKey: "controller:latest",
								"args":   []any{"--leader-elect"},
								"volumeMounts": []any{
									map[string]any{nameKey: managerVal, "mountPath": "/data"},
								},
							},
						},
						"volumes": []any{
							map[string]any{nameKey: managerVal, "emptyDir": map[string]any{}},
						},
					},
				},
			},
		}

		yamlBytes, err := sigsyaml.Marshal(deployment)
		Expect(err).NotTo(HaveOccurred())
		yamlContent := string(yamlBytes)

		start, end := FindManagerContainerRange(yamlContent)
		Expect(start).To(BeNumerically(">=", 0),
			"should find manager despite nested name: manager fields")

		lines := strings.Split(yamlContent, "\n")
		rangeContent := strings.Join(lines[start:end+1], "\n")

		By("the range includes the actual manager container")
		Expect(rangeContent).To(ContainSubstring("name: manager"))
		Expect(rangeContent).To(ContainSubstring("controller:latest"))
		Expect(rangeContent).To(ContainSubstring("--leader-elect"))

		By("the range excludes the sidecar despite its port being named manager")
		Expect(rangeContent).NotTo(ContainSubstring("name: sidecar"))
		Expect(rangeContent).NotTo(ContainSubstring("sidecar:v1"))

		By("the range excludes the volumes section")
		Expect(rangeContent).NotTo(ContainSubstring("emptyDir"))
	})
})

var _ = Describe("templateEnvironmentVariables", func() {
	It("should not break FindManagerContainerRange for subsequent callers", func() {
		yaml := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-controller-manager
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
    spec:
      containers:
      - args:
        - --leader-elect
        env:
        - name: MY_VAR
          value: hello
        image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m`

		result := templateEnvironmentVariables(yaml)

		By("the Helm env directive should be indented, not at column 0")
		for line := range strings.SplitSeq(result, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, ".Values.manager.env") && !strings.HasPrefix(trimmed, "#") {
				_, indent := LeadingWhitespace(line)
				Expect(indent).To(BeNumerically(">", 0),
					"Helm env directive must not be at indent 0: %q", line)
			}
		}

		By("FindManagerContainerRange should still find the manager after env templating")
		start, end := FindManagerContainerRange(result)
		Expect(start).To(BeNumerically(">=", 0),
			"FindManagerContainerRange must not return -1 after templateEnvironmentVariables")

		lines := strings.Split(result, "\n")
		rangeContent := strings.Join(lines[start:end+1], "\n")
		Expect(rangeContent).To(ContainSubstring("name: manager"))
		Expect(rangeContent).To(ContainSubstring(".Values.manager.env"))
	})
})

var _ = Describe("IsManagerServiceAccount", func() {
	It("returns false for nil resource", func() {
		Expect(IsManagerServiceAccount(nil)).To(BeFalse())
	})

	It("returns true for unprefixed controller-manager name", func() {
		resource := makeUnstructuredServiceAccount("controller-manager")
		Expect(IsManagerServiceAccount(resource)).To(BeTrue())
	})

	It("returns true for prefixed controller-manager name", func() {
		resource := makeUnstructuredServiceAccount("test-project-controller-manager")
		Expect(IsManagerServiceAccount(resource)).To(BeTrue())
	})

	It("returns false for external ServiceAccount names", func() {
		resource := makeUnstructuredServiceAccount("external-sa")
		Expect(IsManagerServiceAccount(resource)).To(BeFalse())
	})
})

func makeUnstructuredServiceAccount(name string) *unstructured.Unstructured {
	resource := &unstructured.Unstructured{}
	resource.SetAPIVersion("v1")
	resource.SetKind("ServiceAccount")
	resource.SetName(name)
	return resource
}
