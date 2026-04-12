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

package scaffolds

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/kustomize"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/templates"
	charttemplates "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/templates/chart-templates"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/templates/github"
)

const (
	defaultManifestsFile = "dist/install.yaml"
)

var _ plugins.Scaffolder = &editKustomizeScaffolder{}

type editKustomizeScaffolder struct {
	config          config.Config
	fs              machinery.Filesystem
	force           bool
	manifestsFile   string
	outputDir       string
	crdSubchart     bool
	samplesSubchart bool
}

// NewKustomizeHelmScaffolder returns a new Scaffolder for HelmPlugin using kustomize output
func NewKustomizeHelmScaffolder(
	cfg config.Config, force bool, manifestsFile, outputDir string, crdSubchart, samplesSubchart bool,
) plugins.Scaffolder {
	return &editKustomizeScaffolder{
		config:          cfg,
		force:           force,
		manifestsFile:   manifestsFile,
		outputDir:       outputDir,
		crdSubchart:     crdSubchart,
		samplesSubchart: samplesSubchart,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *editKustomizeScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold generates the complete Helm chart from kustomize output
func (s *editKustomizeScaffolder) Scaffold() error {
	slog.Info("Generating Helm Chart from kustomize output")

	// Ensure chart directory structure exists
	if err := s.ensureChartDirectoryExists(); err != nil {
		return fmt.Errorf("failed to create chart directory: %w", err)
	}

	// Generate fresh kustomize output if using default file
	if s.manifestsFile == defaultManifestsFile {
		if err := s.generateKustomizeOutput(); err != nil {
			return fmt.Errorf("failed to generate kustomize output: %w", err)
		}
	}

	// Parse the kustomize output into organized resource groups
	parser := kustomize.NewParser(s.manifestsFile)
	resources, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse kustomize output from %s: %w", s.manifestsFile, err)
	}

	// Log Custom Resource status based on flags
	s.logCustomResourceStatus(resources)

	// Analyze resources to determine chart features
	hasWebhooks, hasPrometheus, hasMetrics, hasClusterScopedRBAC := s.analyzeChartFeatures(resources)

	// Remove stale ServiceMonitor if Prometheus is enabled
	if hasPrometheus {
		s.removeStaleServiceMonitor()
	}

	namePrefix := resources.EstimatePrefix(s.config.GetProjectName())
	chartName := s.config.GetProjectName()

	// Extract namespace configuration for multi-namespace RBAC
	_, roleNamespaces := s.extractNamespaceConfiguration(resources, namePrefix)

	chartConverter := kustomize.NewChartConverter(resources, namePrefix, chartName, s.outputDir, roleNamespaces)
	deploymentConfig := chartConverter.ExtractDeploymentConfig()

	// Create scaffold for standard Helm chart files (uses machinery defaults 0755/0644).
	scaffold := machinery.NewScaffold(s.fs, machinery.WithConfig(s.config))

	// Handle sub-charts based on flags
	if s.crdSubchart || s.samplesSubchart {
		if subchartErr := s.scaffoldSubcharts(resources, scaffold); subchartErr != nil {
			return fmt.Errorf("failed to scaffold sub-charts: %w", subchartErr)
		}
	}

	// Define the standard Helm chart files to generate
	chartFiles := []machinery.Builder{
		&github.HelmChartCI{Force: s.force},
		&templates.HelmChart{OutputDir: s.outputDir, Force: s.force, CRDSubchart: s.crdSubchart},
		&templates.HelmValuesBasic{
			// values.yaml with dynamic config
			HasWebhooks:          hasWebhooks,
			HasMetrics:           hasMetrics,
			HasClusterScopedRBAC: hasClusterScopedRBAC,
			RoleNamespaces:       roleNamespaces,
			DeploymentConfig:     deploymentConfig,
			OutputDir:            s.outputDir,
			Force:                s.force,
		},
		&templates.HelmIgnore{OutputDir: s.outputDir, Force: s.force},
		&charttemplates.HelmHelpers{OutputDir: s.outputDir, Force: s.force},
		&charttemplates.Notes{
			OutputDir: s.outputDir,
			Force:     s.force,
		},
	}

	// Only scaffold the generic ServiceMonitor when the project does NOT already
	// provide one via kustomize (../prometheus). This avoids duplicate objects
	// with the same name within the Helm chart.
	if !hasPrometheus {
		// Find the metrics service name from parsed resources
		metricsServiceName := namePrefix + "-controller-manager-metrics-service"
		for _, svc := range resources.Services {
			if strings.Contains(svc.GetName(), "metrics-service") {
				metricsServiceName = svc.GetName()
				break
			}
		}

		chartFiles = append(chartFiles, &charttemplates.ServiceMonitor{
			OutputDir:   s.outputDir,
			ServiceName: metricsServiceName,
			Force:       s.force,
		})
	}

	// If CRD sub-chart is enabled, temporarily remove CRDs from resources
	// since they'll be written to the sub-chart instead
	var savedCRDs []*unstructured.Unstructured
	if s.crdSubchart {
		savedCRDs = resources.CustomResourceDefinitions
		resources.CustomResourceDefinitions = nil
	}

	// Generate template files from kustomize output
	if writeErr := chartConverter.WriteChartFiles(s.fs); writeErr != nil {
		return fmt.Errorf("failed to write chart template files: %w", writeErr)
	}

	// Restore CRDs if they were saved
	if s.crdSubchart {
		resources.CustomResourceDefinitions = savedCRDs
	}

	// Generate standard Helm chart files
	if err = scaffold.Execute(chartFiles...); err != nil {
		return fmt.Errorf("failed to generate Helm chart files: %w", err)
	}

	slog.Info("Helm Chart generation completed successfully")
	return nil
}

// generateKustomizeOutput runs make build-installer to generate the manifests file
func (s *editKustomizeScaffolder) generateKustomizeOutput() error {
	slog.Info("Generating kustomize output with make build-installer")

	// Check if Makefile exists
	if _, err := os.Stat("Makefile"); os.IsNotExist(err) {
		return fmt.Errorf("makefile not found in current directory")
	}

	// Run make build-installer
	cmd := exec.Command("make", "build-installer")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run make build-installer: %w", err)
	}

	// Verify that the manifests file was created
	if _, err := os.Stat(defaultManifestsFile); os.IsNotExist(err) {
		return fmt.Errorf("%s was not generated by make build-installer", defaultManifestsFile)
	}

	return nil
}

// ensureChartDirectoryExists creates the chart directory structure if it doesn't exist
func (s *editKustomizeScaffolder) ensureChartDirectoryExists() error {
	dirs := []string{
		filepath.Join(s.outputDir, "chart"),
		filepath.Join(s.outputDir, "chart", "templates"),
	}

	// Create additional directories based on flags
	if s.crdSubchart {
		dirs = append(dirs,
			filepath.Join(s.outputDir, "chart", "crds"),
			filepath.Join(s.outputDir, "chart", "crds", "templates"),
		)
	}

	if s.samplesSubchart {
		dirs = append(dirs,
			filepath.Join(s.outputDir, "chart", "samples"),
			filepath.Join(s.outputDir, "chart", "samples", "templates"),
		)
	} else if s.crdSubchart {
		// If crd-subchart but not samples-subchart, samples go to templates/samples/
		dirs = append(dirs,
			filepath.Join(s.outputDir, "chart", "templates", "samples"),
		)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// scaffoldSubcharts creates sub-charts for CRDs and/or samples based on flags
func (s *editKustomizeScaffolder) scaffoldSubcharts(
	resources *kustomize.ParsedResources, scaffold *machinery.Scaffold,
) error {
	// Handle CRD sub-chart
	if s.crdSubchart {
		// Create the CRD sub-chart's Chart.yaml
		if err := scaffold.Execute(&templates.CRDSubchart{
			OutputDir: s.outputDir,
			Force:     s.force,
		}); err != nil {
			return fmt.Errorf("failed to generate CRD sub-chart Chart.yaml: %w", err)
		}

		// Write CRDs to the CRD sub-chart's templates/ directory
		if len(resources.CustomResourceDefinitions) > 0 {
			crdWriter := kustomize.NewCRDWriter(s.outputDir)
			if err := crdWriter.WriteCRDs(s.fs, resources.CustomResourceDefinitions); err != nil {
				return fmt.Errorf("failed to write CRDs to sub-chart: %w", err)
			}
		}
	}

	// Handle CustomResource samples
	if len(resources.CustomResources) > 0 {
		if s.samplesSubchart {
			// Create samples sub-chart
			if err := scaffold.Execute(
				&templates.SamplesSubchart{
					OutputDir: s.outputDir,
					Force:     s.force,
				},
				&templates.SamplesReadme{
					OutputDir: s.outputDir,
					Force:     s.force,
				},
			); err != nil {
				return fmt.Errorf("failed to generate samples sub-chart files: %w", err)
			}

			slog.Info(
				"Writing Custom Resource samples to samples sub-chart",
				"count", len(resources.CustomResources),
				"note", "Install samples sub-chart AFTER main chart: helm install samples ./chart/samples",
			)
			samplesWriter := kustomize.NewSamplesWriter(s.outputDir, true)
			if err := samplesWriter.WriteSamples(s.fs, resources.CustomResources); err != nil {
				return fmt.Errorf("failed to write CR samples to sub-chart: %w", err)
			}
		} else if s.crdSubchart {
			// Samples go to templates/samples/ (conditional in main chart)
			slog.Info(
				"Writing Custom Resource samples to templates/samples/",
				"count", len(resources.CustomResources),
				"note", "Samples are conditional on .Values.samples.install (defaults to false)",
			)
			samplesWriter := kustomize.NewSamplesWriter(s.outputDir, false)
			if err := samplesWriter.WriteSamples(s.fs, resources.CustomResources); err != nil {
				return fmt.Errorf("failed to write CR samples to templates: %w", err)
			}
		}
	}

	return nil
}

// logCustomResourceStatus logs the status of Custom Resources based on flags
func (s *editKustomizeScaffolder) logCustomResourceStatus(resources *kustomize.ParsedResources) {
	if len(resources.CustomResources) == 0 {
		return
	}

	if s.crdSubchart {
		if s.samplesSubchart {
			slog.Info(
				"Custom Resource instances found. They will be added to a separate samples sub-chart",
				"count", len(resources.CustomResources),
				"note", "Samples sub-chart should be installed AFTER the main chart (controller must be running first)",
			)
		} else {
			slog.Info(
				"Custom Resource instances found. They will be added to templates/samples/",
				"count", len(resources.CustomResources),
				"note", "Samples are conditional on .Values.samples.install (defaults to false)",
			)
		}
	} else {
		slog.Warn(
			"Custom Resource instances found. They will be ignored and not included in the Helm chart",
			"count", len(resources.CustomResources),
			"note",
			"CRs are environment-specific and should be created manually after chart installation, "+
				"or use --crd-subchart flag to include them",
		)
		for _, cr := range resources.CustomResources {
			slog.Warn(
				"Ignoring Custom Resource instance",
				"kind", cr.GetKind(),
				"apiVersion", cr.GetAPIVersion(),
				"name", cr.GetName(),
			)
		}
	}
}

// analyzeChartFeatures analyzes resources to determine chart features
func (s *editKustomizeScaffolder) analyzeChartFeatures(
	resources *kustomize.ParsedResources,
) (hasWebhooks, hasPrometheus, hasMetrics, hasClusterScopedRBAC bool) {
	hasWebhooks = len(resources.WebhookConfigurations) > 0 || len(resources.Certificates) > 0
	hasPrometheus = len(resources.ServiceMonitors) > 0
	hasMetrics = hasPrometheus

	if !hasMetrics {
		for _, svc := range resources.Services {
			if strings.Contains(svc.GetName(), "metrics") {
				hasMetrics = true
				break
			}
		}
	}

	// Check if project has cluster-scoped RBAC (ClusterRole resources for business logic).
	for _, cr := range resources.ClusterRoles {
		name := cr.GetName()
		// Exclude Kubebuilder-scaffolded metrics roles
		if strings.HasSuffix(name, "-metrics-auth-role") || strings.HasSuffix(name, "-metrics-reader") {
			continue
		}
		hasClusterScopedRBAC = true
		break
	}

	return hasWebhooks, hasPrometheus, hasMetrics, hasClusterScopedRBAC
}

// removeStaleServiceMonitor removes stale ServiceMonitor to avoid duplicates
func (s *editKustomizeScaffolder) removeStaleServiceMonitor() {
	staleSM := filepath.Join(s.outputDir, "chart", "templates", "monitoring", "servicemonitor.yaml")
	if rmErr := s.fs.FS.Remove(staleSM); rmErr != nil && !os.IsNotExist(rmErr) {
		slog.Warn("failed to remove stale generic ServiceMonitor", "path", staleSM, "error", rmErr)
	}
}

// extractNamespaceConfiguration extracts namespace configuration for multi-namespace RBAC
func (s *editKustomizeScaffolder) extractNamespaceConfiguration(
	resources *kustomize.ParsedResources, namePrefix string,
) (managerNamespace string, roleNamespaces map[string]string) {
	managerNamespace = namePrefix + "-system"
	if resources.Deployment != nil {
		if ns := resources.Deployment.GetNamespace(); ns != "" {
			managerNamespace = ns
		}
	}

	roleNamespaces = make(map[string]string)
	for _, role := range resources.Roles {
		ns := role.GetNamespace()
		if ns != "" && ns != managerNamespace {
			roleName := role.GetName()
			suffix := strings.TrimPrefix(roleName, namePrefix+"-")
			roleNamespaces[suffix] = ns
		}
	}
	for _, binding := range resources.RoleBindings {
		ns := binding.GetNamespace()
		if ns != "" && ns != managerNamespace {
			bindingName := binding.GetName()
			suffix := strings.TrimPrefix(bindingName, namePrefix+"-")
			roleNamespaces[suffix] = ns
		}
	}

	return managerNamespace, roleNamespaces
}
