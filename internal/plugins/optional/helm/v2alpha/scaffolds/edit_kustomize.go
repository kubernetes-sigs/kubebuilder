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

	"sigs.k8s.io/kubebuilder/v4/internal/plugins/optional/helm/v2alpha/scaffolds/internal/kustomize"
	"sigs.k8s.io/kubebuilder/v4/internal/plugins/optional/helm/v2alpha/scaffolds/internal/templates"
	charttemplates "sigs.k8s.io/kubebuilder/v4/internal/plugins/optional/helm/v2alpha/scaffolds/internal/templates/chart-templates"
	"sigs.k8s.io/kubebuilder/v4/internal/plugins/optional/helm/v2alpha/scaffolds/internal/templates/github"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
)

const (
	defaultManifestsFile = "dist/install.yaml"
)

var _ plugins.Scaffolder = &editKustomizeScaffolder{}

type editKustomizeScaffolder struct {
	config        config.Config
	fs            machinery.Filesystem
	force         bool
	manifestsFile string
	outputDir     string
}

// NewKustomizeHelmScaffolder returns a new Scaffolder for HelmPlugin using kustomize output
func NewKustomizeHelmScaffolder(cfg config.Config, force bool, manifestsFile, outputDir string) plugins.Scaffolder {
	return &editKustomizeScaffolder{
		config:        cfg,
		force:         force,
		manifestsFile: manifestsFile,
		outputDir:     outputDir,
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

	// Warn if Custom Resource instances were found and will be ignored
	if len(resources.CustomResources) > 0 {
		slog.Warn(
			"Custom Resource instances found. They will be ignored and not included in the Helm chart",
			"count", len(resources.CustomResources),
			"note", "CRs are environment-specific and should be created manually after chart installation",
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

	// Analyze resources to determine chart features
	hasWebhooks := len(resources.WebhookConfigurations) > 0 || len(resources.Certificates) > 0
	// Prometheus is enabled when ServiceMonitor resources exist (../prometheus enabled)
	hasPrometheus := len(resources.ServiceMonitors) > 0
	// Metrics are enabled either when ServiceMonitor exists or when a metrics service is present
	hasMetrics := hasPrometheus
	if !hasMetrics {
		for _, svc := range resources.Services {
			if strings.Contains(svc.GetName(), "metrics") {
				hasMetrics = true
				break
			}
		}
	}

	// When Prometheus is enabled via kustomize, ensure any previously-generated
	// generic ServiceMonitor file is removed to avoid duplicates in the chart.
	if hasPrometheus {
		staleSM := filepath.Join(s.outputDir, "chart", "templates", "monitoring", "servicemonitor.yaml")
		if rmErr := s.fs.FS.Remove(staleSM); rmErr != nil && !os.IsNotExist(rmErr) {
			// Not fatal; log and continue
			slog.Warn("failed to remove stale generic ServiceMonitor", "path", staleSM, "error", rmErr)
		}
	}
	namePrefix := resources.EstimatePrefix(s.config.GetProjectName())
	chartName := s.config.GetProjectName()
	chartConverter := kustomize.NewChartConverter(resources, namePrefix, chartName, s.outputDir)
	deploymentConfig := chartConverter.ExtractDeploymentConfig()

	// Create scaffold for standard Helm chart files (uses machinery defaults 0755/0644).
	scaffold := machinery.NewScaffold(s.fs, machinery.WithConfig(s.config))

	// Define the standard Helm chart files to generate
	chartFiles := []machinery.Builder{
		&github.HelmChartCI{Force: s.force},
		&templates.HelmChart{OutputDir: s.outputDir, Force: s.force},
		&templates.HelmValuesBasic{
			// values.yaml with dynamic config
			HasWebhooks:      hasWebhooks,
			HasMetrics:       hasMetrics,
			DeploymentConfig: deploymentConfig,
			OutputDir:        s.outputDir,
			Force:            s.force,
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

	// Generate template files from kustomize output
	if writeErr := chartConverter.WriteChartFiles(s.fs); writeErr != nil {
		return fmt.Errorf("failed to write chart template files: %w", writeErr)
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

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
