//go:build integration

/*
Copyright 2025 The Kubernetes Authors.

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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser", func() {
	var (
		parser   *Parser
		tempFile string
	)

	BeforeEach(func() {
		// Create a temporary file for testing
		tempDir := GinkgoT().TempDir()
		tempFile = filepath.Join(tempDir, "test-manifest.yaml")
	})

	Context("with valid YAML containing various resources", func() {
		BeforeEach(func() {
			yamlContent := `---
apiVersion: v1
kind: Namespace
metadata:
  name: test-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: test-system
spec:
  replicas: 1
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager
  namespace: test-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: manager-role
subjects:
- kind: ServiceAccount
  name: controller-manager
  namespace: test-system
---
apiVersion: v1
kind: Service
metadata:
  name: controller-manager-metrics-service
  namespace: test-system
spec:
  ports:
  - name: https
    port: 8443
    targetPort: 8443
  selector:
    control-plane: controller-manager
`
			err := os.WriteFile(tempFile, []byte(yamlContent), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser = NewParser(tempFile)
		})

		It("should parse all resources correctly", func() {
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).NotTo(BeNil())

			// Check that resources were parsed
			Expect(resources.Namespace).NotTo(BeNil())
			Expect(resources.Deployment).NotTo(BeNil())
			Expect(resources.ServiceAccount).NotTo(BeNil())

			// Check RBAC resources
			Expect(resources.ClusterRoles).To(HaveLen(1))
			Expect(resources.ClusterRoleBindings).To(HaveLen(1))

			// Check Services
			Expect(resources.Services).To(HaveLen(1))
		})

		It("should identify correct resource types", func() {
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			Expect(resources.Namespace.GetKind()).To(Equal("Namespace"))
			Expect(resources.Deployment.GetKind()).To(Equal("Deployment"))
			Expect(resources.ServiceAccount.GetKind()).To(Equal("ServiceAccount"))
		})
	})

	Context("with webhook configuration", func() {
		BeforeEach(func() {
			yamlContent := `---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: validating-webhook-configuration
webhooks:
- name: test.example.com
  clientConfig:
    service:
      name: webhook-service
      namespace: test-system
      path: "/validate"
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: serving-cert
  namespace: test-system
spec:
  dnsNames:
  - webhook-service.test-system.svc
  - webhook-service.test-system.svc.cluster.local
  issuerRef:
    kind: Issuer
    name: selfsigned-issuer
  secretName: webhook-server-cert
`
			err := os.WriteFile(tempFile, []byte(yamlContent), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser = NewParser(tempFile)
		})

		It("should parse webhook configurations", func() {
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			Expect(resources.WebhookConfigurations).To(HaveLen(1))
			Expect(resources.Certificates).To(HaveLen(1))

			webhook := resources.WebhookConfigurations[0]
			Expect(webhook.GetKind()).To(Equal("ValidatingWebhookConfiguration"))

			cert := resources.Certificates[0]
			Expect(cert.GetKind()).To(Equal("Certificate"))
		})
	})

	Context("with ServiceMonitor", func() {
		BeforeEach(func() {
			yamlContent := `---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: controller-manager-metrics-monitor
  namespace: test-system
spec:
  endpoints:
  - path: /metrics
    port: https
  selector:
    matchLabels:
      control-plane: controller-manager
`
			err := os.WriteFile(tempFile, []byte(yamlContent), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser = NewParser(tempFile)
		})

		It("should parse ServiceMonitor", func() {
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			Expect(resources.ServiceMonitors).To(HaveLen(1))

			monitor := resources.ServiceMonitors[0]
			Expect(monitor.GetKind()).To(Equal("ServiceMonitor"))
		})
	})

	Context("with empty or invalid YAML", func() {
		It("should handle empty file gracefully", func() {
			err := os.WriteFile(tempFile, []byte(""), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser = NewParser(tempFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).NotTo(BeNil())
		})

		It("should return error for invalid YAML", func() {
			invalidYAML := `invalid: yaml: content: [unclosed`
			err := os.WriteFile(tempFile, []byte(invalidYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser = NewParser(tempFile)
			_, err = parser.Parse()
			Expect(err).To(HaveOccurred())
		})

		It("should return error for non-existent file", func() {
			parser = NewParser("/non/existent/file.yaml")
			_, err := parser.Parse()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("resource organization", func() {
		BeforeEach(func() {
			yamlContent := `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: tests.example.com
spec:
  group: example.com
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned-issuer
  namespace: test-system
spec:
  selfSigned: {}
`
			err := os.WriteFile(tempFile, []byte(yamlContent), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser = NewParser(tempFile)
		})

		It("should organize CRDs and Issuers correctly", func() {
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			Expect(resources.CustomResourceDefinitions).To(HaveLen(1))
			Expect(resources.Issuer).NotTo(BeNil())

			crd := resources.CustomResourceDefinitions[0]
			Expect(crd.GetKind()).To(Equal("CustomResourceDefinition"))

			issuer := resources.Issuer
			Expect(issuer.GetKind()).To(Equal("Issuer"))
		})
	})

	Context("with sample Custom Resources", func() {
		BeforeEach(func() {
			yamlContent := `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: captains.crew.testproject.org
spec:
  group: crew.testproject.org
  names:
    kind: Captain
    plural: captains
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: sailors.crew.testproject.org
spec:
  group: crew.testproject.org
  names:
    kind: Sailor
    plural: sailors
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: myapps.example.com
spec:
  group: example.com
  names:
    kind: MyApp
    plural: myapps
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
---
apiVersion: crew.testproject.org/v1
kind: Captain
metadata:
  name: captain-sample
  namespace: test-system
spec:
  rank: "Admiral"
---
apiVersion: crew.testproject.org/v1
kind: Sailor
metadata:
  name: sailor-sample
  namespace: test-system
spec:
  rank: "Seaman"
---
apiVersion: example.com/v1alpha1
kind: MyApp
metadata:
  name: myapp-sample
  namespace: test-system
spec:
  replicas: 1
`
			err := os.WriteFile(tempFile, []byte(yamlContent), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser = NewParser(tempFile)
		})

		It("should parse sample CRs correctly", func() {
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			// Verify that custom resources are identified as samples
			Expect(resources.SampleResources).To(HaveLen(3))

			// Check kinds of sample resources
			kinds := make([]string, 0, len(resources.SampleResources))
			for _, sample := range resources.SampleResources {
				kinds = append(kinds, sample.GetKind())
			}
			Expect(kinds).To(ContainElements("Captain", "Sailor", "MyApp"))
		})

		It("should not categorize infrastructure as samples", func() {
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			// Verify infrastructure resources are NOT in samples
			for _, sample := range resources.SampleResources {
				Expect(sample.GetKind()).NotTo(BeElementOf(
					"Deployment", "Service", "ServiceAccount",
					"Role", "ClusterRole", "Namespace",
				))
			}
		})

		It("should only categorize CRs with defined CRDs as samples", func() {
			yamlContent := `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: captains.crew.testproject.org
spec:
  group: crew.testproject.org
  names:
    kind: Captain
    plural: captains
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
---
apiVersion: crew.testproject.org/v1
kind: Captain
metadata:
  name: captain-sample
spec:
  rank: Admiral
---
apiVersion: other.example.com/v1
kind: Orphan
metadata:
  name: orphan-sample
spec:
  value: test
`
			tempDir := GinkgoT().TempDir()
			tempFile := filepath.Join(tempDir, "test.yaml")
			err := os.WriteFile(tempFile, []byte(yamlContent), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser := NewParser(tempFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			// Captain has a CRD defined, should be in samples
			Expect(resources.SampleResources).To(HaveLen(1))
			Expect(resources.SampleResources[0].GetKind()).To(Equal("Captain"))

			// Orphan has NO CRD defined, should be in Other
			Expect(resources.Other).To(HaveLen(1))
			Expect(resources.Other[0].GetKind()).To(Equal("Orphan"))
		})
	})

	Context("with mixed infrastructure and sample CRs", func() {
		BeforeEach(func() {
			yamlContent := `---
apiVersion: v1
kind: Namespace
metadata:
  name: test-system
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: captains.crew.testproject.org
spec:
  group: crew.testproject.org
  names:
    kind: Captain
    plural: captains
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: test-system
spec:
  replicas: 1
---
apiVersion: crew.testproject.org/v1
kind: Captain
metadata:
  name: captain-sample
  namespace: test-system
spec:
  rank: "Admiral"
---
apiVersion: v1
kind: Service
metadata:
  name: metrics-service
  namespace: test-system
spec:
  ports:
  - port: 8443
`
			err := os.WriteFile(tempFile, []byte(yamlContent), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser = NewParser(tempFile)
		})

		It("should correctly separate infrastructure from samples", func() {
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			// Infrastructure should be in their respective categories
			Expect(resources.Namespace).NotTo(BeNil())
			Expect(resources.Deployment).NotTo(BeNil())
			Expect(resources.Services).To(HaveLen(1))

			// Custom Resource should be in samples
			Expect(resources.SampleResources).To(HaveLen(1))
			Expect(resources.SampleResources[0].GetKind()).To(Equal("Captain"))

			// Samples should not be empty
			Expect(resources.Other).To(BeEmpty())
		})
	})

	Context("with custom prefix", func() {
		BeforeEach(func() {
			yamlContent := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ln-controller-manager
  namespace: long-name-test-system
spec:
  replicas: 1
---
apiVersion: v1
kind: Service
metadata:
  name: ln-controller-manager-metrics-service
  namespace: long-name-test-system
spec:
  ports:
  - name: https
    port: 8443
    targetPort: 8443
  selector:
    control-plane: ln-controller-manager
`
			err := os.WriteFile(tempFile, []byte(yamlContent), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser = NewParser(tempFile)
		})

		It("should use the correct prefix", func() {
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			Expect(resources.EstimatePrefix("long-name")).To(Equal("ln"))
		})
	})
})
