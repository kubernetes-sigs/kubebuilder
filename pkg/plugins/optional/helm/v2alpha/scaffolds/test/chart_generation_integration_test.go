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

package test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"helm.sh/helm/v3/pkg/action"
	helmChartLoader "helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds"
)

var _ = Describe("Chart Generation Integration Tests", func() {
	var (
		fs             machinery.Filesystem
		tmpDir         string
		manifestsFile  string
		outputDir      string
		projectConfig  config.Config
		scaffolderBase plugins.Scaffolder
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

			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

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

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)

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

			By("keeping cert-manager commented when kustomize output has no webhooks")
			workflow, err := os.ReadFile(filepath.Join(tmpDir, ".github", "workflows", "test-chart.yml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(workflow)).To(ContainSubstring(
				"#      - name: Install cert-manager via Helm (wait for readiness)"))
		})
	})

	Context("Webhook and Cert-Manager Integration", func() {
		It("should generate webhook templates with cert-manager integration and proper templating", func() {
			kustomizeYAML := createKustomizeWithWebhooksAndCertManager("e2e-test")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			projectConfig.SetProjectName("e2e-test")
			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")

			By("verifying webhook directory exists")
			webhookDir := filepath.Join(chartPath, "templates", "webhook")
			info, err := os.Stat(webhookDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())

			By("enabling cert-manager when kustomize output has webhooks")
			workflow, err := os.ReadFile(filepath.Join(tmpDir, ".github", "workflows", "test-chart.yml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(workflow)).To(ContainSubstring(
				"      - name: Install cert-manager via Helm (wait for readiness)"))
			Expect(string(workflow)).NotTo(ContainSubstring(
				"#      - name: Install cert-manager via Helm (wait for readiness)"))

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
			Expect(string(valuesContent)).To(ContainSubstring("certManager:\n  enabled: true"))

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)
		})
	})

	Context("Chart Name Handling", func() {
		It("should use project name in helpers regardless of kustomize namePrefix", func() {
			// Kustomize output with custom namePrefix
			kustomizeYAML := createKustomizeWithCustomPrefix("custom-prefix", "test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			projectConfig.SetProjectName("test-project")
			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

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

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)
		})

		It("should properly template cert-manager resources when chart name is used", func() {
			kustomizeYAML := createKustomizeWithWebhooksAndCertManager("e2e-test")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			projectConfig.SetProjectName("e2e-test")
			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

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

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)
		})
	})

	// helmTemplate scaffolds kustomizeYAML into a chart and runs `helm template`, returning the
	// combined output. Specs are skipped when helm is not on PATH.
	helmTemplate := func(kustomizeYAML string, setArgs ...string) (string, error) {
		if _, err := exec.LookPath("helm"); err != nil {
			Skip("helm binary not found on PATH; skipping render-based test")
		}

		Expect(setupKustomizeFile(manifestsFile, kustomizeYAML)).To(Succeed())

		scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
		scaffolderBase.InjectFS(fs)
		Expect(scaffolderBase.Scaffold()).To(Succeed())

		chartPath := filepath.Join(tmpDir, outputDir, "chart")
		args := append([]string{"template", "my-release", chartPath, "--namespace", "my-namespace"}, setArgs...)
		out, err := exec.Command("helm", args...).CombinedOutput()
		return string(out), err
	}

	// serviceAccountDoc returns the ServiceAccount YAML document from a multi-document render.
	serviceAccountDoc := func(rendered string) string {
		for _, doc := range strings.Split(rendered, "\n---") {
			if strings.Contains(doc, "\nkind: ServiceAccount\n") {
				return doc
			}
		}
		return ""
	}

	Context("ServiceAccount name resolution (rendered)", func() {
		const generatedName = "my-release-test-project-controller-manager"

		runHelmTemplate := func(setArgs ...string) (string, error) {
			return helmTemplate(createKustomizeForServiceAccountRender("test-project"), setArgs...)
		}

		renderChart := func(setArgs ...string) string {
			out, err := runHelmTemplate(setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		renderChartExpectFailure := func(setArgs ...string) string {
			out, err := runHelmTemplate(setArgs...)
			Expect(err).To(HaveOccurred(), "helm template should have failed, got: %s", out)
			return out
		}

		// Anchor on the line start so a binding subject ("- kind: ServiceAccount") is not
		// counted as a created ServiceAccount resource.
		hasServiceAccountManifest := func(rendered string) bool {
			return strings.HasPrefix(rendered, "kind: ServiceAccount\n") ||
				strings.Contains(rendered, "\nkind: ServiceAccount\n")
		}

		saSubjectNames := func(rendered string) []string {
			re := regexp.MustCompile(`- kind: ServiceAccount\s+name:\s+(\S+)`)
			var names []string
			for _, m := range re.FindAllStringSubmatch(rendered, -1) {
				names = append(names, m[1])
			}
			return names
		}

		// Match "kind:" at the line start so a query for RoleBinding never matches ClusterRoleBinding.
		countKind := func(rendered, kind string) int {
			return strings.Count(rendered, "\nkind: "+kind+"\n")
		}

		DescribeTable("resolves serviceAccountName across every enabled/name combination",
			func(setArgs []string, wantName string, wantManifest bool) {
				rendered := renderChart(setArgs...)

				By("the Deployment references the expected ServiceAccount")
				Expect(rendered).To(ContainSubstring("serviceAccountName: " + wantName))

				By("every RBAC binding subject resolves to that same ServiceAccount")
				subjects := saSubjectNames(rendered)
				Expect(subjects).NotTo(BeEmpty())
				Expect(subjects).To(HaveEach(wantName))

				By("the ServiceAccount manifest is created only when the chart owns the SA")
				Expect(hasServiceAccountManifest(rendered)).To(Equal(wantManifest))
			},
			Entry("defaults (enabled=true, no name): chart creates and uses the generated SA",
				[]string{}, generatedName, true),
			Entry("enabled=true with a custom name: name ignored, chart owns the SA",
				[]string{"--set", "serviceAccount.enabled=true", "--set", "serviceAccount.name=custom-sa"},
				generatedName, true),
			Entry("enabled=false with a name: uses the external SA, creates none",
				[]string{"--set", "serviceAccount.enabled=false", "--set", "serviceAccount.name=external-sa"},
				"external-sa", false),
			Entry("enabled=false with name=default: opts into the namespace default SA, creates none",
				[]string{"--set", "serviceAccount.enabled=false", "--set", "serviceAccount.name=default"},
				"default", false),
			Entry("enabled null with a name: behaves as disabled, uses that name, creates none",
				[]string{"--set", "serviceAccount.enabled=null", "--set", "serviceAccount.name=external-sa"},
				"external-sa", false),
		)

		// Failing at render time prevents silently binding operator RBAC to the shared default SA.
		DescribeTable("fails to render when the ServiceAccount is disabled and no name is set",
			func(setArgs []string) {
				out := renderChartExpectFailure(setArgs...)
				Expect(out).To(ContainSubstring(
					"serviceAccount.name is required when serviceAccount.enabled=false"))
				Expect(out).To(ContainSubstring(
					"set name: default explicitly to use the namespace default ServiceAccount"))
			},
			Entry("name unset", []string{"--set", "serviceAccount.enabled=false"}),
			Entry("name empty", []string{"--set", "serviceAccount.enabled=false", "--set", "serviceAccount.name="}),
			Entry("name null", []string{"--set", "serviceAccount.enabled=false", "--set", "serviceAccount.name=null"}),
			Entry("toggle null, no name: behaves as disabled", []string{"--set", "serviceAccount.enabled=null"}),
		)

		DescribeTable("keeps every binding subject consistent with the Deployment across RBAC scope modes",
			func(setArgs []string, wantName string) {
				rendered := renderChart(setArgs...)

				Expect(rendered).To(ContainSubstring("serviceAccountName: " + wantName))
				subjects := saSubjectNames(rendered)
				Expect(subjects).NotTo(BeEmpty())
				Expect(subjects).To(HaveEach(wantName))
			},
			Entry("cluster-scoped, default SA", []string{}, generatedName),
			Entry("namespaced, default SA",
				[]string{"--set", "rbac.namespaced=true"}, generatedName),
			Entry("cluster-scoped, external SA",
				[]string{"--set", "serviceAccount.enabled=false", "--set", "serviceAccount.name=external-sa"},
				"external-sa"),
			Entry("namespaced, external SA",
				[]string{
					"--set", "rbac.namespaced=true",
					"--set", "serviceAccount.enabled=false", "--set", "serviceAccount.name=external-sa",
				},
				"external-sa"),
			Entry("cluster-scoped, explicit name=default",
				[]string{"--set", "serviceAccount.enabled=false", "--set", "serviceAccount.name=default"}, "default"),
			Entry("namespaced, explicit name=default",
				[]string{
					"--set", "rbac.namespaced=true",
					"--set", "serviceAccount.enabled=false", "--set", "serviceAccount.name=default",
				},
				"default"),
		)

		// Scope rules from helm-v2-alpha.md: manager bindings switch with rbac.namespaced,
		// leader-election is always a RoleBinding, metrics-auth is always a ClusterRoleBinding.
		DescribeTable("applies the documented RBAC scope rules for SA-bearing bindings",
			func(setArgs []string, wantClusterRoleBindings, wantRoleBindings int) {
				rendered := renderChart(setArgs...)

				By("binding kinds follow the documented cluster/namespaced rules")
				Expect(countKind(rendered, "ClusterRoleBinding")).To(Equal(wantClusterRoleBindings))
				Expect(countKind(rendered, "RoleBinding")).To(Equal(wantRoleBindings))

				By("whatever bindings render, their SA subjects stay consistent")
				Expect(saSubjectNames(rendered)).To(HaveEach(generatedName))
			},
			Entry("cluster-scoped, metrics off: manager CRB + leader-election RB",
				[]string{}, 1, 1),
			Entry("namespaced, metrics off: manager switches to RB, leader-election RB, no CRB",
				[]string{"--set", "rbac.namespaced=true"}, 0, 2),
			Entry("cluster-scoped, metrics on: manager CRB + metrics-auth CRB",
				[]string{"--set", "metrics.enabled=true", "--set", "metrics.secure=true"}, 2, 1),
			Entry("namespaced, metrics on: metrics-auth stays CRB, manager+leader-election RB",
				[]string{"--set", "rbac.namespaced=true", "--set", "metrics.enabled=true", "--set", "metrics.secure=true"},
				1, 2),
		)

		It("never leaks the generated name when an external SA is selected", func() {
			rendered := renderChart("--set", "serviceAccount.enabled=false", "--set", "serviceAccount.name=external-sa")
			Expect(rendered).NotTo(ContainSubstring("serviceAccountName: " + generatedName))
		})

		It("never leaks a stale custom name while the chart manages its own SA", func() {
			rendered := renderChart("--set", "serviceAccount.enabled=true", "--set", "serviceAccount.name=custom-sa")
			Expect(rendered).NotTo(ContainSubstring("serviceAccountName: custom-sa"))
		})
	})

	// When the source ServiceAccount already carries annotations, Kustomize lists annotations before
	// labels. The generator must merge into that block; a second annotations key makes the manifest
	// invalid YAML and fails `helm template`.
	Context("ServiceAccount annotations (rendered)", func() {
		renderChart := func(setArgs ...string) string {
			out, err := helmTemplate(createKustomizeForServiceAccountWithAnnotationsRender("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		It("merges into the existing annotations block instead of duplicating it", func() {
			sa := serviceAccountDoc(renderChart())

			Expect(sa).NotTo(BeEmpty(), "expected a ServiceAccount manifest in the render")
			Expect(strings.Count(sa, "annotations:")).To(Equal(1),
				"ServiceAccount must keep a single annotations: block, got:\n%s", sa)
			Expect(sa).To(ContainSubstring("example.com/existing-annotation: preserved-value"))
		})

		It("renders user-supplied annotations alongside the scaffolded one", func() {
			sa := serviceAccountDoc(renderChart("--set", "serviceAccount.annotations.team=platform"))

			Expect(strings.Count(sa, "annotations:")).To(Equal(1))
			Expect(sa).To(ContainSubstring("example.com/existing-annotation: preserved-value"))
			Expect(sa).To(ContainSubstring("team: platform"))
		})
	})

	Context("ServiceAccount labels (rendered)", func() {
		renderChart := func(setArgs ...string) string {
			out, err := helmTemplate(createKustomizeForServiceAccountRender("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		It("renders user-supplied labels alongside the scaffolded ones", func() {
			sa := serviceAccountDoc(renderChart("--set", "serviceAccount.labels.env=prod"))

			Expect(sa).NotTo(BeEmpty(), "expected a ServiceAccount manifest in the render")
			Expect(strings.Count(sa, "labels:")).To(Equal(1),
				"ServiceAccount must keep a single labels: block, got:\n%s", sa)
			Expect(sa).To(ContainSubstring("env: prod"))
		})

		It("does not duplicate a scaffolded label a user tries to override", func() {
			sa := serviceAccountDoc(renderChart(
				"--set", "serviceAccount.labels.app\\.kubernetes\\.io/name=override"))

			Expect(strings.Count(sa, "app.kubernetes.io/name:")).To(Equal(1),
				"a user override must not add a second app.kubernetes.io/name label, got:\n%s", sa)
		})
	})

	// serviceMonitorDoc returns the ServiceMonitor document from a multi-document render,
	// or "" when it was not rendered.
	serviceMonitorDoc := func(rendered string) string {
		for _, doc := range strings.Split(rendered, "\n---") {
			if strings.Contains(doc, "\nkind: ServiceMonitor\n") {
				return doc
			}
		}
		return ""
	}

	Context("ServiceMonitor labels and annotations (rendered, kustomize-derived)", func() {
		// The ServiceMonitor is gated on metrics.enabled, which the fixtures leave off.
		renderChart := func(setArgs ...string) string {
			args := append([]string{"--set", "metrics.enabled=true"}, setArgs...)
			out, err := helmTemplate(createKustomizeWithServiceMonitor("test-project"), args...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		It("renders user-supplied labels on the kustomize-derived ServiceMonitor", func() {
			sm := serviceMonitorDoc(renderChart(
				"--set", "prometheus.enabled=true",
				"--set", "prometheus.labels.team=platform",
			))

			Expect(sm).NotTo(BeEmpty(), "expected a ServiceMonitor manifest in the render")
			Expect(strings.Count(sm, "labels:")).To(Equal(1),
				"ServiceMonitor must keep a single labels: block, got:\n%s", sm)
			Expect(sm).To(ContainSubstring("team: platform"))
		})

		It("renders user-supplied annotations on the kustomize-derived ServiceMonitor", func() {
			sm := serviceMonitorDoc(renderChart(
				"--set", "prometheus.enabled=true",
				"--set", "prometheus.annotations.example\\.com/env=staging",
			))

			Expect(sm).NotTo(BeEmpty(), "expected a ServiceMonitor manifest in the render")
			Expect(strings.Count(sm, "annotations:")).To(Equal(1),
				"ServiceMonitor must keep a single annotations: block, got:\n%s", sm)
			Expect(sm).To(ContainSubstring("example.com/env: staging"))
		})

		It("does not duplicate a scaffolded label a user tries to override", func() {
			sm := serviceMonitorDoc(renderChart(
				"--set", "prometheus.enabled=true",
				"--set", "prometheus.labels.control-plane=myvalue",
			))

			Expect(sm).NotTo(BeEmpty(), "expected a ServiceMonitor manifest in the render")
			// omit strips keys already defined by the scaffolded labels block, so
			// the user value must not appear; only the scaffolded value remains.
			Expect(sm).NotTo(ContainSubstring("control-plane: myvalue"),
				"the omit guard must prevent duplicating a scaffolded label, got:\n%s", sm)
		})

		It("does not render the ServiceMonitor when metrics are disabled", func() {
			sm := serviceMonitorDoc(renderChart(
				"--set", "metrics.enabled=false",
				"--set", "prometheus.enabled=true",
			))

			Expect(sm).To(BeEmpty(),
				"the ServiceMonitor scrapes the metrics Service, so it must not render without it")
		})
	})

	Context("ServiceMonitor labels and annotations (rendered, static fallback)", func() {
		// No ServiceMonitor in the kustomize input — the static fallback template is scaffolded.
		// The ServiceMonitor is gated on metrics.enabled, which the fixture leaves off.
		renderChart := func(setArgs ...string) string {
			args := append([]string{"--set", "metrics.enabled=true"}, setArgs...)
			out, err := helmTemplate(createBasicKustomizeOutput("test-project"), args...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		It("renders user-supplied labels on the static fallback ServiceMonitor", func() {
			sm := serviceMonitorDoc(renderChart(
				"--set", "prometheus.enabled=true",
				"--set", "prometheus.labels.team=platform",
			))

			Expect(sm).NotTo(BeEmpty(), "expected a ServiceMonitor manifest in the render")
			Expect(strings.Count(sm, "labels:")).To(Equal(1),
				"ServiceMonitor must keep a single labels: block, got:\n%s", sm)
			Expect(sm).To(ContainSubstring("team: platform"))
		})

		It("renders user-supplied annotations on the static fallback ServiceMonitor", func() {
			sm := serviceMonitorDoc(renderChart(
				"--set", "prometheus.enabled=true",
				"--set", "prometheus.annotations.example\\.com/env=staging",
			))

			Expect(sm).NotTo(BeEmpty(), "expected a ServiceMonitor manifest in the render")
			Expect(strings.Count(sm, "annotations:")).To(Equal(1),
				"ServiceMonitor must keep a single annotations: block, got:\n%s", sm)
			Expect(sm).To(ContainSubstring("example.com/env: staging"))
		})

		It("does not duplicate a scaffolded label a user tries to override", func() {
			sm := serviceMonitorDoc(renderChart(
				"--set", "prometheus.enabled=true",
				"--set", "prometheus.labels.control-plane=myvalue",
			))

			Expect(sm).NotTo(BeEmpty(), "expected a ServiceMonitor manifest in the render")
			// omit strips keys already defined by the scaffolded labels block, so
			// the user value must not appear; only the scaffolded value remains.
			Expect(sm).NotTo(ContainSubstring("control-plane: myvalue"),
				"the omit guard must prevent duplicating a scaffolded label, got:\n%s", sm)
		})

		It("does not render the ServiceMonitor when metrics are disabled", func() {
			sm := serviceMonitorDoc(renderChart(
				"--set", "metrics.enabled=false",
				"--set", "prometheus.enabled=true",
			))

			Expect(sm).To(BeEmpty(),
				"the ServiceMonitor scrapes the metrics Service, so it must not render without it")
		})
	})

	Context("NetworkPolicy (rendered)", func() {
		// networkPolicyDoc returns the NetworkPolicy document whose name ends with the given
		// suffix from a multi-document render, or "" when it was not rendered.
		networkPolicyDoc := func(rendered, suffix string) string {
			for _, doc := range strings.Split(rendered, "\n---") {
				if strings.Contains(doc, "\nkind: NetworkPolicy\n") && strings.Contains(doc, suffix) {
					return doc
				}
			}
			return ""
		}

		renderWithNetworkPolicies := func(setArgs ...string) string {
			out, err := helmTemplate(createKustomizeWithWebhooks("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		It("renders both policies with the metrics and webhook ports when enabled", func() {
			rendered := renderWithNetworkPolicies(
				"--set", "networkPolicy.enabled=true",
				"--set", "metrics.enabled=true",
				"--set", "webhook.enabled=true",
			)

			metricsPolicy := networkPolicyDoc(rendered, "allow-metrics-traffic")
			Expect(metricsPolicy).NotTo(BeEmpty(), "metrics NetworkPolicy should be rendered")
			Expect(metricsPolicy).To(ContainSubstring("metrics: enabled"))
			Expect(metricsPolicy).To(ContainSubstring("port: 8443"))

			webhookPolicy := networkPolicyDoc(rendered, "allow-webhook-traffic")
			Expect(webhookPolicy).NotTo(BeEmpty(), "webhook NetworkPolicy should be rendered")
			Expect(webhookPolicy).To(ContainSubstring("port: 9443"))
		})

		DescribeTable("renders no NetworkPolicy when networkPolicy.enabled is false",
			func(metricsEnabled, webhookEnabled string) {
				rendered := renderWithNetworkPolicies(
					"--set", "networkPolicy.enabled=false",
					"--set", "metrics.enabled="+metricsEnabled,
					"--set", "webhook.enabled="+webhookEnabled,
				)

				Expect(rendered).NotTo(ContainSubstring("kind: NetworkPolicy"))
			},
			Entry("with metrics and webhook enabled", "true", "true"),
			Entry("with only metrics enabled", "true", "false"),
			Entry("with only webhook enabled", "false", "true"),
			Entry("with both disabled", "false", "false"),
		)

		It("suppresses the metrics policy when metrics are disabled but keeps the webhook policy", func() {
			rendered := renderWithNetworkPolicies(
				"--set", "networkPolicy.enabled=true",
				"--set", "metrics.enabled=false",
				"--set", "webhook.enabled=true",
			)

			Expect(networkPolicyDoc(rendered, "allow-metrics-traffic")).To(BeEmpty(),
				"metrics NetworkPolicy must be gated by metrics.enabled")
			Expect(networkPolicyDoc(rendered, "allow-webhook-traffic")).NotTo(BeEmpty(),
				"webhook NetworkPolicy should still render")
		})

		It("suppresses the webhook policy when webhooks are disabled but keeps the metrics policy", func() {
			rendered := renderWithNetworkPolicies(
				"--set", "networkPolicy.enabled=true",
				"--set", "metrics.enabled=true",
				"--set", "webhook.enabled=false",
			)

			Expect(networkPolicyDoc(rendered, "allow-webhook-traffic")).To(BeEmpty(),
				"webhook NetworkPolicy must be gated by webhook.enabled")
			Expect(networkPolicyDoc(rendered, "allow-metrics-traffic")).NotTo(BeEmpty(),
				"metrics NetworkPolicy should still render")
		})

		It("renders no NetworkPolicy when networkPolicy is enabled but metrics and webhooks are disabled", func() {
			rendered := renderWithNetworkPolicies(
				"--set", "networkPolicy.enabled=true",
				"--set", "metrics.enabled=false",
				"--set", "webhook.enabled=false",
			)

			Expect(rendered).NotTo(ContainSubstring("kind: NetworkPolicy"),
				"both policies must be gated off when their features are disabled")
		})

		It("tracks custom metrics and webhook ports in the rendered policies", func() {
			rendered := renderWithNetworkPolicies(
				"--set", "networkPolicy.enabled=true",
				"--set", "metrics.enabled=true", "--set", "metrics.port=9000",
				"--set", "webhook.enabled=true", "--set", "webhook.port=9001",
			)

			Expect(networkPolicyDoc(rendered, "allow-metrics-traffic")).To(ContainSubstring("port: 9000"))
			Expect(networkPolicyDoc(rendered, "allow-webhook-traffic")).To(ContainSubstring("port: 9001"))
		})
	})

	Context("Disabled admission webhooks (rendered)", func() {
		It("does not render webhook resources when cert-manager stays enabled", func() {
			rendered, err := helmTemplate(
				createKustomizeWithWebhooksAndCertManager("test-project"),
				"--set", "webhook.enabled=false",
				"--set", "certManager.enabled=true",
				"--set", "networkPolicy.enabled=true",
			)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", rendered)

			Expect(rendered).NotTo(ContainSubstring("kind: ValidatingWebhookConfiguration"))
			Expect(rendered).NotTo(ContainSubstring("-serving-cert"))
			Expect(rendered).NotTo(ContainSubstring("webhook-server-cert"))
			Expect(rendered).NotTo(ContainSubstring("-webhook-service"))
			Expect(rendered).NotTo(ContainSubstring("allow-webhook-traffic"))
			Expect(rendered).To(ContainSubstring("kind: Issuer"),
				"cert-manager resources unrelated to the disabled webhook should remain")
		})
	})

	Context("NetworkPolicy conversion from kustomize (rendered)", func() {
		networkPolicyDoc := func(rendered, suffix string) string {
			for _, doc := range strings.Split(rendered, "\n---") {
				if strings.Contains(doc, "\nkind: NetworkPolicy\n") && strings.Contains(doc, suffix) {
					return doc
				}
			}
			return ""
		}

		renderConverted := func(setArgs ...string) string {
			out, err := helmTemplate(createKustomizeWithWebhooksAndNetworkPolicies("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		It("renders all converted policies with their gates and ports when everything is enabled", func() {
			rendered := renderConverted(
				"--set", "networkPolicy.enabled=true",
				"--set", "metrics.enabled=true",
				"--set", "webhook.enabled=true",
			)

			Expect(networkPolicyDoc(rendered, "-allow-metrics-traffic")).To(ContainSubstring("port: 8443"))
			Expect(networkPolicyDoc(rendered, "-allow-webhook-traffic")).To(ContainSubstring("port: 9443"))
			Expect(networkPolicyDoc(rendered, "allow-dns-traffic")).To(ContainSubstring("port: 5353"))
			Expect(networkPolicyDoc(rendered, "disallow-metrics-traffic")).To(ContainSubstring("port: 7777"),
				"a policy whose name only resembles the scaffolded ones must keep its port")
		})

		It("renders no converted policy when networkPolicy.enabled is false", func() {
			rendered := renderConverted(
				"--set", "networkPolicy.enabled=false",
				"--set", "metrics.enabled=true",
				"--set", "webhook.enabled=true",
			)

			Expect(rendered).NotTo(ContainSubstring("kind: NetworkPolicy"))
		})

		It("gates only the scaffolded policies on their feature toggles", func() {
			rendered := renderConverted(
				"--set", "networkPolicy.enabled=true",
				"--set", "metrics.enabled=false",
				"--set", "webhook.enabled=false",
			)

			Expect(networkPolicyDoc(rendered, "-allow-metrics-traffic")).To(BeEmpty(),
				"converted metrics policy must be gated by metrics.enabled")
			Expect(networkPolicyDoc(rendered, "-allow-webhook-traffic")).To(BeEmpty(),
				"converted webhook policy must be gated by webhook.enabled")
			Expect(networkPolicyDoc(rendered, "allow-dns-traffic")).NotTo(BeEmpty(),
				"custom policies must only be gated by networkPolicy.enabled")
			Expect(networkPolicyDoc(rendered, "disallow-metrics-traffic")).NotTo(BeEmpty(),
				"a policy whose name only resembles the scaffolded ones must only be gated by networkPolicy.enabled")
		})

		It("renders a conversion webhook chart and rejects disabling its webhook", func() {
			out, err := helmTemplate(createKustomizeConversionOnlyWithNetworkPolicies("test-project"))
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)

			var crd string
			for _, doc := range strings.Split(out, "\n---") {
				if strings.Contains(doc, "\nkind: CustomResourceDefinition\n") {
					crd = doc
					break
				}
			}
			Expect(crd).NotTo(BeEmpty(), "the conversion CRD must render")
			Expect(crd).To(ContainSubstring("strategy: Webhook"))
			Expect(crd).To(ContainSubstring("name: my-release-test-project-webhook-service"))
			Expect(crd).To(ContainSubstring("namespace: my-namespace"))
			Expect(networkPolicyDoc(out, "-allow-webhook-traffic")).To(ContainSubstring("port: 9443"),
				"the webhook NetworkPolicy must render with webhook.enabled defaulting to true")

			workflow, err := os.ReadFile(filepath.Join(tmpDir, ".github", "workflows", "test-chart.yml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(workflow)).To(ContainSubstring(
				"      - name: Install cert-manager via Helm (wait for readiness)"),
				"conversion webhooks from kustomize output must enable cert-manager in the workflow")
			Expect(string(workflow)).NotTo(ContainSubstring(
				"#      - name: Install cert-manager via Helm (wait for readiness)"))

			out, err = helmTemplate(createKustomizeConversionOnlyWithNetworkPolicies("test-project"),
				"--set", "webhook.enabled=false")
			Expect(err).To(HaveOccurred(), "helm template should reject disabling a required webhook: %s", out)
			Expect(out).To(ContainSubstring(
				"webhook.enabled must be true for CRD conversion"))

			out, err = helmTemplate(createKustomizeConversionOnlyWithNetworkPolicies("test-project"),
				"--set", "crd.enabled=false", "--set", "webhook.enabled=false")
			Expect(err).NotTo(HaveOccurred(),
				"helm template should allow disabling an unrendered conversion CRD: %s", out)
		})

		It("tracks custom ports in the converted scaffolded policies only", func() {
			rendered := renderConverted(
				"--set", "networkPolicy.enabled=true",
				"--set", "metrics.enabled=true", "--set", "metrics.port=9100",
				"--set", "webhook.enabled=true", "--set", "webhook.port=9101",
			)

			Expect(networkPolicyDoc(rendered, "-allow-metrics-traffic")).To(ContainSubstring("port: 9100"))
			Expect(networkPolicyDoc(rendered, "-allow-webhook-traffic")).To(ContainSubstring("port: 9101"))
			Expect(networkPolicyDoc(rendered, "allow-dns-traffic")).To(ContainSubstring("port: 5353"))
			Expect(networkPolicyDoc(rendered, "disallow-metrics-traffic")).To(ContainSubstring("port: 7777"))
		})
	})

	Context("Custom Output Directory", func() {
		It("should support custom output directory via --output-dir flag", func() {
			kustomizeYAML := createBasicKustomizeOutput("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			customOutputDir := "custom-charts"
			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, customOutputDir)
			scaffolderBase.InjectFS(fs)

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

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)
		})
	})

	Context("Values Extraction", func() {
		It("should extract deployment configuration to values.yaml", func() {
			kustomizeYAML := createKustomizeWithFullDeploymentConfig("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

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

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)
		})
	})

	// Validates the full pipeline for deployments with custom volumes alongside system volumes.
	// Custom volumes must appear only via extraVolumes template; system volumes remain literal.
	Context("Custom volumes deduplication", func() {
		It("should render custom volumes only through extraVolumes template, not as literal entries", func() {
			kustomizeYAML := createKustomizeWithCustomVolumes("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")
			managerTemplatePath := filepath.Join(chartPath, "templates", "manager", "manager.yaml")

			By("reading the generated manager template")
			managerBytes, err := os.ReadFile(managerTemplatePath)
			Expect(err).NotTo(HaveOccurred())
			managerStr := string(managerBytes)

			By("verifying extraVolumes appears via Helm template")
			Expect(managerStr).To(ContainSubstring(".Values.manager.extraVolumes"),
				"manager template must reference extraVolumes from values")

			By("verifying extraVolumeMounts appears via Helm template")
			Expect(managerStr).To(ContainSubstring(".Values.manager.extraVolumeMounts"),
				"manager template must reference extraVolumeMounts from values")

			By("verifying no literal custom volume entries remain in the template")
			Expect(managerStr).NotTo(ContainSubstring("app-config"),
				"custom volume name must not appear as literal entry in manager template")
			Expect(managerStr).NotTo(ContainSubstring("app-secret"),
				"custom volume name must not appear as literal entry in manager template")

			By("verifying system volumes still appear in the template")
			Expect(managerStr).To(ContainSubstring("webhook-certs"),
				"system volume webhook-certs must remain in manager template")
			Expect(managerStr).To(ContainSubstring("metrics-certs"),
				"system volume metrics-certs must remain in manager template")

			By("verifying custom volumes are extracted to values.yaml")
			valuesPath := filepath.Join(chartPath, "values.yaml")
			valuesBytes, err := os.ReadFile(valuesPath)
			Expect(err).NotTo(HaveOccurred())
			valuesStr := string(valuesBytes)
			Expect(valuesStr).To(ContainSubstring("extraVolumes:"),
				"extraVolumes must be present in values.yaml")
			Expect(valuesStr).To(ContainSubstring("extraVolumeMounts:"),
				"extraVolumeMounts must be present in values.yaml")
			Expect(valuesStr).To(ContainSubstring("app-config"),
				"custom volume app-config must be in values.yaml")

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)
		})
	})

	// Validates the pipeline when only custom volumes exist (no system volumes like webhook-certs
	// or metrics-certs). All volumes should be extracted to values and stripped from the template.
	Context("Custom volumes without system volumes", func() {
		It("should extract all volumes to values and leave none as literal entries", func() {
			kustomizeYAML := createKustomizeWithCustomVolumesOnly("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")
			managerTemplatePath := filepath.Join(chartPath, "templates", "manager", "manager.yaml")

			By("reading the generated manager template")
			managerBytes, err := os.ReadFile(managerTemplatePath)
			Expect(err).NotTo(HaveOccurred())
			managerStr := string(managerBytes)

			By("verifying extraVolumes appears via Helm template")
			Expect(managerStr).To(ContainSubstring(".Values.manager.extraVolumes"),
				"manager template must reference extraVolumes from values")

			By("verifying extraVolumeMounts appears via Helm template")
			Expect(managerStr).To(ContainSubstring(".Values.manager.extraVolumeMounts"),
				"manager template must reference extraVolumeMounts from values")

			By("verifying no literal custom volume entries remain in the template")
			Expect(managerStr).NotTo(ContainSubstring("app-config"),
				"custom volume name must not appear as literal entry in manager template")
			Expect(managerStr).NotTo(ContainSubstring("app-secret"),
				"custom volume name must not appear as literal entry in manager template")

			By("verifying custom volumes are extracted to values.yaml")
			valuesPath := filepath.Join(chartPath, "values.yaml")
			valuesBytes, err := os.ReadFile(valuesPath)
			Expect(err).NotTo(HaveOccurred())
			valuesStr := string(valuesBytes)
			Expect(valuesStr).To(ContainSubstring("extraVolumes:"),
				"extraVolumes must be present in values.yaml")
			Expect(valuesStr).To(ContainSubstring("extraVolumeMounts:"),
				"extraVolumeMounts must be present in values.yaml")

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)
		})
	})

	// Validates the pipeline when a sidecar container appears before the manager and the
	// default-container annotation identifies which container is the manager.
	Context("Sidecar before manager with default-container annotation", func() {
		It("should extract and strip volumes from the correct container", func() {
			kustomizeYAML := createKustomizeWithSidecarBeforeManager("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")
			managerTemplatePath := filepath.Join(chartPath, "templates", "manager", "manager.yaml")

			By("reading the generated manager template")
			managerBytes, err := os.ReadFile(managerTemplatePath)
			Expect(err).NotTo(HaveOccurred())
			managerStr := string(managerBytes)

			By("verifying custom volumes only appear via Helm template")
			Expect(managerStr).To(ContainSubstring(".Values.manager.extraVolumes"))
			Expect(managerStr).NotTo(ContainSubstring("app-config"),
				"custom volume must not appear as literal entry")

			By("verifying system volumes remain in the template")
			Expect(managerStr).To(ContainSubstring("webhook-certs"))

			By("verifying values.yaml has the manager's extraVolumes")
			valuesPath := filepath.Join(chartPath, "values.yaml")
			valuesBytes, err := os.ReadFile(valuesPath)
			Expect(err).NotTo(HaveOccurred())
			valuesStr := string(valuesBytes)
			Expect(valuesStr).To(ContainSubstring("extraVolumes:"))
			Expect(valuesStr).To(ContainSubstring("app-config"))

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)
		})

		It("should template manager container fields and leave sidecar fields as literals", func() {
			kustomizeYAML := createKustomizeWithSidecarBeforeManager("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")
			managerTemplatePath := filepath.Join(chartPath, "templates", "manager", "manager.yaml")

			By("reading the generated manager template")
			managerBytes, err := os.ReadFile(managerTemplatePath)
			Expect(err).NotTo(HaveOccurred())
			managerStr := string(managerBytes)

			By("verifying the manager's image is templated")
			Expect(managerStr).To(ContainSubstring(".Values.manager.image.repository"))

			By("verifying the sidecar's image remains as a literal")
			Expect(managerStr).To(ContainSubstring("image: sidecar:v1"))

			By("verifying the sidecar's env remains as a literal")
			Expect(managerStr).To(ContainSubstring("SIDECAR_MODE"))

			By("verifying the sidecar's resources remain as literals")
			Expect(managerStr).To(ContainSubstring("cpu: 100m"),
				"sidecar-unique resource value must remain literal")
			Expect(managerStr).To(ContainSubstring("memory: 32Mi"),
				"sidecar-unique memory value must remain literal")

			By("verifying the sidecar's securityContext remains as a literal")
			Expect(managerStr).To(ContainSubstring("runAsNonRoot: true"),
				"sidecar securityContext must not be templated")

			By("verifying the manager's resources are templated")
			Expect(managerStr).To(ContainSubstring(".Values.manager.resources"))

			By("verifying the manager's env is templated")
			Expect(managerStr).To(ContainSubstring(".Values.manager.env"))

			By("verifying the manager's args are templated")
			Expect(managerStr).To(ContainSubstring(".Values.manager.args"))

			By("verifying the chart loads cleanly")
			chart, err := helmChartLoader.LoadDir(chartPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(chart.Validate()).To(Succeed())
		})
	})

	// A project that already has hand-authored tolerations, nodeSelector, and affinity in
	// its manager Deployment (typical of a project created before the Helm plugin was added)
	// must produce a chart where each scheduling field appears exactly once in
	// templates/manager/manager.yaml as a Helm-templated stanza only, without any leftover
	// raw YAML list items that would cause duplicate blocks and Helm parse errors.
	Context("Scheduling fields upgrade-path", func() {
		It("should produce exactly one Helm-templated stanza per scheduling field with no raw remnants", func() {
			kustomizeYAML := createKustomizeWithTolerationsAndSchedulingFields("test-project")
			err := setupKustomizeFile(manifestsFile, kustomizeYAML)
			Expect(err).NotTo(HaveOccurred())

			scaffolderBase = scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolderBase.InjectFS(fs)

			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(tmpDir, outputDir, "chart")
			managerTemplatePath := filepath.Join(chartPath, "templates", "manager", "manager.yaml")

			By("reading the generated manager template")
			managerBytes, err := os.ReadFile(managerTemplatePath)
			Expect(err).NotTo(HaveOccurred())
			managerStr := string(managerBytes)

			By("verifying tolerations appears exactly once and is Helm-templated")
			Expect(strings.Count(managerStr, "tolerations:")).To(Equal(1),
				"tolerations: must appear exactly once in the manager template")
			Expect(managerStr).To(ContainSubstring("{{- with .Values.manager.tolerations }}"),
				"manager template must contain Helm with-block for tolerations")
			Expect(managerStr).To(ContainSubstring("tolerations: {{ toYaml . | nindent"),
				"manager template must use toYaml for tolerations")

			By("verifying no raw toleration list items remain")
			Expect(managerStr).NotTo(ContainSubstring("effect: NoSchedule"),
				"raw toleration effect must not remain in manager template")
			Expect(managerStr).NotTo(ContainSubstring("key: node-role.kubernetes.io/control-plane"),
				"raw toleration key must not remain in manager template")
			Expect(managerStr).NotTo(ContainSubstring("key: dedicated"),
				"raw toleration key must not remain in manager template")

			By("verifying nodeSelector appears exactly once and is Helm-templated")
			Expect(strings.Count(managerStr, "nodeSelector:")).To(Equal(1),
				"nodeSelector: must appear exactly once in the manager template")
			Expect(managerStr).To(ContainSubstring("{{- with .Values.manager.nodeSelector }}"))
			Expect(managerStr).NotTo(ContainSubstring("kubernetes.io/os: linux"),
				"raw nodeSelector entry must not remain in the manager template")

			By("verifying affinity appears exactly once and is Helm-templated")
			Expect(strings.Count(managerStr, "affinity:")).To(Equal(1),
				"affinity: must appear exactly once in the manager template")
			Expect(managerStr).To(ContainSubstring("{{- with .Values.manager.affinity }}"))
			Expect(managerStr).NotTo(ContainSubstring("nodeAffinity:"),
				"raw affinity sub-field must not remain in the manager template")

			By("verifying tolerations are extracted to values.yaml")
			valuesPath := filepath.Join(chartPath, "values.yaml")
			valuesBytes, err := os.ReadFile(valuesPath)
			Expect(err).NotTo(HaveOccurred())
			valuesStr := string(valuesBytes)
			Expect(valuesStr).To(ContainSubstring("tolerations:"),
				"tolerations must be extracted to values.yaml")

			By("linting the generated chart")
			lintResult := action.NewLint().Run([]string{chartPath}, nil)
			Expect(lintResult.Errors).To(BeEmpty(), "helm lint failed: %v", lintResult.Errors)
		})
	})

	// The `tpl` change in templateControllerManagerArgs evaluates each manager.args entry as a
	// Helm template (`{{ tpl . $ }}`) instead of rendering it verbatim (`{{ . }}`). These specs
	// render the chart with `helm template` to prove every combination resolves correctly: the
	// chart's own default extracted args (no values override), plain literal args (backwards
	// compatibility), a single templated arg, several templated args together, templated and
	// literal args mixed in the same list, and a templated arg that calls a Helm template
	// function.
	Context("Manager args templating (rendered)", func() {
		writeValuesFile := func(valuesContent string) string {
			valuesFile := filepath.Join(tmpDir, "manager-args-values.yaml")
			Expect(os.WriteFile(valuesFile, []byte(valuesContent), 0o600)).To(Succeed())
			return valuesFile
		}

		// renderWithArgs scaffolds the chart and renders it with `helm template`. When
		// valuesContent is empty, no `-f` override is passed, so the chart renders with its own
		// generated values.yaml (i.e. the default manager.args extracted from the kustomize
		// output), exercising the same code path production users hit before ever touching
		// manager.args themselves.
		renderWithManagerArgs := func(valuesContent string) string {
			var setArgs []string
			if valuesContent != "" {
				setArgs = []string{"-f", writeValuesFile(valuesContent)}
			}
			out, err := helmTemplate(createKustomizeWithFullDeploymentConfig("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		renderedManagerArgs := func(renderedManifest string) []string {
			managerArgs, err := renderedContainerArgs(renderedManifest, "manager")
			Expect(err).NotTo(HaveOccurred())
			return managerArgs
		}

		It("should render the manager Deployment successfully when manager.args uses the chart's own "+
			"default values (no values override)", func() {
			rendered := renderWithManagerArgs("")

			By("the default extracted arg renders as a literal, unaffected by tpl")
			Expect(renderedManagerArgs(rendered)).To(ContainElement("--leader-elect"))
		})

		It("should connect metrics values to a manager that omits the metrics bind argument", func() {
			renderWithoutMetricsBindAddressArg := func(setArgs ...string) string {
				out, err := helmTemplate(
					createKustomizeWithMetricsServiceWithoutManagerArg("test-project"), setArgs...)
				Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
				return out
			}

			By("using the metrics Service port when metrics are enabled")
			rendered := renderWithoutMetricsBindAddressArg("--set", "metrics.enabled=true")
			renderedArgs := renderedManagerArgs(rendered)
			Expect(renderedArgs).To(ContainElement("--metrics-bind-address=:7443"))
			Expect(renderedArgs).NotTo(ContainElement("--metrics-bind-address=0"))
			Expect(renderedArgs).NotTo(ContainElement("--metrics-secure=false"))

			By("using an explicitly configured metrics port")
			rendered = renderWithoutMetricsBindAddressArg(
				"--set", "metrics.enabled=true",
				"--set", "metrics.port=9000",
			)
			renderedArgs = renderedManagerArgs(rendered)
			Expect(renderedArgs).To(ContainElement("--metrics-bind-address=:9000"))
			Expect(renderedArgs).NotTo(ContainElement("--metrics-bind-address=:7443"))

			By("disabling the manager metrics server when metrics are disabled")
			rendered = renderWithoutMetricsBindAddressArg("--set", "metrics.enabled=false")
			renderedArgs = renderedManagerArgs(rendered)
			Expect(renderedArgs).To(ContainElement("--metrics-bind-address=0"))
			Expect(renderedArgs).NotTo(ContainElement("--metrics-bind-address=:7443"))

			By("passing the secure setting to the manager")
			rendered = renderWithoutMetricsBindAddressArg(
				"--set", "metrics.enabled=true", "--set", "metrics.secure=false")
			renderedArgs = renderedManagerArgs(rendered)
			Expect(renderedArgs).To(ContainElement("--metrics-bind-address=:7443"))
			Expect(renderedArgs).To(ContainElement("--metrics-secure=false"))
		})

		It("should add metrics arguments when the source manager has no args field", func() {
			rendered, err := helmTemplate(
				createKustomizeWithMetricsServiceAndNoManagerArgs("test-project"),
				"--set", "metrics.enabled=true",
			)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", rendered)

			By("serving metrics on the Service port")
			Expect(renderedManagerArgs(rendered)).To(ContainElement("--metrics-bind-address=:7443"))
		})

		It("should add metrics arguments when the source manager has an empty args list", func() {
			rendered, err := helmTemplate(
				createKustomizeWithMetricsServiceAndEmptyManagerArgs("test-project"),
				"--set", "metrics.enabled=true",
			)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", rendered)

			By("serving metrics on the Service port")
			Expect(renderedManagerArgs(rendered)).To(ContainElement("--metrics-bind-address=:7443"))
		})

		It("should accept manager.args overrides when the source manager has no args field", func() {
			rendered, err := helmTemplate(
				createKustomizeWithMetricsServiceAndNoManagerArgs("test-project"),
				"--set", "manager.args[0]=--leader-elect",
			)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", rendered)

			By("keeping the user-provided argument alongside the generated metrics argument")
			Expect(renderedManagerArgs(rendered)).To(ContainElements(
				"--metrics-bind-address=:7443",
				"--leader-elect",
			))
		})

		It("should preserve sidecar arguments while configuring the manager container", func() {
			rendered, err := helmTemplate(
				createKustomizeWithSidecarBeforeManagerAndMetricsService("test-project"),
				"--set", "metrics.enabled=true",
			)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", rendered)

			By("configuring metrics on the manager")
			Expect(renderedManagerArgs(rendered)).To(ContainElement("--metrics-bind-address=:7443"))

			By("leaving the sidecar's command-line arguments unchanged")
			sidecarArgs, err := renderedContainerArgs(rendered, "sidecar")
			Expect(err).NotTo(HaveOccurred())
			Expect(sidecarArgs).To(Equal([]string{"--sidecar-flag"}))
		})

		DescribeTable("should resolve manager.args entries through tpl when values are provided",
			func(valuesContent string, additionalManagerArgs []string) {
				rendered := renderWithManagerArgs(valuesContent)

				expectedManagerArgs := append([]string{
					"--metrics-bind-address=0",
					"--health-probe-bind-address=:8081",
				}, additionalManagerArgs...)
				renderedArgs := renderedManagerArgs(rendered)
				Expect(renderedArgs).To(Equal(expectedManagerArgs), "rendered manager args: %v", renderedArgs)
			},
			Entry("should keep plain literal args unchanged when no template syntax is used (backwards compatible)",
				"manager:\n  args:\n  - --leader-elect\n  - --zap-log-level=info\n",
				[]string{"--leader-elect", "--zap-log-level=info"},
			),
			Entry("should resolve to the release namespace when an arg references .Release.Namespace",
				"manager:\n  args:\n  - --leader-election-namespace={{ .Release.Namespace }}\n",
				[]string{"--leader-election-namespace=my-namespace"},
			),
			Entry("should resolve every arg independently and keep list order when multiple args are templated",
				"manager:\n  args:\n"+
					"  - --leader-election-namespace={{ .Release.Namespace }}\n"+
					"  - --release-name={{ .Release.Name }}\n"+
					"  - --chart-name={{ .Chart.Name }}\n",
				[]string{
					"--leader-election-namespace=my-namespace",
					"--release-name=my-release",
					"--chart-name=test-project",
				},
			),
			Entry("should resolve templated args and keep literal args unchanged when both appear in the same list",
				"manager:\n  args:\n"+
					"  - --leader-elect\n"+
					"  - --leader-election-namespace={{ .Release.Namespace }}\n"+
					"  - --zap-log-level=info\n",
				[]string{
					"--leader-elect",
					"--leader-election-namespace=my-namespace",
					"--zap-log-level=info",
				},
			),
			Entry("should resolve an arg when it calls a Helm template function",
				`manager:
  args:
  - --extra-flag={{ printf "%s-%s" .Release.Name .Release.Namespace }}
`,
				[]string{"--extra-flag=my-release-my-namespace"},
			),
		)
	})

	Context("Metrics feature toggles (rendered)", func() {
		renderChart := func(setArgs ...string) string {
			out, err := helmTemplate(createKustomizeWithMetricsResources("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		It("should enable the manager and metrics resources together", func() {
			rendered := renderChart(
				"--set", "metrics.enabled=true",
				"--set", "prometheus.enabled=true",
				"--set", "networkPolicy.enabled=true",
			)

			By("binding the manager to the metrics Service port")
			managerArgs, err := renderedContainerArgs(rendered, "manager")
			Expect(err).NotTo(HaveOccurred())
			Expect(managerArgs).To(ContainElement("--metrics-bind-address=:7443"))

			By("rendering the metrics Service")
			metricsServices, err := renderedResourcesByKind(rendered, "Service")
			Expect(err).NotTo(HaveOccurred())
			Expect(metricsServices).To(HaveLen(1))

			By("rendering the ServiceMonitor")
			serviceMonitors, err := renderedResourcesByKind(rendered, "ServiceMonitor")
			Expect(err).NotTo(HaveOccurred())
			Expect(serviceMonitors).To(HaveLen(1))

			By("rendering the metrics NetworkPolicy")
			metricsPolicies, err := renderedResourcesByKind(rendered, "NetworkPolicy")
			Expect(err).NotTo(HaveOccurred())
			Expect(metricsPolicies).To(HaveLen(1))
		})

		It("should disable the manager and metrics resources together", func() {
			rendered := renderChart(
				"--set", "metrics.enabled=false",
				"--set", "prometheus.enabled=true",
				"--set", "networkPolicy.enabled=true",
			)

			By("disabling the manager metrics server")
			managerArgs, err := renderedContainerArgs(rendered, "manager")
			Expect(err).NotTo(HaveOccurred())
			Expect(managerArgs).To(Equal([]string{
				"--metrics-bind-address=0",
				"--health-probe-bind-address=:8081",
				"--leader-elect",
			}))

			By("removing the metrics Service")
			metricsServices, err := renderedResourcesByKind(rendered, "Service")
			Expect(err).NotTo(HaveOccurred())
			Expect(metricsServices).To(BeEmpty())

			By("removing the ServiceMonitor even though prometheus stays enabled")
			serviceMonitors, err := renderedResourcesByKind(rendered, "ServiceMonitor")
			Expect(err).NotTo(HaveOccurred())
			Expect(serviceMonitors).To(BeEmpty())

			By("removing the metrics NetworkPolicy")
			metricsPolicies, err := renderedResourcesByKind(rendered, "NetworkPolicy")
			Expect(err).NotTo(HaveOccurred())
			Expect(metricsPolicies).To(BeEmpty())
		})

		It("should use insecure metrics settings consistently in the manager and ServiceMonitor", func() {
			rendered := renderChart(
				"--set", "metrics.enabled=true",
				"--set", "metrics.secure=false",
				"--set", "prometheus.enabled=true",
			)

			By("passing the insecure metrics flag to the manager")
			managerArgs, err := renderedContainerArgs(rendered, "manager")
			Expect(err).NotTo(HaveOccurred())
			Expect(managerArgs).To(ContainElements(
				"--metrics-bind-address=:7443",
				"--metrics-secure=false",
			))

			By("configuring the ServiceMonitor for HTTP")
			serviceMonitors, err := renderedResourcesByKind(rendered, "ServiceMonitor")
			Expect(err).NotTo(HaveOccurred())
			Expect(serviceMonitors).To(HaveLen(1))
			endpoints, found, err := unstructured.NestedSlice(serviceMonitors[0].Object, "spec", "endpoints")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(endpoints).To(HaveLen(1))
			endpoint, ok := endpoints[0].(map[string]interface{})
			Expect(ok).To(BeTrue())
			endpointPort, _, err := unstructured.NestedString(endpoint, "port")
			Expect(err).NotTo(HaveOccurred())
			endpointScheme, _, err := unstructured.NestedString(endpoint, "scheme")
			Expect(err).NotTo(HaveOccurred())
			Expect(endpointPort).To(Equal("http"))
			Expect(endpointScheme).To(Equal("http"))
			_, hasBearerToken, err := unstructured.NestedString(endpoint, "bearerTokenFile")
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBearerToken).To(BeFalse())
			_, hasTLSConfig, err := unstructured.NestedMap(endpoint, "tlsConfig")
			Expect(err).NotTo(HaveOccurred())
			Expect(hasTLSConfig).To(BeFalse())
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

func createKustomizeConversionOnlyWithNetworkPolicies(projectName string) string {
	return createBasicKustomizeOutput(projectName) + `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.io
spec:
  conversion:
    strategy: Webhook
    webhook:
      clientConfig:
        service:
          name: ` + projectName + `-webhook-service
          namespace: ` + projectName + `-system
          path: /convert
      conversionReviewVersions:
        - v1
  group: example.io
  names:
    kind: Widget
    listKind: WidgetList
    plural: widgets
    singular: widget
  scope: Namespaced
  versions:
    - name: v1
      schema:
        openAPIV3Schema:
          type: object
      served: true
      storage: true
    - name: v2
      schema:
        openAPIV3Schema:
          type: object
      served: true
      storage: false
---
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
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ` + projectName + `-allow-webhook-traffic
  namespace: ` + projectName + `-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
  policyTypes:
    - Ingress
  ingress:
    - ports:
        - port: 9443
          protocol: TCP
`
}

func createKustomizeWithWebhooksAndNetworkPolicies(projectName string) string {
	return createKustomizeWithWebhooks(projectName) + `---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ` + projectName + `-allow-metrics-traffic
  namespace: ` + projectName + `-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
  policyTypes:
    - Ingress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            metrics: enabled
      ports:
        - port: 8443
          protocol: TCP
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ` + projectName + `-allow-webhook-traffic
  namespace: ` + projectName + `-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
  policyTypes:
    - Ingress
  ingress:
    - ports:
        - port: 9443
          protocol: TCP
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ` + projectName + `-allow-dns-traffic
  namespace: ` + projectName + `-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
  policyTypes:
    - Ingress
  ingress:
    - ports:
        - port: 5353
          protocol: UDP
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ` + projectName + `-disallow-metrics-traffic
  namespace: ` + projectName + `-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
  policyTypes:
    - Ingress
  ingress:
    - ports:
        - port: 7777
          protocol: TCP
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

// createKustomizeWithMetricsServiceWithoutManagerArg represents a project whose metrics
// Service is present but whose manager args do not include --metrics-bind-address.
func createKustomizeWithMetricsServiceWithoutManagerArg(projectName string) string {
	kustomizeYAML := strings.Replace(
		createKustomizeWithFullDeploymentConfig(projectName),
		"        - --metrics-bind-address=:8443\n",
		"",
		1,
	)

	return kustomizeYAML + `---
apiVersion: v1
kind: Service
metadata:
  name: ` + projectName + `-controller-manager-metrics-service
  namespace: ` + projectName + `-system
spec:
  ports:
  - name: https
    port: 7443
    protocol: TCP
    targetPort: 7443
  selector:
    control-plane: controller-manager
`
}

// createKustomizeWithMetricsResources represents a project with the metrics Service and the
// scaffolded metrics NetworkPolicy. The chart adds the fallback ServiceMonitor during scaffolding.
func createKustomizeWithMetricsResources(projectName string) string {
	return createKustomizeWithMetricsServiceWithoutManagerArg(projectName) + `---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ` + projectName + `-allow-metrics-traffic
  namespace: ` + projectName + `-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          metrics: enabled
    ports:
    - port: 7443
      protocol: TCP
`
}

// createKustomizeWithMetricsServiceAndNoManagerArgs represents a project whose manager has no
// args field at all, while its metrics Service still exposes the configured endpoint.
func createKustomizeWithMetricsServiceAndNoManagerArgs(projectName string) string {
	kustomizeYAML := strings.Replace(
		createKustomizeWithMetricsServiceWithoutManagerArg(projectName),
		`        args:
        - --leader-elect
        - --health-probe-bind-address=:8081
`,
		"",
		1,
	)
	return kustomizeYAML
}

// createKustomizeWithMetricsServiceAndEmptyManagerArgs represents a project whose manager
// explicitly declares an empty args list.
func createKustomizeWithMetricsServiceAndEmptyManagerArgs(projectName string) string {
	return strings.Replace(
		createKustomizeWithMetricsServiceWithoutManagerArg(projectName),
		`        args:
        - --leader-elect
        - --health-probe-bind-address=:8081
`,
		`        args: []
`,
		1,
	)
}

// createKustomizeForServiceAccountRender extends createBasicKustomizeOutput (Namespace +
// ServiceAccount + Deployment) for the serviceAccountName render tests. It adds:
//   - a pod-template annotations block, otherwise the chart nil-pointers on
//     .Values.manager.pod.annotations under `helm template`;
//   - an explicit serviceAccountName on the pod spec, which the helper rewrites;
//   - the three bindings that carry the manager ServiceAccount as a subject, so tests can check every
//     subject stays consistent: manager (switches with rbac.namespaced), leader-election (always a
//     RoleBinding), and metrics-auth (always a ClusterRoleBinding, only when metrics is secure).
func createKustomizeForServiceAccountRender(projectName string) string {
	withPod := strings.Replace(
		createBasicKustomizeOutput(projectName),
		`    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:`,
		`    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager
    spec:
      serviceAccountName: `+projectName+`-controller-manager
      containers:`,
		1,
	)

	return withPod + `---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-manager-role
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ` + projectName + `-manager-role
subjects:
- kind: ServiceAccount
  name: ` + projectName + `-controller-manager
  namespace: ` + projectName + `-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-leader-election-role
  namespace: ` + projectName + `-system
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-leader-election-rolebinding
  namespace: ` + projectName + `-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: ` + projectName + `-leader-election-role
subjects:
- kind: ServiceAccount
  name: ` + projectName + `-controller-manager
  namespace: ` + projectName + `-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-metrics-auth-role
rules:
- apiGroups: ["authentication.k8s.io"]
  resources: ["tokenreviews"]
  verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-metrics-auth-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ` + projectName + `-metrics-auth-role
subjects:
- kind: ServiceAccount
  name: ` + projectName + `-controller-manager
  namespace: ` + projectName + `-system
`
}

// createKustomizeForServiceAccountWithAnnotationsRender reuses the serviceAccountName render
// fixture but gives the ServiceAccount pre-existing annotations. Kustomize sorts metadata keys
// alphabetically, so annotations lands before labels: the ordering that must merge into the
// existing annotations block rather than emit a duplicate key.
func createKustomizeForServiceAccountWithAnnotationsRender(projectName string) string {
	return strings.Replace(
		createKustomizeForServiceAccountRender(projectName),
		`kind: ServiceAccount
metadata:
  labels:`,
		`kind: ServiceAccount
metadata:
  annotations:
    example.com/existing-annotation: preserved-value
  labels:`,
		1,
	)
}

func setupKustomizeFile(filePath, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(content), 0o644)
}

// decodeRenderedResources returns the non-empty Kubernetes objects from a Helm render.
// Keeping render assertions at the Kubernetes object level makes tests independent of Helm
// template layout and YAML indentation.
func decodeRenderedResources(renderedManifest string) ([]*unstructured.Unstructured, error) {
	decoder := k8syaml.NewYAMLOrJSONDecoder(strings.NewReader(renderedManifest), 4096)
	var renderedResources []*unstructured.Unstructured

	for {
		resource := &unstructured.Unstructured{}
		err := decoder.Decode(resource)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode rendered resource: %w", err)
		}
		if len(resource.Object) == 0 {
			continue
		}
		renderedResources = append(renderedResources, resource)
	}

	return renderedResources, nil
}

func renderedResourcesByKind(
	renderedManifest, resourceKind string,
) ([]*unstructured.Unstructured, error) {
	allResources, err := decodeRenderedResources(renderedManifest)
	if err != nil {
		return nil, err
	}

	var matchingResources []*unstructured.Unstructured
	for _, resource := range allResources {
		if resource.GetKind() == resourceKind {
			matchingResources = append(matchingResources, resource)
		}
	}
	return matchingResources, nil
}

// renderedContainerArgs returns the effective args from a named container in a rendered
// Deployment.
func renderedContainerArgs(renderedManifest, containerName string) ([]string, error) {
	deployments, err := renderedResourcesByKind(renderedManifest, "Deployment")
	if err != nil {
		return nil, err
	}

	for _, deployment := range deployments {
		containers, found, err := unstructured.NestedSlice(
			deployment.Object, "spec", "template", "spec", "containers")
		if err != nil {
			return nil, fmt.Errorf("read deployment containers: %w", err)
		}
		if !found {
			continue
		}

		for _, rawContainer := range containers {
			container, ok := rawContainer.(map[string]interface{})
			if !ok {
				continue
			}

			containerNameFromManifest, _, err := unstructured.NestedString(container, "name")
			if err != nil {
				return nil, fmt.Errorf("read container name: %w", err)
			}
			if containerNameFromManifest != containerName {
				continue
			}

			containerArgs, found, err := unstructured.NestedStringSlice(container, "args")
			if err != nil {
				return nil, fmt.Errorf("read %s container args: %w", containerName, err)
			}
			if !found {
				return nil, fmt.Errorf("%s container has no args", containerName)
			}
			return containerArgs, nil
		}
	}

	return nil, fmt.Errorf("rendered Deployment container %q not found", containerName)
}

// createKustomizeWithTolerationsAndSchedulingFields simulates a manager.yaml that already
// contains custom tolerations, nodeSelector, and affinity. Chart generation must not emit a
// duplicate tolerations block, which would make Helm render fail.
func createKustomizeWithTolerationsAndSchedulingFields(projectName string) string {
	return `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-system
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
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
        operator: Exists
      - effect: NoExecute
        key: dedicated
        operator: Equal
        value: controller
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/arch
                operator: In
                values:
                - amd64
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      serviceAccountName: ` + projectName + `-controller-manager
`
}

func createKustomizeWithCustomVolumes(projectName string) string {
	return `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-system
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
        volumeMounts:
        - name: webhook-certs
          mountPath: /tmp/k8s-webhook-server/serving-certs
          readOnly: true
        - name: metrics-certs
          mountPath: /tmp/k8s-metrics-server/metrics-certs
          readOnly: true
        - name: app-config
          mountPath: /etc/config
        - name: app-secret
          mountPath: /etc/secret
          readOnly: true
      volumes:
      - name: webhook-certs
        secret:
          secretName: webhook-server-cert
      - name: metrics-certs
        secret:
          secretName: metrics-server-cert
      - name: app-config
        configMap:
          name: my-config
      - name: app-secret
        secret:
          secretName: my-secret
      serviceAccountName: ` + projectName + `-controller-manager
`
}

func createKustomizeWithSidecarBeforeManager(projectName string) string {
	return `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-system
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
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: sidecar
        image: sidecar:v1
        env:
        - name: SIDECAR_MODE
          value: "active"
        resources:
          limits:
            cpu: 100m
            memory: 32Mi
        securityContext:
          runAsNonRoot: true
      - name: manager
        image: controller:latest
        args:
        - --leader-elect
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
        env:
        - name: MANAGER_ENV
          value: "production"
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
        volumeMounts:
        - name: webhook-certs
          mountPath: /tmp/k8s-webhook-server/serving-certs
          readOnly: true
        - name: app-config
          mountPath: /etc/config
      volumes:
      - name: webhook-certs
        secret:
          secretName: webhook-server-cert
      - name: app-config
        configMap:
          name: my-config
      serviceAccountName: ` + projectName + `-controller-manager
`
}

// createKustomizeWithSidecarBeforeManagerAndMetricsService represents a sidecar-first Pod whose
// manager has no args field. The fixture verifies that manager-specific defaults do not leak into
// the sidecar when the templater has to add the manager args block.
func createKustomizeWithSidecarBeforeManagerAndMetricsService(projectName string) string {
	kustomizeYAML := strings.Replace(
		createKustomizeWithSidecarBeforeManager(projectName),
		`        args:
        - --leader-elect
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
`,
		"",
		1,
	)
	kustomizeYAML = strings.Replace(
		kustomizeYAML,
		`        image: sidecar:v1
        env:`,
		`        image: sidecar:v1
        args:
        - --sidecar-flag
        env:`,
		1,
	)

	return kustomizeYAML + `---
apiVersion: v1
kind: Service
metadata:
  name: ` + projectName + `-controller-manager-metrics-service
  namespace: ` + projectName + `-system
spec:
  ports:
  - name: https
    port: 7443
    protocol: TCP
    targetPort: 7443
  selector:
    control-plane: controller-manager
`
}

func createKustomizeWithCustomVolumesOnly(projectName string) string {
	return `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
  name: ` + projectName + `-system
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
        volumeMounts:
        - name: app-config
          mountPath: /etc/config
        - name: app-secret
          mountPath: /etc/secret
          readOnly: true
      volumes:
      - name: app-config
        configMap:
          name: my-config
      - name: app-secret
        secret:
          secretName: my-secret
      serviceAccountName: ` + projectName + `-controller-manager
`
}

// createKustomizeWithServiceMonitor extends createBasicKustomizeOutput with a ServiceMonitor
// resource that carries the labels kustomize typically emits, so the kustomize-derived
// (rather than the static fallback) ServiceMonitor template is scaffolded.
func createKustomizeWithServiceMonitor(projectName string) string {
	return createBasicKustomizeOutput(projectName) + `---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: ` + projectName + `
    control-plane: controller-manager
  name: ` + projectName + `-controller-manager-metrics-monitor
spec:
  endpoints:
  - path: /metrics
    port: http
    scheme: http
  selector:
    matchLabels:
      app.kubernetes.io/name: ` + projectName + `
      control-plane: controller-manager
`
}
