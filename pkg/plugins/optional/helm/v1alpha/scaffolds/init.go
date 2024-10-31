/*
Copyright 2024 The Kubernetes Authors.

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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/prometheus"

	"sigs.k8s.io/yaml"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates"
	chart_templates "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates"
	templatescertmanager "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/cert-manager"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/manager"
	templatesmetrics "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/metrics"
	templateswebhooks "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/webhook"
)

var _ plugins.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config config.Config

	fs machinery.Filesystem

	force bool
}

// NewInitHelmScaffolder returns a new Scaffolder for HelmPlugin
func NewInitHelmScaffolder(config config.Config, force bool) plugins.Scaffolder {
	return &initScaffolder{
		config: config,
		force:  force,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *initScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold scaffolds the Helm chart with the necessary files.
func (s *initScaffolder) Scaffold() error {
	log.Println("Generating Helm Chart to distribute project")

	// Extract Images scaffolded with DeployImage to add ENVVAR to the values
	imagesEnvVars := s.getDeployImagesEnvVars()

	// Extract webhooks from generated YAML files (generated by controller-gen)
	webhooks, err := extractWebhooksFromGeneratedFiles()
	if err != nil {
		return fmt.Errorf("failed to extract webhooks: %w", err)
	}

	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	buildScaffold := []machinery.Builder{
		&templates.HelmChart{},
		&templates.HelmValues{
			HasWebhooks:  len(webhooks) > 0,
			Webhooks:     webhooks,
			DeployImages: imagesEnvVars,
			Force:        s.force,
		},
		&templates.HelmIgnore{},
		&chart_templates.HelmHelpers{},
		&manager.ManagerDeployment{
			Force:        s.force,
			DeployImages: len(imagesEnvVars) > 0,
			HasWebhooks:  len(webhooks) > 0,
		},
		&templatescertmanager.Certificate{},
		&templatesmetrics.MetricsService{},
		&prometheus.Monitor{},
	}

	if len(webhooks) > 0 {
		buildScaffold = append(buildScaffold, &templateswebhooks.WebhookTemplate{})
		buildScaffold = append(buildScaffold, &templateswebhooks.WebhookService{})
	}

	if err := scaffold.Execute(buildScaffold...); err != nil {
		return fmt.Errorf("error scaffolding helm-chart manifests: %v", err)
	}

	// Copy relevant files from config/ to dist/chart/templates/
	err = s.copyConfigFiles()
	if err != nil {
		return fmt.Errorf("failed to copy manifests from config to dist/chart/templates/: %v", err)
	}

	return nil
}

// getDeployImagesEnvVars will return the values to append the envvars for projects
// which has the APIs scaffolded with DeployImage plugin
func (s *initScaffolder) getDeployImagesEnvVars() map[string]string {
	deployImages := make(map[string]string)

	pluginConfig := struct {
		Resources []struct {
			Kind    string            `json:"kind"`
			Options map[string]string `json:"options"`
		} `json:"resources"`
	}{}

	const deployImageKey = "deploy-image.go.kubebuilder.io/v1-alpha"
	err := s.config.DecodePluginConfig(deployImageKey, &pluginConfig)
	if err == nil {
		for _, res := range pluginConfig.Resources {
			image, ok := res.Options["image"]
			if ok {
				deployImages[strings.ToUpper(res.Kind)] = image
			}
		}
	}
	return deployImages
}

// Extract webhooks from manifests.yaml file
func extractWebhooksFromGeneratedFiles() ([]helm.WebhookYAML, error) {
	var webhooks []helm.WebhookYAML
	manifestFile := "config/webhook/manifests.yaml"
	if _, err := os.Stat(manifestFile); err == nil {
		content, err := os.ReadFile(manifestFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read manifests.yaml: %w", err)
		}

		// Process the content to extract webhooks
		webhooks = append(webhooks, extractWebhookYAML(content)...)
	} else {
		// Return empty if no webhooks were found
		return webhooks, nil
	}

	return webhooks, nil
}

// extractWebhookYAML parses the webhook YAML content and returns a list of WebhookYAML
func extractWebhookYAML(content []byte) []helm.WebhookYAML {
	var webhooks []helm.WebhookYAML

	type WebhookConfig struct {
		Kind     string `yaml:"kind"`
		Webhooks []struct {
			Name         string `yaml:"name"`
			ClientConfig struct {
				Service struct {
					Name      string `yaml:"name"`
					Namespace string `yaml:"namespace"`
					Path      string `yaml:"path"`
				} `yaml:"service"`
				CABundle string `yaml:"caBundle"`
			} `yaml:"clientConfig"`
			Rules                   []helm.WebhookRule `yaml:"rules"`
			FailurePolicy           string             `yaml:"failurePolicy"`
			SideEffects             string             `yaml:"sideEffects"`
			AdmissionReviewVersions []string           `yaml:"admissionReviewVersions"`
		} `yaml:"webhooks"`
	}

	// Split the input into different documents (to handle multiple YAML docs in one file)
	docs := strings.Split(string(content), "---")

	for _, doc := range docs {
		var webhookConfig WebhookConfig
		if err := yaml.Unmarshal([]byte(doc), &webhookConfig); err != nil {
			log.Errorf("Error unmarshalling webhook YAML: %v", err)
			continue
		}

		// Determine the webhook type (mutating or validating)
		webhookType := "unknown"
		if webhookConfig.Kind == "MutatingWebhookConfiguration" {
			webhookType = "mutating"
		} else if webhookConfig.Kind == "ValidatingWebhookConfiguration" {
			webhookType = "validating"
		}

		// Parse each webhook and append it to the result
		for _, webhook := range webhookConfig.Webhooks {
			for i := range webhook.Rules {
				// If apiGroups is empty, set it to [""] to ensure proper YAML output
				if len(webhook.Rules[i].APIGroups) == 0 {
					webhook.Rules[i].APIGroups = []string{""}
				}
			}
			webhooks = append(webhooks, helm.WebhookYAML{
				Name:                    webhook.Name,
				Type:                    webhookType,
				Path:                    webhook.ClientConfig.Service.Path,
				Rules:                   webhook.Rules,
				FailurePolicy:           webhook.FailurePolicy,
				SideEffects:             webhook.SideEffects,
				AdmissionReviewVersions: webhook.AdmissionReviewVersions,
			})
		}
	}
	return webhooks
}

// Helper function to copy files from config/ to dist/chart/templates/
func (s *initScaffolder) copyConfigFiles() error {
	configDirs := []struct {
		SrcDir  string
		DestDir string
		SubDir  string
	}{
		{"config/rbac", "dist/chart/templates/rbac", "rbac"},
		{"config/crd/bases", "dist/chart/templates/crd", "crd"},
		{"config/network-policy", "dist/chart/templates/network-policy", "networkPolicy"},
	}

	for _, dir := range configDirs {
		// Ensure destination directory exists
		if err := os.MkdirAll(dir.DestDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir.DestDir, err)
		}

		files, err := filepath.Glob(filepath.Join(dir.SrcDir, "*.yaml"))
		if err != nil {
			return err
		}

		for _, srcFile := range files {
			destFile := filepath.Join(dir.DestDir, filepath.Base(srcFile))
			err := copyFileWithHelmLogic(srcFile, destFile, dir.SubDir, s.config.GetProjectName())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFileWithHelmLogic reads the source file, modifies the content for Helm, applies patches
// to spec.conversion if applicable, and writes it to the destination
func copyFileWithHelmLogic(srcFile, destFile, subDir, projectName string) error {
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		log.Printf("Source file does not exist: %s", srcFile)
		return err
	}

	content, err := os.ReadFile(srcFile)
	if err != nil {
		log.Printf("Error reading source file: %s", srcFile)
		return err
	}

	contentStr := string(content)

	// Skip kustomization.yaml or kustomizeconfig.yaml files
	if strings.HasSuffix(srcFile, "kustomization.yaml") ||
		strings.HasSuffix(srcFile, "kustomizeconfig.yaml") {
		return nil
	}

	// Apply RBAC-specific replacements
	if subDir == "rbac" {
		contentStr = strings.Replace(contentStr,
			"name: controller-manager",
			fmt.Sprintf("name: %s-controller-manager", projectName), -1)
		contentStr = strings.Replace(contentStr,
			"name: metrics-reader",
			fmt.Sprintf("name: %s-metrics-reader", projectName), 1)
	}

	// Conditionally handle CRD patches and annotations for CRDs
	if subDir == "crd" {
		kind, group := extractKindAndGroupFromFileName(filepath.Base(srcFile))
		hasWebhookPatch := false

		// Retrieve patch content for the CRD's spec.conversion, if it exists
		patchContent, patchExists, err := getCRDPatchContent(kind, group)
		if err != nil {
			return err
		}

		// If patch content exists, inject it under spec.conversion with Helm conditional
		if patchExists {
			conversionSpec := extractConversionSpec(patchContent)
			contentStr = injectConversionSpecWithCondition(contentStr, conversionSpec)
			hasWebhookPatch = true
		}

		// Inject annotations after "annotations:" in a single block without extra spaces
		contentStr = injectAnnotations(contentStr, hasWebhookPatch)
	}

	// Remove existing labels if necessary
	contentStr = removeLabels(contentStr)

	// Replace namespace with Helm template variable
	contentStr = strings.ReplaceAll(contentStr, "namespace: system", "namespace: {{ .Release.Namespace }}")

	contentStr = strings.Replace(contentStr, "metadata:", `metadata:
  labels:
    {{- include "chart.labels" . | nindent 4 }}`, 1)

	var wrappedContent string
	if isMetricRBACFile(subDir, srcFile) {
		wrappedContent = fmt.Sprintf(
			"{{- if and .Values.rbac.enable .Values.metrics.enable }}\n%s{{- end -}}\n", contentStr)
	} else {
		wrappedContent = fmt.Sprintf(
			"{{- if .Values.%s.enable }}\n%s{{- end -}}\n", subDir, contentStr)
	}

	if err := os.MkdirAll(filepath.Dir(destFile), os.ModePerm); err != nil {
		return err
	}

	err = os.WriteFile(destFile, []byte(wrappedContent), os.ModePerm)
	if err != nil {
		log.Printf("Error writing destination file: %s", destFile)
		return err
	}

	log.Printf("Successfully copied %s to %s", srcFile, destFile)
	return nil
}

// extractKindAndGroupFromFileName extracts the kind and group from a CRD filename
func extractKindAndGroupFromFileName(fileName string) (kind, group string) {
	parts := strings.Split(fileName, "_")
	if len(parts) >= 2 {
		group = strings.Split(parts[0], ".")[0] // Extract group up to the first dot
		kind = strings.TrimSuffix(parts[1], ".yaml")
	}
	return kind, group
}

// getCRDPatchContent finds and reads the appropriate patch content for a given kind and group
func getCRDPatchContent(kind, group string) (string, bool, error) {
	// First, look for patches that contain both "webhook", the group, and kind in their filename
	groupKindPattern := fmt.Sprintf("config/crd/patches/webhook_*%s*%s*.yaml", group, kind)
	patchFiles, err := filepath.Glob(groupKindPattern)
	if err != nil {
		return "", false, fmt.Errorf("failed to list patches: %v", err)
	}

	// If no group-specific patch found, search for patches that contain only "webhook" and the kind
	if len(patchFiles) == 0 {
		kindOnlyPattern := fmt.Sprintf("config/crd/patches/webhook_*%s*.yaml", kind)
		patchFiles, err = filepath.Glob(kindOnlyPattern)
		if err != nil {
			return "", false, fmt.Errorf("failed to list patches: %v", err)
		}
	}

	// Read the first matching patch file (if any)
	if len(patchFiles) > 0 {
		patchContent, err := os.ReadFile(patchFiles[0])
		if err != nil {
			return "", false, fmt.Errorf("failed to read patch file %s: %v", patchFiles[0], err)
		}
		return string(patchContent), true, nil
	}

	return "", false, nil
}

// extractConversionSpec extracts only the conversion section from the patch content
func extractConversionSpec(patchContent string) string {
	specStart := strings.Index(patchContent, "conversion:")
	if specStart == -1 {
		return ""
	}
	return patchContent[specStart:]
}

// injectConversionSpecWithCondition inserts the conversion spec under the main spec field with Helm conditional
func injectConversionSpecWithCondition(contentStr, conversionSpec string) string {
	specPosition := strings.Index(contentStr, "spec:")
	if specPosition == -1 {
		return contentStr // No spec field found; return unchanged
	}
	conditionalSpec := fmt.Sprintf("\n  {{- if .Values.webhook.enable }}\n  %s\n  {{- end }}",
		strings.TrimRight(conversionSpec, "\n"))
	return contentStr[:specPosition+5] + conditionalSpec + contentStr[specPosition+5:]
}

// injectAnnotations inserts the required annotations after the "annotations:" field in a single block without
// extra spaces
func injectAnnotations(contentStr string, hasWebhookPatch bool) string {
	annotationsBlock := `
    {{- if .Values.certmanager.enable }}
    cert-manager.io/inject-ca-from: "{{ .Release.Namespace }}/serving-cert"
    {{- end }}
    {{- if .Values.crd.keep }}
    "helm.sh/resource-policy": keep
    {{- end }}`
	if hasWebhookPatch {
		return strings.Replace(contentStr, "annotations:", "annotations:"+annotationsBlock, 1)
	}

	// Apply only resource policy if no webhook patch
	resourcePolicy := `
    {{- if .Values.crd.keep }}
    "helm.sh/resource-policy": keep
    {{- end }}`
	return strings.Replace(contentStr, "annotations:", "annotations:"+resourcePolicy, 1)
}

// isMetricRBACFile checks if the file is in the "rbac"
// subdirectory and matches one of the metric-related RBAC filenames
func isMetricRBACFile(subDir, srcFile string) bool {
	return subDir == "rbac" && (strings.HasSuffix(srcFile, "metrics_auth_role.yaml") ||
		strings.HasSuffix(srcFile, "metrics_auth_role_binding.yaml") ||
		strings.HasSuffix(srcFile, "metrics_reader_role.yaml"))
}

// removeLabels removes any existing labels section from the content
func removeLabels(content string) string {
	labelRegex := `(?m)^  labels:\n(?:    [^\n]+\n)*`
	re := regexp.MustCompile(labelRegex)

	return re.ReplaceAllString(content, "")
}
