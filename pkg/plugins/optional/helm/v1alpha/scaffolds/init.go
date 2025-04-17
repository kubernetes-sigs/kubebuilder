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
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates"
	charttemplates "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates"
	templatescertmanager "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/cert-manager"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/manager"
	templatesmetrics "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/metrics"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/prometheus"
	templateswebhooks "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/chart-templates/webhook"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds/internal/templates/github"
)

var _ plugins.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config config.Config

	fs machinery.Filesystem

	force bool
}

// Define constants for repeated strings
const (
	deploymentKind        = "Deployment"
	managerContainerName  = "manager"
	controllerManagerName = "controller-manager"
)

// NewInitHelmScaffolder returns a new Scaffolder for HelmPlugin
func NewInitHelmScaffolder(cfg config.Config, force bool) plugins.Scaffolder {
	return &initScaffolder{
		config: cfg,
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

	imagesEnvVars := s.getDeployImagesEnvVars()

	// Extract manager values when --force flag is used
	var managerValues map[string]interface{}
	var extractErr error
	if s.force {
		// First try to get values directly from manager.yaml
		managerValues, extractErr = s.extractManagerValues()
		if extractErr != nil {
			log.Warnf("Failed to extract manager values from manager.yaml: %v", extractErr)

			// If that fails, try to get values from kustomization patches
			managerValues, extractErr = s.extractManagerValuesFromKustomization()
			if extractErr != nil {
				log.Warnf("Failed to extract manager values from kustomization: %v", extractErr)

				// As a last resort, try to build the manifests using kustomize
				managerValues, extractErr = s.extractManagerValuesUsingKustomize()
				if extractErr != nil {
					log.Warnf("Failed to extract manager values using kustomize: %v", extractErr)
				}
			}
		}
	}

	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	// Found webhooks by looking at the config our scaffolds files
	mutatingWebhooks, validatingWebhooks, err := s.extractWebhooksFromGeneratedFiles()
	if err != nil {
		return fmt.Errorf("failed to extract webhooks: %w", err)
	}
	hasWebhooks := hasWebhooksWith(s.config) || (len(mutatingWebhooks) > 0 && len(validatingWebhooks) > 0)

	buildScaffold := []machinery.Builder{
		&github.HelmChartCI{},
		&templates.HelmChart{},
		&templates.HelmValues{
			HasWebhooks:   hasWebhooks,
			DeployImages:  imagesEnvVars,
			Force:         s.force,
			ManagerValues: managerValues,
		},
		&templates.HelmIgnore{},
		&charttemplates.HelmHelpers{},
		&manager.Deployment{
			Force:        s.force,
			DeployImages: len(imagesEnvVars) > 0,
			HasWebhooks:  hasWebhooks,
		},
		&templatescertmanager.Certificate{HasWebhooks: hasWebhooks},
		&templatesmetrics.Service{},
		&prometheus.Monitor{},
	}

	if len(mutatingWebhooks) > 0 || len(validatingWebhooks) > 0 {
		buildScaffold = append(buildScaffold,
			&templateswebhooks.Template{
				MutatingWebhooks:   mutatingWebhooks,
				ValidatingWebhooks: validatingWebhooks,
			},
		)
	}

	if hasWebhooks {
		buildScaffold = append(buildScaffold,
			&templateswebhooks.Service{},
		)
	}

	if err = scaffold.Execute(buildScaffold...); err != nil {
		return fmt.Errorf("error scaffolding helm-chart manifests: %w", err)
	}

	// Copy relevant files from config/ to dist/chart/templates/
	err = s.copyConfigFiles()
	if err != nil {
		return fmt.Errorf("failed to copy manifests from config to dist/chart/templates/: %w", err)
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

	err := s.config.DecodePluginConfig(plugin.KeyFor(v1alpha1.Plugin{}), &pluginConfig)
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

// extractWebhooksFromGeneratedFiles parses the files generated by controller-gen under
// config/webhooks and created Mutating and Validating helper structures to
// generate the webhook manifest for the helm-chart
func (s *initScaffolder) extractWebhooksFromGeneratedFiles() (mutatingWebhooks []templateswebhooks.DataWebhook,
	validatingWebhooks []templateswebhooks.DataWebhook, err error,
) {
	manifestFile := "config/webhook/manifests.yaml"

	if _, err = os.Stat(manifestFile); os.IsNotExist(err) {
		log.Printf("webhook manifests were not found at %s", manifestFile)
		return nil, nil, nil
	}

	content, err := os.ReadFile(manifestFile)
	if err != nil {
		return nil, nil,
			fmt.Errorf("failed to read %q: %w", manifestFile, err)
	}

	docs := strings.Split(string(content), "---")
	for _, doc := range docs {
		var webhookConfig struct {
			Kind     string `yaml:"kind"`
			Webhooks []struct {
				Name         string `yaml:"name"`
				ClientConfig struct {
					Service struct {
						Name      string `yaml:"name"`
						Namespace string `yaml:"namespace"`
						Path      string `yaml:"path"`
					} `yaml:"service"`
				} `yaml:"clientConfig"`
				Rules                   []templateswebhooks.DataWebhookRule `yaml:"rules"`
				FailurePolicy           string                              `yaml:"failurePolicy"`
				SideEffects             string                              `yaml:"sideEffects"`
				AdmissionReviewVersions []string                            `yaml:"admissionReviewVersions"`
			} `yaml:"webhooks"`
		}

		if err := yaml.Unmarshal([]byte(doc), &webhookConfig); err != nil {
			log.Errorf("fail to unmarshalling webhook YAML: %v", err)
			continue
		}

		for _, w := range webhookConfig.Webhooks {
			for i := range w.Rules {
				if len(w.Rules[i].APIGroups) == 0 {
					w.Rules[i].APIGroups = []string{""}
				}
			}
			webhook := templateswebhooks.DataWebhook{
				Name:                    w.Name,
				ServiceName:             fmt.Sprintf("%s-webhook-service", s.config.GetProjectName()),
				Path:                    w.ClientConfig.Service.Path,
				FailurePolicy:           w.FailurePolicy,
				SideEffects:             w.SideEffects,
				AdmissionReviewVersions: w.AdmissionReviewVersions,
				Rules:                   w.Rules,
			}

			switch webhookConfig.Kind {
			case "MutatingWebhookConfiguration":
				mutatingWebhooks = append(mutatingWebhooks, webhook)
			case "ValidatingWebhookConfiguration":
				validatingWebhooks = append(validatingWebhooks, webhook)
			}
		}
	}

	return mutatingWebhooks, validatingWebhooks, nil
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
		// Check if the source directory exists
		if _, err := os.Stat(dir.SrcDir); os.IsNotExist(err) {
			// Skip if the source directory does not exist
			continue
		}

		files, err := filepath.Glob(filepath.Join(dir.SrcDir, "*.yaml"))
		if err != nil {
			return fmt.Errorf("failed finding files in %q: %w", dir.SrcDir, err)
		}

		// Skip processing if the directory is empty (no matching files)
		if len(files) == 0 {
			continue
		}

		// Ensure destination directory exists
		if err := os.MkdirAll(dir.DestDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir.DestDir, err)
		}

		for _, srcFile := range files {
			destFile := filepath.Join(dir.DestDir, filepath.Base(srcFile))

			hasConvertionalWebhook := false
			if hasWebhooksWith(s.config) {
				resources, err := s.config.GetResources()
				if err != nil {
					break
				}
				for _, res := range resources {
					if res.HasConversionWebhook() {
						hasConvertionalWebhook = true
						break
					}
				}
			}

			err := copyFileWithHelmLogic(srcFile, destFile, dir.SubDir, s.config.GetProjectName(), hasConvertionalWebhook)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFileWithHelmLogic reads the source file, modifies the content for Helm, applies patches
// to spec.conversion if applicable, and writes it to the destination
func copyFileWithHelmLogic(srcFile, destFile, subDir, projectName string, hasConvertionalWebhook bool) error {
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		log.Printf("Source file does not exist: %s", srcFile)
		return fmt.Errorf("source file does not exist %q: %w", srcFile, err)
	}

	content, err := os.ReadFile(srcFile)
	if err != nil {
		log.Printf("Error reading source file: %s", srcFile)
		return fmt.Errorf("failed to read file %q: %w", srcFile, err)
	}

	contentStr := string(content)

	// Skip kustomization.yaml or kustomizeconfig.yaml files
	if strings.HasSuffix(srcFile, "kustomization.yaml") ||
		strings.HasSuffix(srcFile, "kustomizeconfig.yaml") {
		return nil
	}

	// Apply RBAC-specific replacements
	if subDir == "rbac" {
		contentStr = strings.ReplaceAll(contentStr,
			"name: controller-manager",
			"name: {{ .Values.controllerManager.serviceAccountName }}")
		contentStr = strings.Replace(contentStr,
			"name: metrics-reader",
			fmt.Sprintf("name: %s-metrics-reader", projectName), 1)

		contentStr = strings.ReplaceAll(contentStr,
			"name: metrics-auth-role",
			fmt.Sprintf("name: %s-metrics-auth-role", projectName))
		contentStr = strings.Replace(contentStr,
			"name: metrics-auth-rolebinding",
			fmt.Sprintf("name: %s-metrics-auth-rolebinding", projectName), 1)

		if strings.Contains(contentStr, ".Values.controllerManager.serviceAccountName") &&
			strings.Contains(contentStr, "kind: ServiceAccount") &&
			!strings.Contains(contentStr, "RoleBinding") {
			// The generated Service Account does not have the annotations field so we must add it.
			contentStr = strings.Replace(contentStr,
				"metadata:", `metadata:
  {{- if and .Values.controllerManager.serviceAccount .Values.controllerManager.serviceAccount.annotations }}
  annotations:
    {{- range $key, $value := .Values.controllerManager.serviceAccount.annotations }}
    {{ $key }}: {{ $value }}
    {{- end }}
  {{- end }}`, 1)
		}
		contentStr = strings.ReplaceAll(contentStr,
			"name: leader-election-role",
			fmt.Sprintf("name: %s-leader-election-role", projectName))
		contentStr = strings.Replace(contentStr,
			"name: leader-election-rolebinding",
			fmt.Sprintf("name: %s-leader-election-rolebinding", projectName), 1)
		contentStr = strings.ReplaceAll(contentStr,
			"name: manager-role",
			fmt.Sprintf("name: %s-manager-role", projectName))
		contentStr = strings.Replace(contentStr,
			"name: manager-rolebinding",
			fmt.Sprintf("name: %s-manager-rolebinding", projectName), 1)

		// The generated files do not include the namespace
		if strings.Contains(contentStr, "leader-election-rolebinding") ||
			strings.Contains(contentStr, "leader-election-role") {
			namespace := `
  namespace: {{ .Release.Namespace }}`
			contentStr = strings.Replace(contentStr, "metadata:", "metadata:"+namespace, 1)
		}
	}

	// Conditionally handle CRD patches and annotations for CRDs
	if subDir == "crd" {
		kind, group := extractKindAndGroupFromFileName(filepath.Base(srcFile))
		hasWebhookPatch := false

		// Retrieve patch content for the CRD's spec.conversion, if it exists
		patchContent, patchExists, errPatch := getCRDPatchContent(kind, group)
		if errPatch != nil {
			return errPatch
		}

		// If patch content exists, inject it under spec.conversion with Helm conditional
		if patchExists {
			conversionSpec := extractConversionSpec(patchContent)
			// Projects scaffolded with old Kubebuilder versions does not have the conversion
			// webhook properly generated because before 4.4.0 this feature was not fully addressed.
			// The patch was added by default when should not. See the related fixes:
			//
			// Issue fixed in release 4.3.1: (which will cause the injection of webhook conditionals for projects without
			// conversion webhooks)
			// (kustomize/v2, go/v4): Corrected the generation of manifests under config/crd/patches
			// to ensure the /convert service patch is only created for webhooks configured with --conversion. (#4280)
			//
			// Conversion webhook fully fixed in release 4.4.0:
			// (kustomize/v2, go/v4): Fixed CA injection for conversion webhooks. Previously, the CA injection
			// was applied incorrectly to all CRDs instead of only conversion types. The issue dates back to release 3.5.0
			// due to kustomize/v2-alpha changes. Now, conversion webhooks are properly generated. (#4254, #4282)
			if len(conversionSpec) > 0 && !hasConvertionalWebhook {
				log.Warn("\n" +
					"============================================================\n" +
					"| [WARNING] Webhook Patch Issue Detected                   |\n" +
					"============================================================\n" +
					"Webhook patch found, but no conversion webhook is configured for this project.\n\n" +
					"Note: Older scaffolds have an issue where the conversion webhook patch was \n" +
					"      scaffolded by default, and conversion webhook injection was not properly limited \n" +
					"      to specific CRDs.\n\n" +
					"Recommended Action:\n" +
					"   - Upgrade your project to the latest available version.\n" +
					"   - Consider using the 'alpha generate' command.\n\n" +
					"The cert-manager injection and webhook conversion patch found for CRDs will\n" +
					"be skipped and NOT added to the Helm chart.\n" +
					"============================================================")

				hasWebhookPatch = false
			} else {
				contentStr = injectConversionSpecWithCondition(contentStr, conversionSpec)
				hasWebhookPatch = true
			}
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

	if err = os.MkdirAll(filepath.Dir(destFile), os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %q: %w", filepath.Dir(destFile), err)
	}

	err = os.WriteFile(destFile, []byte(wrappedContent), os.ModePerm)
	if err != nil {
		log.Printf("Error writing destination file: %s", destFile)
		return fmt.Errorf("error writing destination file %q: %w", destFile, err)
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
		return "", false, fmt.Errorf("failed to list patches: %w", err)
	}

	// If no group-specific patch found, search for patches that contain only "webhook" and the kind
	if len(patchFiles) == 0 {
		kindOnlyPattern := fmt.Sprintf("config/crd/patches/webhook_*%s*.yaml", kind)
		patchFiles, err = filepath.Glob(kindOnlyPattern)
		if err != nil {
			return "", false, fmt.Errorf("failed to list patches: %w", err)
		}
	}

	// Read the first matching patch file (if any)
	if len(patchFiles) > 0 {
		patchContent, err := os.ReadFile(patchFiles[0])
		if err != nil {
			return "", false, fmt.Errorf("failed to read patch file %q: %w", patchFiles[0], err)
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

func hasWebhooksWith(c config.Config) bool {
	// Get the list of resources
	resources, err := c.GetResources()
	if err != nil {
		return false // If there's an error getting resources, assume no webhooks
	}

	for _, res := range resources {
		if res.HasDefaultingWebhook() || res.HasValidationWebhook() || res.HasConversionWebhook() {
			return true
		}
	}

	return false
}

// findManagerFile attempts to locate the manager.yaml file in common locations
func (s *initScaffolder) findManagerFile() (string, []byte, error) {
	managerLocations := []string{
		"config/manager/manager.yaml",
		"config/default/manager_auth_proxy_patch.yaml",
		"kustomization/manager/manager.yaml",
	}

	for _, location := range managerLocations {
		if _, statErr := os.Stat(location); statErr == nil {
			content, err := os.ReadFile(location)
			if err == nil {
				return location, content, nil
			}
		}
	}

	return "", nil, fmt.Errorf("manager file not found in any of the expected locations")
}

// findDeploymentInYAML looks through YAML documents to find a Deployment
func (s *initScaffolder) findDeploymentInYAML(content []byte) (map[string]interface{}, error) {
	docs := strings.Split(string(content), "---")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var docMap map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &docMap); err != nil {
			continue
		}

		kind, found := docMap["kind"].(string)
		if found && kind == deploymentKind {
			return docMap, nil
		}
	}

	return nil, fmt.Errorf("deployment not found in YAML content")
}

// extractDeploymentValues extracts configuration values from a deployment map
func (s *initScaffolder) extractDeploymentValues(deployment map[string]interface{}) map[string]interface{} {
	values := make(map[string]interface{})

	// Extract spec
	spec, found := deployment["spec"].(map[string]interface{})
	if !found {
		return values
	}

	// Extract replicas
	s.extractReplicasFromSpec(spec, values)

	// Get template.spec
	template, found := spec["template"].(map[string]interface{})
	if !found {
		return values
	}

	templateSpec, found := template["spec"].(map[string]interface{})
	if !found {
		return values
	}

	// Extract security context and termination grace period
	s.extractPodSettings(templateSpec, values)

	// Extract container values
	s.extractContainerValues(templateSpec, values)

	return values
}

// extractReplicasFromSpec extracts replica count from the spec
func (s *initScaffolder) extractReplicasFromSpec(spec map[string]interface{}, values map[string]interface{}) {
	if replicas, found := spec["replicas"].(int); found {
		values["replicas"] = replicas
	} else if replicas, found := spec["replicas"].(float64); found {
		values["replicas"] = int(replicas)
	}
}

// extractPodSettings extracts pod-level settings from the template.spec
func (s *initScaffolder) extractPodSettings(templateSpec map[string]interface{}, values map[string]interface{}) {
	// Extract termination grace period
	if terminationGracePeriod, found := templateSpec["terminationGracePeriodSeconds"].(int); found {
		values["terminationGracePeriodSeconds"] = terminationGracePeriod
	} else if terminationGracePeriod, found := templateSpec["terminationGracePeriodSeconds"].(float64); found {
		values["terminationGracePeriodSeconds"] = int(terminationGracePeriod)
	}

	// Extract security context
	if securityContext, found := templateSpec["securityContext"].(map[string]interface{}); found {
		values["securityContext"] = securityContext
	}
}

// extractContainerValues extracts values from the manager container
func (s *initScaffolder) extractContainerValues(templateSpec map[string]interface{}, values map[string]interface{}) {
	containers, found := templateSpec["containers"].([]interface{})
	if !found || len(containers) == 0 {
		return
	}

	for _, c := range containers {
		container, found := c.(map[string]interface{})
		if !found {
			continue
		}

		containerName, found := container["name"].(string)
		if !found || (containerName != managerContainerName && containerName != controllerManagerName) {
			continue
		}

		// Extract container settings
		s.extractContainerSettings(container, values)
		break
	}
}

// extractContainerSettings extracts settings from a container
func (s *initScaffolder) extractContainerSettings(container map[string]interface{}, values map[string]interface{}) {
	// Extract args
	if args, found := container["args"].([]interface{}); found && len(args) > 0 {
		stringArgs := make([]string, len(args))
		for i, arg := range args {
			if strArg, found := arg.(string); found {
				stringArgs[i] = strArg
			}
		}
		values["args"] = stringArgs
	}

	// Extract resources
	if resources, found := container["resources"].(map[string]interface{}); found {
		values["resources"] = resources
	}

	// Extract probes
	if livenessProbe, found := container["livenessProbe"].(map[string]interface{}); found {
		values["livenessProbe"] = livenessProbe
	}

	if readinessProbe, found := container["readinessProbe"].(map[string]interface{}); found {
		values["readinessProbe"] = readinessProbe
	}
}

// extractManagerValues reads the manager.yaml file and extracts values needed for the Helm values.yaml
func (s *initScaffolder) extractManagerValues() (map[string]interface{}, error) {
	// Find manager file
	_, content, err := s.findManagerFile()
	if err != nil {
		return nil, err
	}

	// Find deployment in YAML
	deployment, err := s.findDeploymentInYAML(content)
	if err != nil {
		s.dumpYAMLContent(content)
		return nil, err
	}

	// Extract values from deployment
	return s.extractDeploymentValues(deployment), nil
}

// extractManagerValuesFromKustomization attempts to extract manager values from kustomization.yaml patches
func (s *initScaffolder) extractManagerValuesFromKustomization() (map[string]interface{}, error) {
	patches, err := s.findKustomizationPatches()
	if err != nil {
		return nil, err
	}

	values := make(map[string]interface{})

	// Process each patch
	for _, patch := range patches {
		patchValues, err := s.extractValuesFromPatchFile(patch)
		if err == nil {
			// Merge patch values into the main values map
			for k, v := range patchValues {
				values[k] = v
			}
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no manager values found in kustomization patches")
	}

	return values, nil
}

// findKustomizationPatches finds and returns all manager-related patches from kustomization.yaml
func (s *initScaffolder) findKustomizationPatches() ([]string, error) {
	kustomizationFile := "config/default/kustomization.yaml"

	// Check if kustomization file exists
	if _, err := os.Stat(kustomizationFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("kustomization file not found at %s", kustomizationFile)
	}

	content, err := os.ReadFile(kustomizationFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read kustomization file: %w", err)
	}

	patches, err := s.parseKustomizationPatches(content, kustomizationFile)
	if err != nil {
		return nil, err
	}

	return patches, nil
}

// parseKustomizationPatches parses a kustomization.yaml file and returns a list of manager-related patches
func (s *initScaffolder) parseKustomizationPatches(content []byte, kustomizationFile string) ([]string, error) {
	// Parse kustomization.yaml
	var kustomization struct {
		Patches []struct {
			Path   string `yaml:"path"`
			Target struct {
				Kind string `yaml:"kind"`
				Name string `yaml:"name"`
			} `yaml:"target"`
		} `yaml:"patches"`
		PatchesStrategicMerge []string `yaml:"patchesStrategicMerge"`
	}

	if err := yaml.Unmarshal(content, &kustomization); err != nil {
		return nil, fmt.Errorf("failed to parse kustomization YAML: %w", err)
	}

	var patches []string

	// Check new-style patches
	for _, patch := range kustomization.Patches {
		if patch.Target.Kind == deploymentKind &&
			(patch.Target.Name == controllerManagerName || patch.Target.Name == managerContainerName) {
			patches = append(patches, patch.Path)
		}
	}

	// Check old-style patchesStrategicMerge
	for _, patchPath := range kustomization.PatchesStrategicMerge {
		// Only process patches that might be related to the manager deployment
		if strings.Contains(patchPath, "manager") {
			patches = append(patches, filepath.Join(filepath.Dir(kustomizationFile), patchPath))
		}
	}

	return patches, nil
}

// extractValuesFromPatchFile extracts values from a patch file
func (s *initScaffolder) extractValuesFromPatchFile(patchPath string) (map[string]interface{}, error) {
	if _, err := os.Stat(patchPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("patch file not found: %s", patchPath)
	}

	content, err := os.ReadFile(patchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read patch file: %w", err)
	}

	var patch map[string]interface{}
	if err := yaml.Unmarshal(content, &patch); err != nil {
		return nil, fmt.Errorf("failed to parse patch YAML: %w", err)
	}

	// Check if this is a Deployment
	kind, ok := patch["kind"].(string)
	if !ok || kind != "Deployment" {
		return nil, fmt.Errorf("patch is not for a Deployment")
	}

	// Extract values similar to extractManagerValues
	values := make(map[string]interface{})
	spec, ok := patch["spec"].(map[string]interface{})
	if !ok {
		return values, nil
	}

	// Extract replicas
	if replicas, ok := spec["replicas"].(int); ok {
		values["replicas"] = replicas
	} else if replicas, ok := spec["replicas"].(float64); ok {
		values["replicas"] = int(replicas)
	}

	// Extract more values if needed...
	// This is simplified, but you could extract other values similar to extractManagerValues

	return values, nil
}

// extractManagerValuesUsingKustomize attempts to extract manager values by running kustomize build
func (s *initScaffolder) extractManagerValuesUsingKustomize() (map[string]interface{}, error) {
	// Check if kustomize is available
	_, err := exec.LookPath("kustomize")
	if err != nil {
		return nil, fmt.Errorf("kustomize command not found: %w", err)
	}

	// Get manifests from kustomize
	output, err := s.runKustomizeBuild()
	if err != nil {
		return nil, err
	}

	// Find manager deployment in manifests
	managerYAML, err := s.findManagerDeploymentInManifests(output)
	if err != nil {
		return nil, err
	}

	// Extract values from deployment
	return s.extractDeploymentValues(managerYAML), nil
}

// runKustomizeBuild runs kustomize build in various directories and returns the output
func (s *initScaffolder) runKustomizeBuild() ([]byte, error) {
	kustomizeDirs := []string{
		"config/default",
		"config/manager",
		"config",
	}

	for _, dir := range kustomizeDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		cmd := exec.Command("kustomize", "build", dir)
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			return output, nil
		}
	}

	return nil, fmt.Errorf("failed to build manifests using kustomize")
}

// findManagerDeploymentInManifests searches for the manager deployment in kustomize output
func (s *initScaffolder) findManagerDeploymentInManifests(output []byte) (map[string]interface{}, error) {
	docs := strings.Split(string(output), "---")
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var manifest map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			continue
		}

		// Look for the manager deployment
		kind, _ := manifest["kind"].(string)
		if kind != deploymentKind {
			continue
		}

		metadata, found := manifest["metadata"].(map[string]interface{})
		if !found {
			continue
		}

		name, found := metadata["name"].(string)
		if !found {
			continue
		}

		if name == controllerManagerName || strings.Contains(name, "manager") {
			return manifest, nil
		}
	}

	return nil, fmt.Errorf("manager deployment not found in generated manifests")
}

// dumpYAMLContent is a helper function to print the YAML content in a readable format
func (s *initScaffolder) dumpYAMLContent(content []byte) {
	// Convert content to string and print each line with line number for debugging
	lines := strings.Split(string(content), "\n")
	log.Warn("YAML content that failed to parse:")
	for i, line := range lines {
		log.Warnf("%3d: %s", i+1, line)
	}
}
