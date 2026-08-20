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
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"

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

	// writeValuesFile writes a values override into the test's temp dir and returns its path.
	writeValuesFile := func(name, content string) string {
		valuesFile := filepath.Join(tmpDir, name)
		Expect(os.WriteFile(valuesFile, []byte(content), 0o600)).To(Succeed())
		return valuesFile
	}

	// renderChartAt renders an already-scaffolded chart. It owns the release name and namespace
	// so specs that render without re-scaffolding do not restate them.
	renderChartAt := func(chartPath string, setArgs ...string) (string, error) {
		args := append([]string{"template", "my-release", chartPath, "--namespace", "my-namespace"}, setArgs...)
		out, err := exec.Command("helm", args...).CombinedOutput()
		return string(out), err
	}

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

		return renderChartAt(filepath.Join(tmpDir, outputDir, "chart"), setArgs...)
	}

	// containerNamed decodes the rendered manager Deployment with the Kubernetes API types and
	// returns the named container. UnmarshalStrict rejects fields the API does not know, so an
	// invalid valueFrom source fails here rather than at apply time.
	containerNamed := func(rendered, name string) *corev1.Container {
		var deployment appsv1.Deployment
		found := false
		for doc := range strings.SplitSeq(rendered, "\n---") {
			if !strings.Contains(doc, "\nkind: Deployment\n") {
				continue
			}
			Expect(yaml.UnmarshalStrict([]byte(doc), &deployment)).To(Succeed(), "rendered Deployment: %s", doc)
			found = true
			break
		}
		Expect(found).To(BeTrue(), "no Deployment in rendered output: %s", rendered)

		var container *corev1.Container
		for i := range deployment.Spec.Template.Spec.Containers {
			if deployment.Spec.Template.Spec.Containers[i].Name == name {
				container = &deployment.Spec.Template.Spec.Containers[i]
			}
		}
		Expect(container).NotTo(BeNil(), "no %s container in rendered Deployment", name)
		return container
	}

	managerContainer := func(rendered string) *corev1.Container {
		return containerNamed(rendered, "manager")
	}

	managerEnv := func(rendered string) []corev1.EnvVar {
		container := managerContainer(rendered)

		// Every render must satisfy the two API rules the map shape exists to guarantee:
		// unique names, and names Kubernetes accepts.
		seen := map[string]struct{}{}
		for _, envVar := range container.Env {
			_, duplicate := seen[envVar.Name]
			Expect(duplicate).To(BeFalse(), "duplicate environment variable %q in %v", envVar.Name, container.Env)
			seen[envVar.Name] = struct{}{}
			// Relaxed, not strict: Kubernetes accepts any printable ASCII except "=", and the
			// chart deliberately does not re-implement the name rule. The strict form would
			// fail renders the API accepts.
			Expect(validation.IsRelaxedEnvVarName(envVar.Name)).To(BeEmpty(),
				"invalid env var name %q", envVar.Name)
		}
		return container.Env
	}

	envNames := func(env []corev1.EnvVar) []string {
		names := make([]string, 0, len(env))
		for _, envVar := range env {
			names = append(names, envVar.Name)
		}
		return names
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
		// renderWithArgs scaffolds the chart and renders it with `helm template`. When
		// valuesContent is empty, no `-f` override is passed, so the chart renders with its own
		// generated values.yaml (i.e. the default manager.args extracted from the kustomize
		// output), exercising the same code path production users hit before ever touching
		// manager.args themselves.
		renderWithArgs := func(valuesContent string) string {
			var setArgs []string
			if valuesContent != "" {
				setArgs = []string{"-f", writeValuesFile("manager-args-values.yaml", valuesContent)}
			}
			out, err := helmTemplate(createKustomizeWithFullDeploymentConfig("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		// managerArgsLines extracts every rendered "- --flag..." list item so assertions do not
		// depend on indentation and are not confused by other list items in the manifest.
		managerArgsLines := func(rendered string) []string {
			var args []string
			for _, line := range strings.Split(rendered, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "- --") {
					args = append(args, strings.TrimPrefix(trimmed, "- "))
				}
			}
			return args
		}

		It("should render the manager Deployment successfully when manager.args uses the chart's own "+
			"default values (no values override)", func() {
			rendered := renderWithArgs("")

			By("the default extracted arg renders as a literal, unaffected by tpl")
			Expect(managerArgsLines(rendered)).To(ContainElement("--leader-elect"))
		})

		DescribeTable("should resolve manager.args entries through tpl when values are provided",
			func(valuesContent string, wantArgs []string, unwantedSubstrings []string) {
				rendered := renderWithArgs(valuesContent)

				args := managerArgsLines(rendered)
				for _, want := range wantArgs {
					Expect(args).To(ContainElement(want), "rendered manager args: %v", args)
				}
				for _, unwanted := range unwantedSubstrings {
					Expect(rendered).NotTo(ContainSubstring(unwanted))
				}
			},
			Entry("should keep plain literal args unchanged when no template syntax is used (backwards compatible)",
				"manager:\n  args:\n  - --leader-elect\n  - --zap-log-level=info\n",
				[]string{"--leader-elect", "--zap-log-level=info"},
				[]string(nil),
			),
			Entry("should resolve to the release namespace when an arg references .Release.Namespace",
				"manager:\n  args:\n  - --leader-election-namespace={{ .Release.Namespace }}\n",
				[]string{"--leader-election-namespace=my-namespace"},
				[]string{"{{ .Release.Namespace }}"},
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
				[]string{"{{ .Release", "{{ .Chart"},
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
				[]string(nil),
			),
			Entry("should resolve an arg when it calls a Helm template function",
				`manager:
  args:
  - --extra-flag={{ printf "%s-%s" .Release.Name .Release.Namespace }}
`,
				[]string{"--extra-flag=my-release-my-namespace"},
				[]string{"{{ printf"},
			),
		)
	})
	// manager.env is the Kubernetes list, kept in the order the project authored, and
	// manager.envOverrides addresses it by name so --set can reach one variable. The two are merged
	// by name before rendering, so a name reaches the Deployment exactly once.
	Context("Manager env source shapes (rendered)", func() {
		renderShape := func(containerFields string, setArgs ...string) []corev1.EnvVar {
			out, err := helmTemplate(createKustomizeWithEnvShape("test-project", containerFields), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return managerEnv(out)
		}

		DescribeTable("should honour an override whatever shape the source declared",
			func(containerFields string) {
				env := renderShape(containerFields, "--set", "manager.envOverrides.LOG_LEVEL=debug")

				Expect(env).To(ContainElement(corev1.EnvVar{Name: "LOG_LEVEL", Value: "debug"}))
			},
			Entry("env absent", `      - args:
        - --leader-elect
        image: controller:latest
        name: manager`),
			Entry("env inline empty list", `      - args:
        - --leader-elect
        env: []
        image: controller:latest
        name: manager`),
			Entry("env inline null", `      - args:
        - --leader-elect
        env: null
        image: controller:latest
        name: manager`),
			Entry("env block list", `      - args:
        - --leader-elect
        env:
        - name: FOO
          value: bar
        image: controller:latest
        name: manager`),
			Entry("env folded onto the dash, inline empty", `      - env: []
        image: controller:latest
        name: manager`),
			Entry("env folded onto the dash, block list", `      - env:
        - name: FOO
          value: bar
        image: controller:latest
        name: manager`),
			// A block scalar's blank line is content, so the shape table has to carry it too: the
			// line-based replacement must find the end of a value that contains one.
			Entry("env with a multiline value", `      - args:
        - --leader-elect
        env:
        - name: MESSAGE
          value: |-
            first line

            third line
        image: controller:latest
        name: manager`),
			Entry("env folded onto the dash, multiline value", `      - env:
        - name: MESSAGE
          value: |-
            first line

            third line
        image: controller:latest
        name: manager`),
		)

		It("should render no env at all when the source had none and nothing is set", func() {
			Expect(renderShape(`      - args:
        - --leader-elect
        image: controller:latest
        name: manager`)).To(BeEmpty())
		})

		It("should keep a scaffolded variable alongside one added at install time", func() {
			env := renderShape(`      - env:
        - name: FOO
          value: bar
        image: controller:latest
        name: manager`, "--set", "manager.envOverrides.LOG_LEVEL=debug")

			Expect(env).To(ContainElement(corev1.EnvVar{Name: "FOO", Value: "bar"}))
			Expect(env).To(ContainElement(corev1.EnvVar{Name: "LOG_LEVEL", Value: "debug"}))
		})
	})

	// The default Go scaffold declares no env on the manager container, so the chart used to omit
	// the keys entirely and silently ignore install-time additions (#5489).
	Context("Manager env for a project with no env in its Deployment (rendered)", func() {
		renderNoEnvSource := func(setArgs ...string) string {
			out, err := helmTemplate(createBasicKustomizeOutput("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		It("should accept an override even though the Deployment declares none", func() {
			Expect(managerEnv(renderNoEnvSource("--set", "manager.envOverrides.LOG_LEVEL=debug"))).
				To(ContainElement(corev1.EnvVar{Name: "LOG_LEVEL", Value: "debug"}))
		})

		It("should render no env block at all when no variable is set", func() {
			Expect(renderNoEnvSource()).NotTo(MatchRegexp(`(?m)^\s+env:`))
		})

		It("should scaffold both keys so they are discoverable", func() {
			renderNoEnvSource()

			values, err := os.ReadFile(filepath.Join(tmpDir, outputDir, "chart", "values.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(values)).To(ContainSubstring("  env: []"))
			Expect(string(values)).To(ContainSubstring("  envOverrides: {}"))
		})
	})

	// Overrides add, replace and remove by name. Every assertion decodes the rendered Deployment
	// with the Kubernetes types, so a duplicate name or a malformed source fails here.
	Context("Manager env overrides (rendered)", func() {
		render := func(setArgs ...string) []corev1.EnvVar {
			out, err := helmTemplate(createKustomizeWithManagerEnv("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return managerEnv(out)
		}

		watchNamespace := corev1.EnvVar{
			Name: "WATCH_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		}

		It("should render the scaffolded list untouched when nothing is overridden", func() {
			env := render()

			Expect(envNames(env)).To(Equal([]string{"BUSYBOX_IMAGE", "MEMCACHED_IMAGE", "WATCH_NAMESPACE"}))
			Expect(env).To(ContainElement(corev1.EnvVar{Name: "BUSYBOX_IMAGE", Value: "busybox:1.36.1"}))
			Expect(env).To(ContainElement(watchNamespace))
		})

		It("should add a variable the chart never scaffolded", func() {
			env := render("--set", "manager.envOverrides.LOG_LEVEL=debug")

			Expect(env).To(ContainElement(corev1.EnvVar{Name: "LOG_LEVEL", Value: "debug"}))
			Expect(env).To(ContainElement(watchNamespace), "the scaffolded entries survive")
		})

		It("should replace a scaffolded literal in place rather than append a second entry", func() {
			env := render("--set", "manager.envOverrides.BUSYBOX_IMAGE=busybox:latest")

			Expect(env).To(ContainElement(corev1.EnvVar{Name: "BUSYBOX_IMAGE", Value: "busybox:latest"}))
			Expect(envNames(env)).To(Equal([]string{"BUSYBOX_IMAGE", "MEMCACHED_IMAGE", "WATCH_NAMESPACE"}))
		})

		// #5948: the old shape appended a second entry with the same name, which the API server
		// rejects under Server-Side Apply.
		It("should replace a scaffolded valueFrom entry with a literal value", func() {
			env := render("--set", "manager.envOverrides.WATCH_NAMESPACE=pinned-ns")

			Expect(env).To(ContainElement(corev1.EnvVar{Name: "WATCH_NAMESPACE", Value: "pinned-ns"}))
			Expect(env).NotTo(ContainElement(watchNamespace), "the source was left behind")
		})

		It("should remove a scaffolded variable when the override is null", func() {
			Expect(envNames(render("--set", "manager.envOverrides.BUSYBOX_IMAGE=null"))).
				To(Equal([]string{"MEMCACHED_IMAGE", "WATCH_NAMESPACE"}))
		})

		It("should ignore a null override for a name the list never declared", func() {
			Expect(envNames(render("--set", "manager.envOverrides.NEVER_SET=null"))).
				To(Equal([]string{"BUSYBOX_IMAGE", "MEMCACHED_IMAGE", "WATCH_NAMESPACE"}))
		})

		DescribeTable("should coerce a non-string value to the string the API requires",
			func(setArg, want string) {
				Expect(render("--set", setArg)).To(ContainElement(corev1.EnvVar{Name: "PORT", Value: want}))
			},
			Entry("numeric", "manager.envOverrides.PORT=8080", "8080"),
			Entry("boolean", "manager.envOverrides.PORT=true", "true"),
			Entry("empty string", "manager.envOverrides.PORT=", ""),
		)

		It("should accept overrides from a values file as well as --set", func() {
			valuesFile := writeValuesFile("env-overrides.yaml",
				"manager:\n  envOverrides:\n    BUSYBOX_IMAGE: from-file\n    WATCH_NAMESPACE: null\n")

			env := render("-f", valuesFile)

			Expect(env).To(ContainElement(corev1.EnvVar{Name: "BUSYBOX_IMAGE", Value: "from-file"}))
			Expect(envNames(env)).NotTo(ContainElement("WATCH_NAMESPACE"))
		})

		DescribeTable("should name what is wrong rather than leak a template error",
			func(setArgs []string, wantMessage string) {
				out, err := helmTemplate(createKustomizeWithManagerEnv("test-project"), setArgs...)

				Expect(err).To(HaveOccurred(), "helm template should have failed, got: %s", out)
				Expect(out).To(ContainSubstring(wantMessage))
			},
			Entry("env as a map", []string{"--set", "manager.env.FOO=bar"},
				"manager.env must be a list of environment variables, got a map"),
			Entry("env as a scalar", []string{"--set", "manager.env=nope"},
				"manager.env must be a list of environment variables, got a string"),
			Entry("envOverrides as a list", []string{"--set", "manager.envOverrides={a,b}"},
				"manager.envOverrides must be a map keyed by variable name, got a slice"),
			Entry("an override that is a map", []string{"--set", "manager.envOverrides.FOO.bar=baz"},
				"manager.envOverrides.FOO must be a scalar or null"),
		)
	})

	// Ordering is the list's own. Kubernetes expands $(VAR) against variables defined earlier, so
	// the order the project authored has to survive - and the chart never inspects a value to
	// decide it, because references can resolve to names only the kubelet sees.
	Context("Manager env ordering (rendered)", func() {
		render := func(setArgs ...string) []corev1.EnvVar {
			out, err := helmTemplate(createKustomizeWithEnvOrder("test-project"), setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return managerEnv(out)
		}

		// Not the alphabetical order: sorting would move APP_URL ahead of the ZBASE it references.
		sourceOrder := []string{"ZBASE", "APP_URL", "APP_NAME"}

		It("should preserve the source list's order", func() {
			Expect(envNames(render())).To(Equal(sourceOrder))
		})

		It("should render a forward $(VAR) alongside envFrom untouched", func() {
			Expect(render()).To(ContainElement(corev1.EnvVar{Name: "APP_URL", Value: "https://$(ZBASE)"}))
		})

		It("should append a name only an override introduced, alphabetically after the list", func() {
			Expect(envNames(render(
				"--set", "manager.envOverrides.ZZ_LAST=z",
				"--set", "manager.envOverrides.AA_FIRST=a",
			))).To(Equal(append(append([]string{}, sourceOrder...), "AA_FIRST", "ZZ_LAST")))
		})

		It("should keep a replaced variable in its original position", func() {
			Expect(envNames(render("--set", "manager.envOverrides.ZBASE=elsewhere.example"))).
				To(Equal(sourceOrder))
		})

		It("should close the gap when a variable is removed", func() {
			Expect(envNames(render("--set", "manager.envOverrides.APP_URL=null"))).
				To(Equal([]string{"ZBASE", "APP_NAME"}))
		})
	})

	// The list may come from kustomize output verbatim, so it can carry a name twice. Merging by
	// name is what keeps that out of the rendered Deployment.
	Context("Manager env duplicates (rendered)", func() {
		It("should render a repeated name once, at its first position with its last value", func() {
			out, err := helmTemplate(createKustomizeWithEnvShape("test-project", `      - env:
        - name: DUPE
          value: first
        - name: MIDDLE
          value: m
        - name: DUPE
          value: last
        image: controller:latest
        name: manager`))
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)

			env := managerEnv(out)

			Expect(envNames(env)).To(Equal([]string{"DUPE", "MIDDLE"}))
			Expect(env).To(ContainElement(corev1.EnvVar{Name: "DUPE", Value: "last"}))
		})
	})

	// A blank line inside a block scalar is content, not structure. The env replacement is
	// line-based, so treating that line as the end of the block strands the rest of the value after
	// the generated block at an indent nothing owns.
	Context("Manager env with a multiline value (rendered)", func() {
		const manager = `      - env:
        - name: MESSAGE
          value: |-
            first line

            third line
        image: controller:latest
        name: manager
`

		It("should carry the block scalar through the round trip unchanged", func() {
			out, err := helmTemplate(createKustomizeWithEnvShape("test-project", manager))
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)

			// |- chomps the trailing newline, so the interior blank survives and the trailing one
			// does not.
			Expect(managerEnv(out)).To(ContainElement(corev1.EnvVar{
				Name: "MESSAGE", Value: "first line\n\nthird line",
			}))
			Expect(out).To(MatchRegexp(`image: "controller:`), "the manager image was lost")
		})
	})

	// valueFrom lives on the list, in the Kubernetes shape, so every source the API defines works
	// without the chart knowing anything about it. fileKeyRef is the newest and the easiest to drop.
	Context("Manager env valueFrom sources (rendered)", func() {
		It("should carry a fileKeyRef entry through untouched", func() {
			out, err := helmTemplate(createKustomizeWithEnvShape("test-project", `      - env:
        - name: FROM_FILE
          valueFrom:
            fileKeyRef:
              volumeName: config
              path: config.env
              key: MY_KEY
        image: controller:latest
        name: manager`))
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)

			env := managerEnv(out)

			Expect(envNames(env)).To(Equal([]string{"FROM_FILE"}))
			Expect(env[0].ValueFrom).NotTo(BeNil())
			Expect(env[0].ValueFrom.FileKeyRef).NotTo(BeNil(), "the source was dropped or rewritten")
			Expect(env[0].ValueFrom.FileKeyRef.Key).To(Equal("MY_KEY"))
		})

		It("should let an override replace a fileKeyRef with a literal", func() {
			out, err := helmTemplate(createKustomizeWithEnvShape("test-project", `      - env:
        - name: FROM_FILE
          valueFrom:
            fileKeyRef:
              volumeName: config
              path: config.env
              key: MY_KEY
        image: controller:latest
        name: manager`), "--set", "manager.envOverrides.FROM_FILE=literal")
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)

			Expect(managerEnv(out)).To(Equal([]corev1.EnvVar{{Name: "FROM_FILE", Value: "literal"}}))
		})
	})

	// Every fix in this area has its own focused spec, which means each is only ever exercised in
	// isolation. This fixture carries them at once: a whole pod spec written into an annotation
	// ahead of the real one, a sidecar declared before the manager, arguments and a command spelled
	// to look like fields, a nested document inside a block scalar, the generated marker and other
	// Go-template syntax as literal text, a multiline value with an interior blank line, a valueFrom
	// entry, a relaxed variable name, a literal .Values reference, and a duplicate name.
	Context("Manager container with everything at once (rendered)", func() {
		const compound = `---
apiVersion: v1
kind: Namespace
metadata:
  name: test-project-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      spec:
        template:
          spec:
            containers:
            - name: manager
              image: stale:v0
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
      - args:
        - --sidecar-only
        image: sidecar:v1
        name: sidecar
        resources:
          limits:
            cpu: 100m
      - args:
        - --leader-elect
        - env:production
        command:
        - |-
          env:
          - name: NESTED
            value: nested
          resources:
            limits:
              cpu: 9
          {{- $envVars := list }}
          {{ .Spec.replicas }}
        env:
        - name: ZBASE
          value: example.com
        - name: APP_URL
          value: https://$(ZBASE)
        - name: MESSAGE
          value: |-
            first line

            third line
        - name: not-a.valid.name!
          value: relaxed
        - name: DOC
          value: see .Values.manager.env for overrides
        - name: WATCH_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: ZBASE
          value: example.com
        image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m
`

		render := func(setArgs ...string) string {
			out, err := helmTemplate(compound, setArgs...)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return out
		}

		watchNamespaceFromField := corev1.EnvVar{
			Name: "WATCH_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		}

		It("should template the manager and leave every lookalike and the sidecar alone", func() {
			out := render()

			By("rendering the whole list: source order, every value form, the repeat collapsed")
			Expect(managerEnv(out)).To(Equal([]corev1.EnvVar{
				{Name: "ZBASE", Value: "example.com"},
				{Name: "APP_URL", Value: "https://$(ZBASE)"},
				{Name: "MESSAGE", Value: "first line\n\nthird line"},
				{Name: "not-a.valid.name!", Value: "relaxed"},
				{Name: "DOC", Value: "see .Values.manager.env for overrides"},
				watchNamespaceFromField,
			}))

			By("returning the arguments and the command byte for byte")
			manager := managerContainer(out)
			Expect(manager.Args).To(Equal([]string{"--leader-elect", "env:production"}))
			Expect(manager.Command).To(Equal([]string{
				"env:\n- name: NESTED\n  value: nested\nresources:\n  limits:\n    cpu: 9\n" +
					"{{- $envVars := list }}\n{{ .Spec.replicas }}",
			}))

			By("returning the sidecar field for field")
			sidecar := containerNamed(out, "sidecar")
			Expect(sidecar.Image).To(Equal("sidecar:v1"))
			Expect(sidecar.ImagePullPolicy).To(BeEmpty())
			Expect(sidecar.Args).To(Equal([]string{"--sidecar-only"}))
			Expect(sidecar.Command).To(BeEmpty())
			Expect(sidecar.Env).To(BeEmpty())
			Expect(sidecar.SecurityContext).To(BeNil())
			Expect(sidecar.Resources.Limits.Cpu().String()).To(Equal("100m"))

			By("leaving the pod spec written into the annotation as text")
			Expect(out).To(ContainSubstring("image: stale:v0"))

			By("templating the manager's own fields, and only the manager's")
			Expect(manager.Image).To(HavePrefix("controller:"))
			values, err := os.ReadFile(filepath.Join(tmpDir, outputDir, "chart", "values.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(values)).To(ContainSubstring("cpu: 500m"), "the manager's resources were not extracted")
			Expect(string(values)).NotTo(ContainSubstring("cpu: 100m"), "the sidecar's resources were extracted")
			Expect(string(values)).NotTo(ContainSubstring("cpu: 9"), "a nested lookalike was extracted")
			Expect(string(values)).NotTo(ContainSubstring("stale:v0"), "the annotation's pod spec was extracted")
		})

		It("should still accept install-time overrides on that container", func() {
			env := managerEnv(render(
				"--set", "manager.envOverrides.ZBASE=elsewhere.example",
				"--set", "manager.envOverrides.MESSAGE=null",
				"--set", "manager.envOverrides.PORT=8080",
				"--set", "manager.envOverrides.LOG_LEVEL=debug",
			))

			// Replaced in place, removed, and the two additions after the list in alphabetical order.
			Expect(env).To(Equal([]corev1.EnvVar{
				{Name: "ZBASE", Value: "elsewhere.example"},
				{Name: "APP_URL", Value: "https://$(ZBASE)"},
				{Name: "not-a.valid.name!", Value: "relaxed"},
				{Name: "DOC", Value: "see .Values.manager.env for overrides"},
				watchNamespaceFromField,
				{Name: "LOG_LEVEL", Value: "debug"},
				{Name: "PORT", Value: "8080"},
			}))
		})
	})

	// The marker that says "this container's env is already generated" is a Helm action. Matched as
	// a substring of the document, any user data carrying that text claims the block already exists
	// and the container keeps its literal env - so the generated chart ignores every override.
	Context("Manager env when user data contains the generated marker", func() {
		const marker = `{{- $envVars := list }}`
		const withMarkerEverywhere = `---
apiVersion: v1
kind: Namespace
metadata:
  name: test-project-system
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
      annotations:
        example.com/note: '{{- $envVars := list }}'
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - command:
        - |-
          {{- $envVars := list }}
        env:
        - name: FOO
          value: bar
        image: controller:latest
        name: manager
`

		// The marker is deliberately only in the annotation and the command here. It is equally
		// dangerous in an env value or an argument, and the applier specs cover those - but both of
		// those are extracted into values.yaml, which is itself rendered as a Go template, so any
		// "{{" in them fails chart generation outright. That is a separate, pre-existing defect.
		It("should template the env and keep the user's copies of the text", func() {
			out, err := helmTemplate(withMarkerEverywhere, "--set", "manager.envOverrides.FOO=changed")
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)

			By("honouring the override, which only happens if the env was templated at all")
			Expect(managerEnv(out)).To(Equal([]corev1.EnvVar{{Name: "FOO", Value: "changed"}}))

			By("returning the user's copies of the marker text verbatim")
			Expect(managerContainer(out).Command).To(Equal([]string{marker}))
			Expect(out).To(ContainSubstring("example.com/note:"))
		})

		It("should still render the scaffolded value when nothing overrides it", func() {
			out, err := helmTemplate(withMarkerEverywhere)
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)

			Expect(managerEnv(out)).To(Equal([]corev1.EnvVar{{Name: "FOO", Value: "bar"}}))
			Expect(managerContainer(out).Command).To(Equal([]string{marker}))
		})
	})

	// A value carrying Go template syntax is the one thing manager.env does not round-trip. It is
	// extracted into values.yaml, and values.yaml is itself scaffolded from a Go template, so the
	// user's braces are executed against the scaffolder rather than written out. This is a
	// pre-existing defect of every field extracted into values.yaml, not of the env shape - pinned
	// here so the documented limitation stays true and a fix cannot land unnoticed.
	Context("Manager env value containing Go template syntax", func() {
		It("should fail chart generation rather than render something else", func() {
			Expect(setupKustomizeFile(manifestsFile, createKustomizeWithEnvShape("test-project",
				`      - env:
        - name: TPL
          value: "{{ .Spec.replicas }}"
        image: controller:latest
        name: manager`))).To(Succeed())

			scaffolder := scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			scaffolder.InjectFS(fs)

			err := scaffolder.Scaffold()
			Expect(err).To(HaveOccurred(), "chart generation unexpectedly succeeded")
			Expect(err.Error()).To(ContainSubstring("failed to execute Helm chart templates"))
		})
	})

	// Removal is a tombstone: the key stays, with a nil value, and the template reads its presence.
	// helm upgrade --reuse-values coalesces this invocation's values over the previous release's,
	// and coalescing deletes a key whose new value is nil when the old values held a value for it -
	// so on that one path the tombstone never reaches the template. The chart cannot detect this:
	// what arrives is a map with no such key, which is indistinguishable from never asking. These
	// specs pin what actually happens so the documented limitation cannot drift away from it.
	Context("Manager env removal under helm upgrade --reuse-values", func() {
		// reuseValues reproduces action.Upgrade.reuseValues: this invocation's --set values are
		// coalesced over the previous release's stored values, dst authoritative. Parsing the flags
		// with helm's own strvals is what makes "=null" a nil here rather than by assertion.
		reuseValues := func(previous map[string]any, setArgs ...string) string {
			newVals := map[string]any{}
			for _, arg := range setArgs {
				Expect(strvals.ParseInto(arg, newVals)).To(Succeed())
			}

			merged, err := yaml.Marshal(chartutil.CoalesceTables(newVals, previous))
			Expect(err).NotTo(HaveOccurred())
			return writeValuesFile("reused-values.yaml", string(merged))
		}

		renderReusing := func(previous map[string]any, setArgs ...string) []corev1.EnvVar {
			out, err := helmTemplate(createKustomizeWithManagerEnv("test-project"),
				"-f", reuseValues(previous, setArgs...))
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			return managerEnv(out)
		}

		It("should remove the variable when the previous release set no such override", func() {
			env := renderReusing(
				map[string]any{"manager": map[string]any{"envOverrides": map[string]any{}}},
				"manager.envOverrides.BUSYBOX_IMAGE=null",
			)

			Expect(envNames(env)).To(Equal([]string{"MEMCACHED_IMAGE", "WATCH_NAMESPACE"}))
		})

		// The limitation. Not a workaround, and not a bug the template can fix: drop the entry from
		// manager.env, or put BUSYBOX_IMAGE: null in a values file passed with -f.
		It("should cancel the previous override rather than remove the variable", func() {
			env := renderReusing(
				map[string]any{"manager": map[string]any{"envOverrides": map[string]any{
					"BUSYBOX_IMAGE": "pinned:1.0",
				}}},
				"manager.envOverrides.BUSYBOX_IMAGE=null",
			)

			// The whole list, so a change to the fixture's own values cannot quietly turn this into
			// a presence check: what it pins is the fall back to the manager.env value.
			Expect(env).To(Equal([]corev1.EnvVar{
				{Name: "BUSYBOX_IMAGE", Value: "busybox:1.36.1"},
				{Name: "MEMCACHED_IMAGE", Value: "memcached:1.6.26-alpine3.19"},
				{
					Name: "WATCH_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
					},
				},
			}))
		})
	})

	// Regenerating over a chart whose values.yaml already holds a list is an ordinary run now:
	// the shape never changed, so there is nothing to migrate and nothing to reject.
	Context("Regeneration over existing values", func() {
		It("should regenerate without complaint and keep rendering the chart", func() {
			Expect(setupKustomizeFile(manifestsFile, createKustomizeWithManagerEnv("test-project"))).To(Succeed())

			first := scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			first.InjectFS(fs)
			Expect(first.Scaffold()).To(Succeed())

			again := scaffolds.NewChartScaffolder(projectConfig, false, manifestsFile, outputDir)
			again.InjectFS(fs)
			Expect(again.Scaffold()).To(Succeed())

			out, err := renderChartAt(filepath.Join(tmpDir, outputDir, "chart"),
				"--set", "manager.envOverrides.LOG_LEVEL=debug")
			Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
			Expect(managerEnv(out)).To(ContainElement(corev1.EnvVar{Name: "LOG_LEVEL", Value: "debug"}))
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

// createKustomizeWithEnvShape builds a manager Deployment whose container fields are exactly
// containerFields, so a spec can pick how the serializer spelled env: absent, inline empty, inline
// null, a block list, or any of those folded onto the sequence dash when env sorts first.
func createKustomizeWithEnvShape(projectName, containerFields string) string {
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
` + containerFields
}

// createKustomizeWithManagerEnv carries the two env shapes the manager.env map has to round-trip:
// plain literals and a valueFrom entry (WATCH_NAMESPACE, the variable that broke in #5948).
func createKustomizeWithManagerEnv(projectName string) string {
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
        args:
        - --leader-elect
        env:
        - name: BUSYBOX_IMAGE
          value: busybox:1.36.1
        - name: MEMCACHED_IMAGE
          value: memcached:1.6.26-alpine3.19
        - name: WATCH_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
`
}

// createKustomizeWithEnvOrder declares its variables in an order the map cannot reproduce on its
// own: APP_URL references $(ZBASE) and both are declared before the alphabetically earlier
// APP_NAME. envFrom is there too, so a reference can resolve to a name the chart never sees -
// which is why expansion is Kubernetes' business and not something the chart may validate.
func createKustomizeWithEnvOrder(projectName string) string {
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
        envFrom:
        - configMapRef:
            name: manager-config
        env:
        - name: ZBASE
          value: example.com
        - name: APP_URL
          value: https://$(ZBASE)
        - name: APP_NAME
          value: ` + projectName + `
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
