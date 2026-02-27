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
	helmChartLoader "helm.sh/helm/v3/pkg/chart/loader"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("Chart Generation Integration Tests", func() {
	var (
		fs             machinery.Filesystem
		tmpDir         string
		manifestsFile  string
		outputDir      string
		projectConfig  config.Config
		scaffolderBase *editKustomizeScaffolder
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "helm-chart-gen-test-*")
		Expect(err).NotTo(HaveOccurred())

		err = os.Chdir(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		fs = machinery.Filesystem{
			FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}

		projectConfig = cfgv3.New()
		projectConfig.SetProjectName("test-project")
		projectConfig.SetDomain("example.io")

		manifestsFile = filepath.Join(tmpDir, "dist", "install.yaml")
		outputDir = "dist"
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Context("Basic Functionality", func() {
		It("should generate valid helm chart with dynamic templates", func() {
			kustomizeYAML := createKustomizeWithCRDAndRBAC("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			scaffolderBase = &editKustomizeScaffolder{
				config:        projectConfig,
				fs:            fs,
				manifestsFile: manifestsFile,
				outputDir:     outputDir,
			}

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")

			By("verifying templates directory structure matches config/ structure")
			expectedDirs := []string{
				"templates/manager",
				"templates/rbac",
				"templates/crd",
			}
			for _, dir := range expectedDirs {
				dirPath := filepath.Join(chartPath, dir)
				info, err := os.Stat(dirPath)
				Expect(err).NotTo(HaveOccurred(), "Directory %s should exist", dir)
				Expect(info.IsDir()).To(BeTrue())
			}

			By("verifying manager deployment template exists")
			managerTemplate := filepath.Join(chartPath, "templates", "manager", "manager.yaml")
			_, err = os.Stat(managerTemplate)
			Expect(err).NotTo(HaveOccurred())

			By("verifying CRD templates exist")
			crdDir := filepath.Join(chartPath, "templates", "crd")
			files, err := afero.ReadDir(afero.NewOsFs(), crdDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).ToNot(BeEmpty())

			By("verifying Chart.yaml exists and is valid")
			chart, err := helmChartLoader.LoadDir(chartPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(chart.Validate()).To(Succeed())
			Expect(chart.Name()).To(Equal("test-project"))

			By("verifying essential files exist")
			essentialFiles := []string{
				"Chart.yaml",
				"values.yaml",
				".helmignore",
				"templates/_helpers.tpl",
			}
			for _, file := range essentialFiles {
				filePath := filepath.Join(chartPath, file)
				_, err := os.Stat(filePath)
				Expect(err).NotTo(HaveOccurred(), "File %s should exist", file)
			}
		})
	})

	Context("Webhook and Cert-Manager Integration", func() {
		It("should generate webhook templates with cert-manager integration and proper templating", func() {
			kustomizeYAML := createKustomizeWithWebhooksAndCertManager("e2e-test")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			projectConfig.SetProjectName("e2e-test")
			scaffolderBase = &editKustomizeScaffolder{
				config:        projectConfig,
				fs:            fs,
				manifestsFile: manifestsFile,
				outputDir:     outputDir,
			}

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")

			By("verifying webhook directory exists")
			webhookDir := filepath.Join(chartPath, "templates", "webhook")
			info, err := os.Stat(webhookDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())

			By("verifying webhook configuration files exist")
			files, err := afero.ReadDir(afero.NewOsFs(), webhookDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).ToNot(BeEmpty())

			By("verifying webhook files contain webhook configurations")
			foundValidatingWebhook := false
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				webhookFile := filepath.Join(webhookDir, file.Name())
				content, err := afero.ReadFile(afero.NewOsFs(), webhookFile)
				Expect(err).NotTo(HaveOccurred())
				contentStr := string(content)
				if strings.Contains(contentStr, "ValidatingWebhookConfiguration") {
					foundValidatingWebhook = true
					break
				}
			}
			Expect(foundValidatingWebhook).To(BeTrue(), "Expected to find ValidatingWebhookConfiguration in webhook templates")

			By("verifying cert-manager templates exist")
			certManagerDir := filepath.Join(chartPath, "templates", "cert-manager")
			certInfo, err := os.Stat(certManagerDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(certInfo.IsDir()).To(BeTrue())

			By("verifying cert-manager is enabled in values.yaml")
			valuesPath := filepath.Join(chartPath, "values.yaml")
			valuesContent, err := afero.ReadFile(afero.NewOsFs(), valuesPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(valuesContent)).To(ContainSubstring("certManager:"))
			Expect(string(valuesContent)).To(ContainSubstring("enable: true"))
		})
	})

	Context("Chart Name Handling", func() {
		It("should use project name in helpers regardless of kustomize namePrefix", func() {
			// Kustomize output with custom namePrefix
			kustomizeYAML := createKustomizeWithCustomPrefix("custom-prefix", "test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			projectConfig.SetProjectName("test-project")
			scaffolderBase = &editKustomizeScaffolder{
				config:        projectConfig,
				fs:            fs,
				manifestsFile: manifestsFile,
				outputDir:     outputDir,
			}

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")

			By("verifying _helpers.tpl uses project name, not kustomize prefix")
			helpersContent, err := os.ReadFile(filepath.Join(chartPath, "templates", "_helpers.tpl"))
			Expect(err).NotTo(HaveOccurred())
			helpersStr := string(helpersContent)

			// Should contain project name-based templates
			Expect(helpersStr).To(ContainSubstring(`define "test-project.name"`))
			Expect(helpersStr).To(ContainSubstring(`define "test-project.fullname"`))
			Expect(helpersStr).To(ContainSubstring(`define "test-project.resourceName"`))
			Expect(helpersStr).To(ContainSubstring(`define "test-project.namespaceName"`))

			// Should NOT contain kustomize prefix in template definitions
			Expect(helpersStr).NotTo(ContainSubstring(`define "custom-prefix.name"`))
			Expect(helpersStr).NotTo(ContainSubstring(`define "custom-prefix.fullname"`))

			By("verifying templates use project name helpers, not kustomize prefix")
			managerContent, err := os.ReadFile(filepath.Join(chartPath, "templates", "manager", "manager.yaml"))
			Expect(err).NotTo(HaveOccurred())
			managerStr := string(managerContent)

			Expect(managerStr).To(ContainSubstring(`include "test-project`))
			Expect(managerStr).NotTo(ContainSubstring(`custom-prefix-controller-manager`),
				"Manager template should not contain hardcoded kustomize prefix")
		})

		It("should properly template cert-manager resources when chart name is used", func() {
			kustomizeYAML := createKustomizeWithWebhooksAndCertManager("e2e-test")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			projectConfig.SetProjectName("e2e-test")
			scaffolderBase = &editKustomizeScaffolder{
				config:        projectConfig,
				fs:            fs,
				manifestsFile: manifestsFile,
				outputDir:     outputDir,
			}

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")
			chartName := "e2e-test"

			By("validating issuer name uses chartname.resourceName for 63-char safety")
			issuerPath := filepath.Join(chartPath, "templates", "cert-manager", "selfsigned-issuer.yaml")
			content, err := afero.ReadFile(afero.NewOsFs(), issuerPath)
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)

			expected := `name: {{ include "` + chartName + `.resourceName" (dict "suffix" "selfsigned-issuer" "context" $) }}`
			Expect(contentStr).To(ContainSubstring(expected),
				"Issuer name should use "+chartName+".resourceName template")
			Expect(contentStr).NotTo(ContainSubstring("e2e-test-selfsigned-issuer"),
				"Issuer name should not be hardcoded to project name")

			By("validating certificate issuerRef uses chartname.resourceName")
			certManagerDir := filepath.Join(chartPath, "templates", "cert-manager")
			files, err := afero.ReadDir(afero.NewOsFs(), certManagerDir)
			Expect(err).NotTo(HaveOccurred())

			foundCertificate := false
			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") || file.Name() == "selfsigned-issuer.yaml" {
					continue
				}

				certPath := filepath.Join(certManagerDir, file.Name())
				content, err := afero.ReadFile(afero.NewOsFs(), certPath)
				Expect(err).NotTo(HaveOccurred())
				contentStr := string(content)

				if strings.Contains(contentStr, "kind: Certificate") {
					foundCertificate = true
					expected := `name: {{ include "` + chartName + `.resourceName" (dict "suffix" "selfsigned-issuer" "context" $) }}`
					Expect(contentStr).To(ContainSubstring(expected),
						"Certificate issuerRef should use "+chartName+".resourceName template in file "+file.Name())
				}
			}
			Expect(foundCertificate).To(BeTrue(), "Expected to find at least one Certificate resource")

			By("validating cert-manager annotations use chartname.resourceName")
			// Check webhook configurations
			webhookDir := filepath.Join(chartPath, "templates", "webhook")
			if exists, _ := afero.DirExists(afero.NewOsFs(), webhookDir); exists {
				files, err := afero.ReadDir(afero.NewOsFs(), webhookDir)
				Expect(err).NotTo(HaveOccurred())

				for _, file := range files {
					if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
						continue
					}

					webhookPath := filepath.Join(webhookDir, file.Name())
					content, err := afero.ReadFile(afero.NewOsFs(), webhookPath)
					Expect(err).NotTo(HaveOccurred())
					contentStr := string(content)

					if strings.Contains(contentStr, "cert-manager.io/inject-ca-from") {
						expected := `{{ include "` + chartName + `.resourceName" (dict "suffix" "serving-cert" "context" $) }}`
						Expect(contentStr).To(ContainSubstring(expected),
							"cert-manager.io/inject-ca-from annotation should use "+chartName+".resourceName in "+file.Name())
						Expect(contentStr).NotTo(ContainSubstring("e2e-test-serving-cert"),
							"cert-manager.io/inject-ca-from annotation should not be hardcoded in "+file.Name())
					}
				}
			}

			By("validating app.kubernetes.io/name label uses chartname.name template")
			// Check all cert-manager resources
			certManagerFiles, err := afero.ReadDir(afero.NewOsFs(), certManagerDir)
			Expect(err).NotTo(HaveOccurred())

			for _, file := range certManagerFiles {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
					continue
				}

				filePath := filepath.Join(certManagerDir, file.Name())
				content, err := afero.ReadFile(afero.NewOsFs(), filePath)
				Expect(err).NotTo(HaveOccurred())
				contentStr := string(content)

				if strings.Contains(contentStr, "app.kubernetes.io/name:") {
					Expect(contentStr).To(ContainSubstring(`app.kubernetes.io/name: {{ include "`+chartName+`.name" . }}`),
						"app.kubernetes.io/name label should use "+chartName+".name template in "+file.Name())
					Expect(contentStr).NotTo(ContainSubstring("app.kubernetes.io/name: e2e-test"),
						"app.kubernetes.io/name label should not be hardcoded in "+file.Name())
				}
			}
		})
	})

	Context("Custom Output Directory", func() {
		It("should support custom output directory via --output-dir flag", func() {
			kustomizeYAML := createBasicKustomizeOutput("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			customOutputDir := "custom-charts"
			scaffolderBase = &editKustomizeScaffolder{
				config:        projectConfig,
				fs:            fs,
				manifestsFile: manifestsFile,
				outputDir:     customOutputDir,
			}

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, customOutputDir, "chart")

			By("verifying chart exists in custom directory")
			info, err := os.Stat(chartPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())

			By("verifying Chart.yaml in custom directory")
			chartFile := filepath.Join(chartPath, "Chart.yaml")
			_, err = os.Stat(chartFile)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Values Extraction", func() {
		It("should extract deployment configuration to values.yaml", func() {
			kustomizeYAML := createKustomizeWithFullDeploymentConfig("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			scaffolderBase = &editKustomizeScaffolder{
				config:        projectConfig,
				fs:            fs,
				manifestsFile: manifestsFile,
				outputDir:     outputDir,
			}

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")
			valuesPath := filepath.Join(chartPath, "values.yaml")
			valuesContent, err := os.ReadFile(valuesPath)
			Expect(err).NotTo(HaveOccurred())
			valuesStr := string(valuesContent)

			By("verifying image configuration is extracted")
			Expect(valuesStr).To(ContainSubstring("image:"))
			Expect(valuesStr).To(ContainSubstring("repository:"))
			Expect(valuesStr).To(ContainSubstring("tag:"))
			Expect(valuesStr).To(ContainSubstring("pullPolicy:"))

			By("verifying resources are extracted")
			Expect(valuesStr).To(ContainSubstring("resources:"))
			Expect(valuesStr).To(ContainSubstring("limits:"))
			Expect(valuesStr).To(ContainSubstring("requests:"))

			By("verifying security context is extracted")
			Expect(valuesStr).To(ContainSubstring("securityContext:"))
		})
	})
})

// Helper functions to create kustomize YAML outputs for different scenarios

func createBasicKustomizeOutput(projectName string) string {
	return `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-controller-manager
  namespace: ` + projectName + `-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
    control-plane: controller-manager
  name: ` + projectName + `-controller-manager
  namespace: ` + projectName + `-system
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
}

func createKustomizeWithCRDAndRBAC(projectName string) string {
	return createBasicKustomizeOutput(projectName) + `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: cronjobs.batch.tutorial.kubebuilder.io
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
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
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ` + projectName + `-manager-role
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
`
}

func createKustomizeWithWebhooks(projectName string) string {
	return createBasicKustomizeOutput(projectName) + `---
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-webhook-service
  namespace: ` + projectName + `-system
spec:
  ports:
  - port: 443
    targetPort: 9443
  selector:
    control-plane: controller-manager
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: ` + projectName + `-validating-webhook-configuration
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: ` + projectName + `-webhook-service
      namespace: ` + projectName + `-system
      path: /validate
  name: validate.example.com
  sideEffects: None
`
}

func createKustomizeWithWebhooksAndCertManager(projectName string) string {
	return createKustomizeWithWebhooks(projectName) + `---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-selfsigned-issuer
  namespace: ` + projectName + `-system
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-serving-cert
  namespace: ` + projectName + `-system
spec:
  dnsNames:
  - ` + projectName + `-webhook-service.` + projectName + `-system.svc
  - ` + projectName + `-webhook-service.` + projectName + `-system.svc.cluster.local
  issuerRef:
    kind: Issuer
    name: ` + projectName + `-selfsigned-issuer
  secretName: webhook-server-cert
`
}

func createKustomizeWithCustomPrefix(prefix, projectName string) string {
	return `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + prefix + `-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
    control-plane: controller-manager
  name: ` + prefix + `-controller-manager
  namespace: ` + prefix + `-system
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
kind: Service
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + prefix + `-controller-manager-metrics-service
  namespace: ` + prefix + `-system
spec:
  ports:
  - port: 8443
    targetPort: 8443
`
}

func createKustomizeWithFullDeploymentConfig(projectName string) string {
	return `---
apiVersion: v1
kind: Namespace
metadata:
  name: ` + projectName + `-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ` + projectName + `-controller-manager
  namespace: ` + projectName + `-system
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
        image: myrepo/controller:v1.2.3
        imagePullPolicy: IfNotPresent
        args:
        - --leader-elect
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
        ports:
        - containerPort: 9443
          name: webhook-server
          protocol: TCP
        env:
        - name: TEST_ENV
          value: "test-value"
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
`
}

func setupKustomizeFile(filePath, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(content), 0o644)
}
