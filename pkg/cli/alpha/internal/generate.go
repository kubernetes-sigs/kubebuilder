/*
Copyright 2023 The Kubernetes Authors.

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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/grafana/v1alpha"
	hemlv1alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha"
)

// Generate store the required info for the command
type Generate struct {
	InputDir  string
	OutputDir string
}

// Generate handles the migration and scaffolding process.
func (opts *Generate) Generate() error {
	projectConfig, err := loadProjectConfig(opts.InputDir)
	if err != nil {
		return err
	}

	if opts.OutputDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		opts.OutputDir = cwd
		if _, err := os.Stat(opts.OutputDir); err == nil {
			log.Warn("Using current working directory to re-scaffold the project")
			log.Warn("This directory will be cleaned up and all files removed before the re-generation")

			// Ensure we clean the correct directory
			log.Info("Cleaning directory:", opts.OutputDir)

			// Use an absolute path to target files directly
			cleanupCmd := fmt.Sprintf("rm -rf %s/*", opts.OutputDir)
			err = util.RunCmd("Running cleanup", "sh", "-c", cleanupCmd)
			if err != nil {
				log.Error("Cleanup failed:", err)
				return err
			}
		}
	}

	if err := createDirectory(opts.OutputDir); err != nil {
		return err
	}

	if err := changeWorkingDirectory(opts.OutputDir); err != nil {
		return err
	}

	if err := kubebuilderInit(projectConfig); err != nil {
		return err
	}

	if err := kubebuilderEdit(projectConfig); err != nil {
		return err
	}

	if err := kubebuilderCreate(projectConfig); err != nil {
		return err
	}

	if err := migrateGrafanaPlugin(projectConfig, opts.InputDir, opts.OutputDir); err != nil {
		return err
	}

	if hasHelmPlugin(projectConfig) {
		if err := kubebuilderHelmEdit(); err != nil {
			return err
		}
	}
	return migrateDeployImagePlugin(projectConfig)
}

// Validate ensures the options are valid and kubebuilder is installed.
func (opts *Generate) Validate() error {
	var err error
	opts.InputDir, err = getInputPath(opts.InputDir)
	if err != nil {
		return err
	}

	_, err = exec.LookPath("kubebuilder")
	if err != nil {
		return fmt.Errorf("kubebuilder not found in the path: %w", err)
	}

	return nil
}

// Helper function to load the project configuration.
func loadProjectConfig(inputDir string) (store.Store, error) {
	projectConfig := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := projectConfig.LoadFrom(fmt.Sprintf("%s/%s", inputDir, yaml.DefaultPath)); err != nil {
		return nil, fmt.Errorf("failed to load PROJECT file: %w", err)
	}
	return projectConfig, nil
}

// Helper function to create the output directory.
func createDirectory(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}
	return nil
}

// Helper function to change the current working directory.
func changeWorkingDirectory(outputDir string) error {
	if err := os.Chdir(outputDir); err != nil {
		return fmt.Errorf("failed to change the working directory to %s: %w", outputDir, err)
	}
	return nil
}

// Initializes the project with Kubebuilder.
func kubebuilderInit(store store.Store) error {
	args := append([]string{"init"}, getInitArgs(store)...)
	return util.RunCmd("kubebuilder init", "kubebuilder", args...)
}

// Edits the project to enable or disable multigroup layout.
func kubebuilderEdit(store store.Store) error {
	if store.Config().IsMultiGroup() {
		args := []string{"edit", "--multigroup"}
		return util.RunCmd("kubebuilder edit", "kubebuilder", args...)
	}
	return nil
}

// Creates APIs and Webhooks for the project.
func kubebuilderCreate(store store.Store) error {
	resources, err := store.Config().GetResources()
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	// First, scaffold all APIs
	for _, r := range resources {
		if err := createAPI(r); err != nil {
			return fmt.Errorf("failed to create API for %s/%s/%s: %w", r.Group, r.Version, r.Kind, err)
		}
	}

	// Then, scaffold all webhooks
	// We cannot create a webhook for an API that does not exist
	for _, r := range resources {
		if err := createWebhook(r); err != nil {
			return fmt.Errorf("failed to create webhook for %s/%s/%s: %w", r.Group, r.Version, r.Kind, err)
		}
	}

	return nil
}

// Migrates the Grafana plugin.
func migrateGrafanaPlugin(store store.Store, src, des string) error {
	var grafanaPlugin struct{}
	err := store.Config().DecodePluginConfig(plugin.KeyFor(v1alpha.Plugin{}), grafanaPlugin)
	if errors.As(err, &config.PluginKeyNotFoundError{}) {
		log.Info("Grafana plugin not found, skipping migration")
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to decode grafana plugin config: %w", err)
	}

	if err := kubebuilderGrafanaEdit(); err != nil {
		return err
	}

	if err := grafanaConfigMigrate(src, des); err != nil {
		return err
	}

	return kubebuilderGrafanaEdit()
}

// Migrates the Deploy Image plugin.
func migrateDeployImagePlugin(store store.Store) error {
	var deployImagePlugin v1alpha1.PluginConfig
	err := store.Config().DecodePluginConfig(plugin.KeyFor(v1alpha1.Plugin{}), &deployImagePlugin)
	if errors.As(err, &config.PluginKeyNotFoundError{}) {
		log.Info("Deploy-image plugin not found, skipping migration")
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to decode deploy-image plugin config: %w", err)
	}

	for _, r := range deployImagePlugin.Resources {
		if err := createAPIWithDeployImage(r); err != nil {
			return fmt.Errorf("failed to create API with deploy-image: %w", err)
		}
	}

	return nil
}

// Creates an API with Deploy Image plugin.
func createAPIWithDeployImage(resource v1alpha1.ResourceData) error {
	args := append([]string{"create", "api"}, getGVKFlagsFromDeployImage(resource)...)
	args = append(args, getDeployImageOptions(resource)...)
	return util.RunCmd("kubebuilder create api", "kubebuilder", args...)
}

// Helper function to get input path.
func getInputPath(inputPath string) (string, error) {
	if inputPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		inputPath = cwd
	}
	projectPath := fmt.Sprintf("%s/%s", inputPath, yaml.DefaultPath)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return "", fmt.Errorf("project path %s does not exist: %w", projectPath, err)
	}
	return inputPath, nil
}

// Helper function to get Init arguments for Kubebuilder.
func getInitArgs(store store.Store) []string {
	var args []string
	plugins := store.Config().GetPluginChain()

	// Define outdated plugin versions that need replacement
	outdatedPlugins := map[string]string{
		"go.kubebuilder.io/v3":       "go.kubebuilder.io/v4",
		"go.kubebuilder.io/v3-alpha": "go.kubebuilder.io/v4",
		"go.kubebuilder.io/v2":       "go.kubebuilder.io/v4",
	}

	// Replace outdated plugins and exit after the first replacement
	for i, plg := range plugins {
		if newPlugin, exists := outdatedPlugins[plg]; exists {
			log.Warnf("We checked that your PROJECT file is configured with the layout '%s', which is no longer supported.\n"+
				"However, we will try our best to re-generate the project using '%s'.", plg, newPlugin)
			plugins[i] = newPlugin
			break
		}
	}

	if len(plugins) > 0 {
		args = append(args, "--plugins", strings.Join(plugins, ","))
	}
	if domain := store.Config().GetDomain(); domain != "" {
		args = append(args, "--domain", domain)
	}
	if repo := store.Config().GetRepository(); repo != "" {
		args = append(args, "--repo", repo)
	}
	return args
}

// Gets the GVK flags for a resource.
func getGVKFlags(resource resource.Resource) []string {
	var args []string
	if resource.Plural != "" {
		args = append(args, "--plural", resource.Plural)
	}
	if resource.Group != "" {
		args = append(args, "--group", resource.Group)
	}
	if resource.Version != "" {
		args = append(args, "--version", resource.Version)
	}
	if resource.Kind != "" {
		args = append(args, "--kind", resource.Kind)
	}
	return args
}

// Gets the GVK flags for a Deploy Image resource.
func getGVKFlagsFromDeployImage(resource v1alpha1.ResourceData) []string {
	var args []string
	if resource.Group != "" {
		args = append(args, "--group", resource.Group)
	}
	if resource.Version != "" {
		args = append(args, "--version", resource.Version)
	}
	if resource.Kind != "" {
		args = append(args, "--kind", resource.Kind)
	}
	return args
}

// Gets the options for a Deploy Image resource.
func getDeployImageOptions(resource v1alpha1.ResourceData) []string {
	var args []string
	if resource.Options.Image != "" {
		args = append(args, fmt.Sprintf("--image=%s", resource.Options.Image))
	}
	if resource.Options.ContainerCommand != "" {
		args = append(args, fmt.Sprintf("--image-container-command=%s", resource.Options.ContainerCommand))
	}
	if resource.Options.ContainerPort != "" {
		args = append(args, fmt.Sprintf("--image-container-port=%s", resource.Options.ContainerPort))
	}
	if resource.Options.RunAsUser != "" {
		args = append(args, fmt.Sprintf("--run-as-user=%s", resource.Options.RunAsUser))
	}
	args = append(args, fmt.Sprintf("--plugins=%s", plugin.KeyFor(v1alpha1.Plugin{})))
	return args
}

// Creates an API resource.
func createAPI(resource resource.Resource) error {
	args := append([]string{"create", "api"}, getGVKFlags(resource)...)
	args = append(args, getAPIResourceFlags(resource)...)

	// Add the external API path flag if the resource is external
	if resource.IsExternal() {
		args = append(args, "--external-api-path", resource.Path)
		args = append(args, "--external-api-domain", resource.Domain)
	}

	return util.RunCmd("kubebuilder create api", "kubebuilder", args...)
}

// Gets flags for API resource creation.
func getAPIResourceFlags(resource resource.Resource) []string {
	var args []string

	if resource.API == nil || resource.API.IsEmpty() {
		args = append(args, "--resource=false")
	} else {
		args = append(args, "--resource")
		if resource.API.Namespaced {
			args = append(args, "--namespaced")
		} else {
			args = append(args, "--namespaced=false")
		}
	}
	if resource.Controller {
		args = append(args, "--controller")
	} else {
		args = append(args, "--controller=false")
	}
	return args
}

// Creates a webhook resource.
func createWebhook(resource resource.Resource) error {
	if resource.Webhooks == nil || resource.Webhooks.IsEmpty() {
		return nil
	}
	args := append([]string{"create", "webhook"}, getGVKFlags(resource)...)
	args = append(args, getWebhookResourceFlags(resource)...)
	return util.RunCmd("kubebuilder create webhook", "kubebuilder", args...)
}

// Gets flags for webhook creation.
func getWebhookResourceFlags(resource resource.Resource) []string {
	var args []string
	if resource.IsExternal() {
		args = append(args, "--external-api-path", resource.Path)
		args = append(args, "--external-api-domain", resource.Domain)
	}
	if resource.HasValidationWebhook() {
		args = append(args, "--programmatic-validation")
	}
	if resource.HasDefaultingWebhook() {
		args = append(args, "--defaulting")
	}
	if resource.HasConversionWebhook() {
		args = append(args, "--conversion")
		if len(resource.Webhooks.Spoke) > 0 {
			for _, spoke := range resource.Webhooks.Spoke {
				args = append(args, "--spoke", spoke)
			}
		}
	}
	return args
}

// Copies files from source to destination.
func copyFile(src, des string) error {
	bytesRead, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("source file path %s does not exist: %w", src, err)
	}
	return os.WriteFile(des, bytesRead, 0o755)
}

// Migrates Grafana configuration files.
func grafanaConfigMigrate(src, des string) error {
	grafanaConfig := fmt.Sprintf("%s/grafana/custom-metrics/config.yaml", src)
	if _, err := os.Stat(grafanaConfig); os.IsNotExist(err) {
		return fmt.Errorf("Grafana config path %s does not exist: %w", grafanaConfig, err)
	}
	return copyFile(grafanaConfig, fmt.Sprintf("%s/grafana/custom-metrics/config.yaml", des))
}

// Edits the project to include the Grafana plugin.
func kubebuilderGrafanaEdit() error {
	args := []string{"edit", "--plugins", plugin.KeyFor(v1alpha.Plugin{})}
	if err := util.RunCmd("kubebuilder edit", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run edit subcommand for Grafana plugin: %w", err)
	}
	return nil
}

// Edits the project to include the Helm plugin.
func kubebuilderHelmEdit() error {
	args := []string{"edit", "--plugins", plugin.KeyFor(hemlv1alpha.Plugin{})}
	if err := util.RunCmd("kubebuilder edit", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run edit subcommand for Helm plugin: %w", err)
	}
	return nil
}

// hasHelmPlugin checks if the Helm plugin is present by inspecting the plugin chain or configuration.
func hasHelmPlugin(cfg store.Store) bool {
	var pluginConfig map[string]interface{}

	// Decode the Helm plugin configuration to check if it's present
	err := cfg.Config().DecodePluginConfig(plugin.KeyFor(hemlv1alpha.Plugin{}), &pluginConfig)
	if err != nil {
		// If the Helm plugin is not found, return false
		if errors.As(err, &config.PluginKeyNotFoundError{}) {
			return false
		}
		// Log other errors if needed
		log.Errorf("Error decoding Helm plugin config: %v", err)
		return false
	}

	// Helm plugin is present
	return true
}
