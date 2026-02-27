//go:build integration

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

package scaffolds

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/kustomize"
)

var _ = Describe("Extras Directory Integration Test", func() {
	var (
		fs     machinery.Filesystem
		tmpDir string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "helm-extras-test-*")
		Expect(err).NotTo(HaveOccurred())

		// Change to tmpDir so relative paths work correctly
		err = os.Chdir(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		fs = machinery.Filesystem{
			FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Context("when converting Kustomize output with extra resources", func() {
		It("should place ConfigMap in extras directory with proper labels", func() {
			// Create a simulated kustomize output with standard resources and a ConfigMap
			kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  name: test-project-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest
---
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: custom-config
  namespace: test-project-system
data:
  key1: value1
  key2: value2
---
apiVersion: v1
kind: Secret
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: custom-secret
  namespace: test-project-system
type: Opaque
data:
  password: c2VjcmV0Cg==
`

			By("writing kustomize output to a file")
			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			By("parsing the kustomize output")
			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).NotTo(BeNil())

			By("verifying ConfigMap and Secret are in Other category")
			Expect(resources.Other).To(HaveLen(2))

			By("converting to Helm chart")
			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying extras directory was created")
			extrasDir := filepath.Join("dist", "chart", "templates", "extras")
			exists, err := afero.Exists(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "extras directory should exist")

			By("verifying extras directory contains the ConfigMap and Secret")
			files, err := afero.ReadDir(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(2), "extras should contain ConfigMap and Secret")

			var configMapFile, secretFile string
			for _, f := range files {
				if f.Name() == "custom-config.yaml" {
					configMapFile = f.Name()
				}
				if f.Name() == "custom-secret.yaml" {
					secretFile = f.Name()
				}
			}
			Expect(configMapFile).NotTo(BeEmpty(), "ConfigMap file should exist")
			Expect(secretFile).NotTo(BeEmpty(), "Secret file should exist")

			By("verifying ConfigMap has proper Helm templating")
			configMapPath := filepath.Join(extrasDir, configMapFile)
			content, err := afero.ReadFile(fs.FS, configMapPath)
			Expect(err).NotTo(HaveOccurred())
			configMapContent := string(content)

			// Verify namespace templating
			Expect(configMapContent).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"ConfigMap should have templated namespace")

			// Note: Resource names without the project prefix are kept as-is
			// This allows users to have custom resource names that don't follow the project naming convention
			Expect(configMapContent).To(ContainSubstring("name: custom-config"),
				"ConfigMap name should be preserved as-is when it doesn't match project prefix")

			// Verify standard Helm labels
			Expect(configMapContent).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`),
				"ConfigMap should have app.kubernetes.io/name label")
			Expect(configMapContent).To(ContainSubstring("app.kubernetes.io/instance: {{ .Release.Name }}"),
				"ConfigMap should have app.kubernetes.io/instance label")
			Expect(configMapContent).To(ContainSubstring("app.kubernetes.io/managed-by: {{ .Release.Service }}"),
				"ConfigMap should have app.kubernetes.io/managed-by label")
			Expect(configMapContent).To(ContainSubstring(`helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}`),
				"ConfigMap should have helm.sh/chart label")

			// Verify data is preserved
			Expect(configMapContent).To(ContainSubstring("key1: value1"),
				"ConfigMap data should be preserved")
			Expect(configMapContent).To(ContainSubstring("key2: value2"),
				"ConfigMap data should be preserved")

			By("verifying Secret has proper Helm templating")
			secretPath := filepath.Join(extrasDir, secretFile)
			content, err = afero.ReadFile(fs.FS, secretPath)
			Expect(err).NotTo(HaveOccurred())
			secretContent := string(content)

			// Verify namespace templating
			Expect(secretContent).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"Secret should have templated namespace")

			// Note: Resource names without the project prefix are kept as-is
			Expect(secretContent).To(ContainSubstring("name: custom-secret"),
				"Secret name should be preserved as-is when it doesn't match project prefix")

			// Verify standard Helm labels
			Expect(secretContent).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`),
				"Secret should have app.kubernetes.io/name label")
			Expect(secretContent).To(ContainSubstring("app.kubernetes.io/managed-by: {{ .Release.Service }}"),
				"Secret should have app.kubernetes.io/managed-by label")

			// Verify data is preserved
			Expect(secretContent).To(ContainSubstring("password: c2VjcmV0Cg=="),
				"Secret data should be preserved")
		})

		It("should place custom Service in extras directory", func() {
			kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: custom-service
  namespace: test-project-system
spec:
  ports:
  - port: 8080
    targetPort: 8080
  selector:
    app: custom
`

			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying custom Service is in extras directory")
			extrasDir := filepath.Join("dist", "chart", "templates", "extras")
			exists, err := afero.Exists(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			files, err := afero.ReadDir(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(1))
			Expect(files[0].Name()).To(Equal("custom-service.yaml"))

			By("verifying Service has proper Helm templating")
			servicePath := filepath.Join(extrasDir, files[0].Name())
			content, err := afero.ReadFile(fs.FS, servicePath)
			Expect(err).NotTo(HaveOccurred())
			serviceContent := string(content)

			Expect(serviceContent).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
			Expect(serviceContent).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`))
		})

		It("should not place webhook or metrics services in extras", func() {
			kustomizeYAML := `---
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-webhook-service
  namespace: test-project-system
spec:
  ports:
  - port: 443
    targetPort: 9443
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-controller-manager-metrics-service
  namespace: test-project-system
spec:
  ports:
  - port: 8443
    targetPort: 8443
`

			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying extras directory was NOT created")
			extrasDir := filepath.Join("dist", "chart", "templates", "extras")
			exists, err := afero.Exists(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse(), "extras directory should not exist for webhook/metrics services")

			By("verifying webhook directory was created")
			webhookDir := filepath.Join("dist", "chart", "templates", "webhook")
			exists, err = afero.Exists(fs.FS, webhookDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			By("verifying metrics directory was created")
			metricsDir := filepath.Join("dist", "chart", "templates", "metrics")
			exists, err = afero.Exists(fs.FS, metricsDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})

	Context("RBAC resource placement", func() {
		It("should ensure all RBAC resources go to rbac directory, never to extras", func() {
			// Critical test: RBAC resources must NEVER end up in extras directory
			kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  name: test-project-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-manager-role
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-project-leader-election-role
  namespace: test-project-system
rules:
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: test-project-manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: test-project-manager-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-leader-election-rolebinding
  namespace: test-project-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: test-project-leader-election-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-config
  namespace: test-project-system
data:
  key: value
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest
`

			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			// Verify parser correctly categorized RBAC resources
			Expect(resources.ServiceAccount).NotTo(BeNil(), "ServiceAccount should be parsed")
			Expect(resources.ClusterRoles).To(HaveLen(1), "should have 1 ClusterRole")
			Expect(resources.Roles).To(HaveLen(1), "should have 1 Role")
			Expect(resources.ClusterRoleBindings).To(HaveLen(1), "should have 1 ClusterRoleBinding")
			Expect(resources.RoleBindings).To(HaveLen(1), "should have 1 RoleBinding")
			Expect(resources.Other).To(HaveLen(1), "ConfigMap should be in Other")

			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying rbac directory exists and contains all RBAC resources")
			rbacDir := filepath.Join("dist", "chart", "templates", "rbac")
			exists, err := afero.Exists(fs.FS, rbacDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "rbac directory must exist")

			rbacFiles, err := afero.ReadDir(fs.FS, rbacDir)
			Expect(err).NotTo(HaveOccurred())
			// Should have: 1 ServiceAccount + 1 ClusterRole + 1 Role + 1 ClusterRoleBinding + 1 RoleBinding = 5 files
			Expect(rbacFiles).To(HaveLen(5), "rbac directory should have exactly 5 RBAC files")

			By("verifying NO RBAC resources in extras directory")
			extrasDir := filepath.Join("dist", "chart", "templates", "extras")
			exists, err = afero.Exists(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "extras directory should exist for ConfigMap")

			extrasFiles, err := afero.ReadDir(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(extrasFiles).To(HaveLen(1), "extras should only have ConfigMap, not RBAC")

			// Verify the extras file is the ConfigMap, not any RBAC resource
			configMapFound := false
			for _, f := range extrasFiles {
				if strings.Contains(f.Name(), "custom-config") {
					configMapFound = true
				}
				// Ensure no RBAC-related files
				Expect(f.Name()).NotTo(ContainSubstring("role"), "no Role files in extras")
				Expect(f.Name()).NotTo(ContainSubstring("rolebinding"), "no RoleBinding files in extras")
				Expect(f.Name()).NotTo(ContainSubstring("serviceaccount"), "no ServiceAccount files in extras")
			}
			Expect(configMapFound).To(BeTrue(), "ConfigMap should be in extras")
		})
	})

	Context("when converting namespace-scoped RBAC resources", func() {
		It("should convert namespace-scoped Roles with explicit namespaces to Helm templates", func() {
			// This test validates the scenario from issue where namespace-scoped Roles
			// (used for cross-namespace permissions, leader election, etc.) must be
			// included in the generated Helm chart, not just the ClusterRole.
			kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  name: test-project-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-manager-role
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
rules:
- apiGroups: ["example.com"]
  resources: ["myresources"]
  verbs: ["get", "list", "watch", "create", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-project-manager-role
  namespace: infrastructure
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
rules:
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "patch", "update", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-project-leader-election-role
  namespace: test-project-system
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
rules:
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-project-events-role
  namespace: production
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
rules:
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: test-project-manager-rolebinding
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: test-project-manager-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-manager-rolebinding
  namespace: infrastructure
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: test-project-manager-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-leader-election-rolebinding
  namespace: test-project-system
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: test-project-leader-election-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-events-rolebinding
  namespace: production
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: test-project-events-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest
`

			By("writing kustomize output to a file")
			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			By("parsing the kustomize output")
			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).NotTo(BeNil())

			By("verifying parser correctly categorized all RBAC resources")
			Expect(resources.ClusterRoles).To(HaveLen(1), "should have 1 ClusterRole")
			Expect(resources.Roles).To(HaveLen(3), "should have 3 namespace-scoped Roles")
			Expect(resources.ClusterRoleBindings).To(HaveLen(1), "should have 1 ClusterRoleBinding")
			Expect(resources.RoleBindings).To(HaveLen(3), "should have 3 RoleBindings")

			By("converting to Helm chart")
			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying rbac directory was created")
			rbacDir := filepath.Join("dist", "chart", "templates", "rbac")
			exists, err := afero.Exists(fs.FS, rbacDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "rbac directory should exist")

			By("verifying all RBAC files are present")
			rbacFiles, err := afero.ReadDir(fs.FS, rbacDir)
			Expect(err).NotTo(HaveOccurred())
			// Should have: 1 ServiceAccount + 1 ClusterRole + 1 ClusterRoleBinding + 3 Roles + 3 RoleBindings = 9 files
			Expect(rbacFiles).To(HaveLen(9), "should have 9 RBAC files total")

			By("verifying ClusterRole file exists")
			// ClusterRole has no namespace, so filename is just the name (with project prefix removed)
			clusterRolePath := filepath.Join(rbacDir, "manager-role.yaml")
			exists, err = afero.Exists(fs.FS, clusterRolePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "ClusterRole file should exist")

			By("verifying infrastructure Role file exists")
			// Role has namespace, so filename includes namespace suffix: name-namespace.yaml
			infrastructureRolePath := filepath.Join(rbacDir, "manager-role-infrastructure.yaml")
			exists, err = afero.Exists(fs.FS, infrastructureRolePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "infrastructure Role file should exist")

			By("verifying project namespace Role file exists")
			// Role in project namespace (test-project-system) should NOT have namespace suffix
			projectRolePath := filepath.Join(rbacDir, "leader-election-role.yaml")
			exists, err = afero.Exists(fs.FS, projectRolePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "project namespace Role file should exist without suffix")

			By("verifying production Role file exists")
			// Role in cross-namespace should have namespace suffix
			productionRolePath := filepath.Join(rbacDir, "events-role-production.yaml")
			exists, err = afero.Exists(fs.FS, productionRolePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "production Role file should exist with suffix")

			By("verifying infrastructure RoleBinding file exists")
			infrastructureBindingPath := filepath.Join(rbacDir, "manager-rolebinding-infrastructure.yaml")
			exists, err = afero.Exists(fs.FS, infrastructureBindingPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "infrastructure RoleBinding file should exist")

			By("verifying project namespace RoleBinding file exists")
			// RoleBinding in project namespace should NOT have namespace suffix
			projectBindingPath := filepath.Join(rbacDir, "leader-election-rolebinding.yaml")
			exists, err = afero.Exists(fs.FS, projectBindingPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "project namespace RoleBinding file should exist without suffix")

			By("verifying production RoleBinding file exists")
			// RoleBinding in cross-namespace should have namespace suffix
			productionBindingPath := filepath.Join(rbacDir, "events-rolebinding-production.yaml")
			exists, err = afero.Exists(fs.FS, productionBindingPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "production RoleBinding file should exist with suffix")

			By("verifying infrastructure Role has proper Helm templating")
			roleContent, err := afero.ReadFile(fs.FS, infrastructureRolePath)
			Expect(err).NotTo(HaveOccurred())
			roleContentStr := string(roleContent)

			// Verify namespace is preserved (not templated to .Release.Namespace)
			// because it's an explicit cross-namespace permission
			Expect(roleContentStr).To(ContainSubstring("namespace: infrastructure"),
				"Role should preserve explicit namespace for cross-namespace permissions")

			// Verify standard Helm labels
			Expect(roleContentStr).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`),
				"Role should have templated app.kubernetes.io/name label")

			// Verify name is templated
			Expect(roleContentStr).To(ContainSubstring(`name: {{ include "test-project.resourceName"`),
				"Role name should be templated")

			// Verify rules are preserved
			Expect(roleContentStr).To(ContainSubstring("apiGroups:"),
				"Role rules should be preserved")
			Expect(roleContentStr).To(ContainSubstring("- apps"),
				"Role should have apps API group")
			Expect(roleContentStr).To(ContainSubstring("- deployments"),
				"Role should have deployments resource")

			By("verifying project namespace Role has proper Helm templating")
			projRoleContent, err := afero.ReadFile(fs.FS, projectRolePath)
			Expect(err).NotTo(HaveOccurred())
			projRoleContentStr := string(projRoleContent)

			// Verify namespace is templated to .Release.Namespace for project namespace
			Expect(projRoleContentStr).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"Role in project namespace should template namespace to .Release.Namespace")
			Expect(projRoleContentStr).NotTo(ContainSubstring("namespace: test-project-system"),
				"Role should not have hardcoded project namespace")

			// Verify leader election permissions
			Expect(projRoleContentStr).To(ContainSubstring("- coordination.k8s.io"))
			Expect(projRoleContentStr).To(ContainSubstring("- leases"))

			By("verifying production Role has proper Helm templating")
			prodRoleContent, err := afero.ReadFile(fs.FS, productionRolePath)
			Expect(err).NotTo(HaveOccurred())
			prodRoleContentStr := string(prodRoleContent)

			// Verify namespace is preserved for cross-namespace Role
			Expect(prodRoleContentStr).To(ContainSubstring("namespace: production"),
				"Role should preserve explicit namespace for cross-namespace permissions")

			// Verify events permissions
			Expect(prodRoleContentStr).To(ContainSubstring("- events"))

			By("verifying infrastructure RoleBinding has proper Helm templating")
			bindingContent, err := afero.ReadFile(fs.FS, infrastructureBindingPath)
			Expect(err).NotTo(HaveOccurred())
			bindingContentStr := string(bindingContent)

			// Verify namespace is preserved
			Expect(bindingContentStr).To(ContainSubstring("namespace: infrastructure"),
				"RoleBinding should preserve explicit namespace")

			// Verify roleRef is templated
			Expect(bindingContentStr).To(ContainSubstring(`name: {{ include "test-project.resourceName"`),
				"RoleBinding roleRef should be templated")

			// Verify subjects namespace is templated (references the controller namespace)
			Expect(bindingContentStr).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"RoleBinding subject namespace should reference the release namespace")

			By("verifying project namespace RoleBinding has proper Helm templating")
			projBindingContent, err := afero.ReadFile(fs.FS, projectBindingPath)
			Expect(err).NotTo(HaveOccurred())
			projBindingContentStr := string(projBindingContent)

			// Verify namespace is templated for project namespace RoleBinding
			Expect(projBindingContentStr).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"RoleBinding metadata namespace should be templated for project namespace")

			// Verify standard Helm labels
			Expect(projBindingContentStr).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`),
				"RoleBinding should have templated labels")

			By("verifying production RoleBinding has proper Helm templating")
			prodBindingContent, err := afero.ReadFile(fs.FS, productionBindingPath)
			Expect(err).NotTo(HaveOccurred())
			prodBindingContentStr := string(prodBindingContent)

			// Verify namespace is preserved for cross-namespace RoleBinding
			Expect(prodBindingContentStr).To(ContainSubstring("namespace: production"),
				"RoleBinding should preserve explicit namespace for cross-namespace binding")

			// Subject namespace should still be templated (references the controller namespace)
			Expect(prodBindingContentStr).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"RoleBinding subject namespace should be templated to Release.Namespace")

			By("verifying ClusterRole does not have namespace field")
			clusterRoleContent, err := afero.ReadFile(fs.FS, clusterRolePath)
			Expect(err).NotTo(HaveOccurred())
			clusterRoleContentStr := string(clusterRoleContent)

			// ClusterRole should not have namespace field
			Expect(clusterRoleContentStr).NotTo(ContainSubstring("namespace:"),
				"ClusterRole should not have namespace field")
		})

		It("should preserve ANY namespace field that differs from manager namespace", func() {
			// This test validates that ANY namespace reference (metadata, subjects, etc.)
			// that is NOT the manager namespace gets preserved exactly as-is
			kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  name: test-project-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-sa
  namespace: external-namespace
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-cross-ns-binding
  namespace: infrastructure
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: some-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system
- kind: ServiceAccount
  name: external-sa
  namespace: external-namespace
- kind: ServiceAccount
  name: another-sa
  namespace: production
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      serviceAccountName: test-project-controller-manager
      containers:
      - name: manager
        image: controller:latest
`

			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying RoleBinding preserves all non-manager namespaces")
			rbacDir := filepath.Join("dist", "chart", "templates", "rbac")
			bindingPath := filepath.Join(rbacDir, "cross-ns-binding-infrastructure.yaml")
			exists, err := afero.Exists(fs.FS, bindingPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "RoleBinding file should exist")

			bindingContent, err := afero.ReadFile(fs.FS, bindingPath)
			Expect(err).NotTo(HaveOccurred())
			bindingStr := string(bindingContent)

			// Metadata namespace should be preserved (cross-namespace)
			Expect(bindingStr).To(ContainSubstring("namespace: infrastructure"),
				"metadata namespace should be preserved")

			// Manager namespace in subjects should be templated
			Expect(bindingStr).To(MatchRegexp(`kind: ServiceAccount\s+name:.*\s+namespace: \{\{ \.Release\.Namespace \}\}`),
				"manager namespace in subject should be templated")

			// External namespaces in subjects should be preserved
			Expect(bindingStr).To(ContainSubstring("namespace: external-namespace"),
				"external-namespace in subject should be preserved")
			Expect(bindingStr).To(ContainSubstring("namespace: production"),
				"production namespace in subject should be preserved")

			// Count namespace occurrences
			infrastructureCount := strings.Count(bindingStr, "namespace: infrastructure")
			externalCount := strings.Count(bindingStr, "namespace: external-namespace")
			productionCount := strings.Count(bindingStr, "namespace: production")
			releaseNsCount := strings.Count(bindingStr, "namespace: {{ .Release.Namespace }}")

			Expect(infrastructureCount).To(Equal(1), "should have 1 infrastructure namespace (metadata)")
			Expect(externalCount).To(Equal(1), "should have 1 external-namespace (subject)")
			Expect(productionCount).To(Equal(1), "should have 1 production namespace (subject)")
			Expect(releaseNsCount).To(BeNumerically(">=", 1), "should have at least 1 templated namespace (manager subject)")

			By("verifying external ServiceAccount preserves its namespace")
			saFiles, err := afero.ReadDir(fs.FS, rbacDir)
			Expect(err).NotTo(HaveOccurred())

			var externalSAFound bool
			for _, f := range saFiles {
				if strings.Contains(f.Name(), "external-sa") {
					externalSAFound = true
					saPath := filepath.Join(rbacDir, f.Name())
					saContent, err := afero.ReadFile(fs.FS, saPath)
					Expect(err).NotTo(HaveOccurred())
					saStr := string(saContent)

					// External SA namespace should be preserved
					Expect(saStr).To(ContainSubstring("namespace: external-namespace"),
						"external ServiceAccount namespace should be preserved")
					Expect(saStr).NotTo(ContainSubstring("namespace: {{ .Release.Namespace }}"),
						"external ServiceAccount should not use Release.Namespace")
				}
			}
			Expect(externalSAFound).To(BeTrue(), "external ServiceAccount file should exist")
		})

		It("should escape existing Go template syntax in CRD samples", func() {
			// Test a CRD with Go template syntax in default values.
			// Real-world example: gitops-promoter's ChangeTransferPolicy CRD has templates
			// in pullRequest.template fields that should be preserved as literal text.
			kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-system
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: changetransferpolicies.promoter.argoproj.io
  namespace: test-project-system
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
spec:
  group: promoter.argoproj.io
  names:
    kind: ChangeTransferPolicy
    listKind: ChangeTransferPolicyList
    plural: changetransferpolicies
    singular: changetransferpolicy
  scope: Namespaced
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        description: ChangeTransferPolicy is the Schema for the changetransferpolicies API
        properties:
          spec:
            properties:
              activeBranch:
                type: string
              pullRequest:
                properties:
                  template:
                    properties:
                      description:
                        default: "Promoting {{ .ChangeTransferPolicy.Spec.ActiveBranch }}"
                        type: string
                      title:
                        default: "Promote {{ trunc 5 .ChangeTransferPolicy.Status.Proposed.Dry.Sha }}"
                        type: string
                    type: object
                type: object
            type: object
        type: object
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      serviceAccountName: test-project-controller-manager
      containers:
      - name: manager
        image: controller:latest
        args:
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
`

			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying CRD file has escaped Go template syntax")
			crdDir := filepath.Join("dist", "chart", "templates", "crd")
			crdPath := filepath.Join(crdDir, "changetransferpolicies.promoter.argoproj.io.yaml")
			exists, err := afero.Exists(fs.FS, crdPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "CRD file should exist")

			crdContent, err := afero.ReadFile(fs.FS, crdPath)
			Expect(err).NotTo(HaveOccurred())
			crdStr := string(crdContent)

			// Existing Go template syntax should be escaped to prevent Helm from parsing it
			Expect(crdStr).To(ContainSubstring(`{{ "{{ .ChangeTransferPolicy.Spec.ActiveBranch }}" }}`),
				"existing template syntax should be escaped")
			Expect(crdStr).To(ContainSubstring(`{{ "{{ trunc 5 .ChangeTransferPolicy.Status.Proposed.Dry.Sha }}" }}`),
				"template functions should be escaped")

			// Verify we don't have unescaped template syntax that would break Helm rendering
			// We check that all ChangeTransferPolicy references are properly wrapped in escaped strings
			// Pattern checks for: default: "...<text>{{ .ChangeTransferPolicy" (not escaped)
			// The properly escaped version is: default: "...{{ "{{ .ChangeTransferPolicy..." }}"
			Expect(crdStr).NotTo(MatchRegexp(`default:\s+"[^{]*\{\{\s*\.ChangeTransferPolicy`),
				"unescaped Go templates should not exist in default values")

			// Helm templates we add should still work (not escaped)
			Expect(crdStr).To(ContainSubstring("{{- if .Values.crd.enable }}"),
				"Helm conditional should be present and NOT escaped")
			Expect(crdStr).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"Helm namespace template should be present and NOT escaped")
			Expect(crdStr).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`),
				"Helm label template should be present and NOT escaped")
		})
	})

	Context("Custom Resource instances", func() {
		It("should ignore Custom Resource instances and not include them in the chart", func() {
			// This test validates that Custom Resources (CR instances, not CRDs) are
			// intentionally ignored and not included in the generated Helm chart.
			// CRs are environment-specific and should not be installed automatically.
			kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-system
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: cronjobs.batch.tutorial.kubebuilder.io
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
spec:
  group: batch.tutorial.kubebuilder.io
  names:
    kind: CronJob
    listKind: CronJobList
    plural: cronjobs
    singular: cronjob
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        description: CronJob is the Schema for the cronjobs API
        type: object
        properties:
          spec:
            type: object
            properties:
              schedule:
                type: string
---
apiVersion: batch.tutorial.kubebuilder.io/v1
kind: CronJob
metadata:
  labels:
    app.kubernetes.io/name: test-project
    app.kubernetes.io/managed-by: kustomize
  name: cronjob-sample
spec:
  schedule: "*/1 * * * *"
---
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: custom-config
  namespace: test-project-system
data:
  key1: value1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      serviceAccountName: test-project-controller-manager
      containers:
      - name: manager
        image: controller:latest
`

			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			By("verifying CRD and CR are correctly parsed")
			Expect(resources.CustomResourceDefinitions).To(HaveLen(1), "should have 1 CRD")
			Expect(resources.CustomResources).To(HaveLen(1), "should have 1 CR instance")
			Expect(resources.Other).To(HaveLen(1), "should have 1 other resource (ConfigMap)")

			By("verifying CR is a CronJob")
			cr := resources.CustomResources[0]
			Expect(cr.GetKind()).To(Equal("CronJob"))
			Expect(cr.GetAPIVersion()).To(Equal("batch.tutorial.kubebuilder.io/v1"))
			Expect(cr.GetName()).To(Equal("cronjob-sample"))

			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying CR is NOT included in the chart (no samples directory)")
			samplesDir := filepath.Join("dist", "chart", "samples")
			exists, err := afero.Exists(fs.FS, samplesDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse(), "samples directory should NOT exist - CRs are ignored")

			By("verifying CR is NOT in extras directory")
			extrasDir := filepath.Join("dist", "chart", "templates", "extras")
			exists, err = afero.Exists(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "extras directory should exist for ConfigMap")

			extrasFiles, err := afero.ReadDir(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(extrasFiles).To(HaveLen(1), "extras should only have ConfigMap, not CR")

			var configMapFound, crFound bool
			for _, f := range extrasFiles {
				if strings.Contains(f.Name(), "custom-config") {
					configMapFound = true
				}
				if strings.Contains(strings.ToLower(f.Name()), "cronjob") {
					crFound = true
				}
			}
			Expect(configMapFound).To(BeTrue(), "ConfigMap should be in extras")
			Expect(crFound).To(BeFalse(), "CR should NOT be in extras")

			By("verifying CRD is in crd directory")
			crdDir := filepath.Join("dist", "chart", "templates", "crd")
			exists, err = afero.Exists(fs.FS, crdDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "crd directory should exist")

			crdFiles, err := afero.ReadDir(fs.FS, crdDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(crdFiles).To(HaveLen(1), "crd directory should have 1 CRD")
		})
	})
})
