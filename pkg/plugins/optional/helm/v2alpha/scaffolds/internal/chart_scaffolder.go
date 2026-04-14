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

package internal

import (
	"fmt"
	"log/slog"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/extractor"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/kustomize"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/templates"
	charttemplates "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/templates/chart-templates"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/templates/github"
)

// ChartScaffolderConfig contains configuration for Helm chart generation.
type ChartScaffolderConfig struct {
	ProjectName   string
	ManifestsFile string
	OutputDir     string
	Force         bool
}

// ChartScaffolder orchestrates the conversion of kustomize output to Helm charts.
// It parses kustomize YAML, analyzes resources, categorizes by function, applies Helm templating,
// and generates machinery.Builders. File writing is handled by machinery.Scaffold.Execute().
type ChartScaffolder struct {
	config ChartScaffolderConfig
}

// NewChartScaffolder creates a new chart scaffolder.
func NewChartScaffolder(config ChartScaffolderConfig) *ChartScaffolder {
	return &ChartScaffolder{config: config}
}

// PrepareTemplates executes the conversion pipeline and returns Machinery builders ready for execution.
// Parses kustomize YAML, analyzes resources to extract metadata and features, converts resources to
// Helm templates, and prepares Machinery builders with the analyzed data.
func (s *ChartScaffolder) PrepareTemplates(_ machinery.Filesystem) ([]machinery.Builder, error) {
	parser := kustomize.NewParser(s.config.ManifestsFile)

	// Note: We always use os.Open() (via parser.Parse()) because the manifests file is on the OS filesystem.
	// The injected filesystem in machinery.Filesystem is used for writing output files, not reading input.
	resources, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse kustomize output from %s: %w", s.config.ManifestsFile, err)
	}

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

	resourceExtractor := extractor.NewExtractor()
	extraction := resourceExtractor.Extract(&extractor.ResourceSet{
		Namespace:                 resources.Namespace,
		Deployment:                resources.Deployment,
		Services:                  resources.Services,
		CustomResourceDefinitions: resources.CustomResourceDefinitions,
		ServiceAccount:            resources.ServiceAccount,
		Roles:                     resources.Roles,
		ClusterRoles:              resources.ClusterRoles,
		RoleBindings:              resources.RoleBindings,
		ClusterRoleBindings:       resources.ClusterRoleBindings,
		WebhookConfigurations:     resources.WebhookConfigurations,
		Certificates:              resources.Certificates,
		Issuer:                    resources.Issuer,
		ServiceMonitors:           resources.ServiceMonitors,
		Other:                     resources.Other,
	}, s.config.ProjectName)

	chartConverter := kustomize.NewChartConverter(
		resources,
		extraction.Metadata.DetectedPrefix,
		extraction.Metadata.ChartName,
		extraction.Metadata.ManagerNamespace,
		s.config.OutputDir,
		extraction.Features.RoleNamespaces,
	)

	// Get builders for kustomize-derived chart templates
	chartBuilders := chartConverter.GetChartBuilders()

	builders := []machinery.Builder{
		&github.HelmChartCI{Force: s.config.Force},
		&templates.HelmChart{
			OutputDir:     s.config.OutputDir,
			ChartMetadata: extraction.Metadata,
		},
		&templates.HelmValues{
			Extraction: extraction,
			OutputDir:  s.config.OutputDir,
			Force:      s.config.Force,
		},
		&templates.HelmIgnore{OutputDir: s.config.OutputDir, Force: s.config.Force},
		&charttemplates.HelmHelpers{OutputDir: s.config.OutputDir, Force: s.config.Force},
		&charttemplates.Notes{
			OutputDir: s.config.OutputDir,
			Force:     s.config.Force,
		},
	}

	// Add generic ServiceMonitor only if kustomize output doesn't provide one
	if !extraction.Features.HasPrometheus {
		metricsServiceName := extraction.Metadata.DetectedPrefix + "-controller-manager-metrics-service"
		for _, svc := range resources.Services {
			svcName := svc.GetName()
			if strings.HasSuffix(svcName, "-metrics-service") ||
				strings.HasSuffix(svcName, "-controller-manager-metrics-service") {
				metricsServiceName = svcName
				break
			}
		}

		builders = append(builders, &charttemplates.ServiceMonitor{
			OutputDir:   s.config.OutputDir,
			ServiceName: metricsServiceName,
			Force:       s.config.Force,
		})
	}

	// Append kustomize-derived chart templates
	builders = append(builders, chartBuilders...)

	return builders, nil
}
