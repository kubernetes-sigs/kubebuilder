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
		cwd, getWdErr := os.Getwd()
		if getWdErr != nil {
			return fmt.Errorf("failed to get working directory: %w", getWdErr)
		}
		opts.OutputDir = cwd
		if _, err = os.Stat(opts.OutputDir); err == nil {
			log.Warn("Using current working directory to re-scaffold the project")
			log.Warn("This directory will be cleaned up and all files removed before the re-generation")

			// Ensure we clean the correct directory
			log.Info("Cleaning directory:", opts.OutputDir)

			// Use an absolute path to target files directly
			cleanupCmd := fmt.Sprintf("rm -rf %s/*", opts.OutputDir)
			err = util.RunCmd("Running cleanup", "sh", "-c", cleanupCmd)
			if err != nil {
				log.Error("Cleanup failed:", err)
				return fmt.Errorf("cleanup failed: %w", err)
			}
		}
	}

	if err = createDirectory(opts.OutputDir); err != nil {
		return fmt.Errorf("error creating output directory %q: %w", opts.OutputDir, err)
	}

	if err = changeWorkingDirectory(opts.OutputDir); err != nil {
		return fmt.Errorf("error changing working directory %q: %w", opts.OutputDir, err)
	}

	if err = kubebuilderInit(projectConfig); err != nil {
		return fmt.Errorf("error initializing project config: %w", err)
	}

	if err = kubebuilderEdit(projectConfig); err != nil {
		return fmt.Errorf("error editing project config: %w", err)
	}

	if err = kubebuilderCreate(projectConfig); err != nil {
		return fmt.Errorf("error creating project config: %w", err)
	}

	if err = migrateGrafanaPlugin(projectConfig, opts.InputDir, opts.OutputDir); err != nil {
		return fmt.Errorf("error migrating Grafana plugin: %w", err)
	}

	if hasHelmPlugin(projectConfig) {
		if err = kubebuilderHelmEdit(); err != nil {
			return fmt.Errorf("error editing Helm plugin: %w", err)
		}
	}

	return migrateDeployImagePlugin(projectConfig)
}

// Validate ensures the options are valid and kubebuilder is installed.
func (opts *Generate) Validate() error {
	var err error
	opts.InputDir, err = getInputPath(opts.InputDir)
	if err != nil {
		return fmt.Errorf("error getting input path %q: %w", opts.InputDir, err)
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
		return fmt.Errorf("failed to create output directory %q: %w", outputDir, err)
	}
	return nil
}

// Helper function to change the current working directory.
func changeWorkingDirectory(outputDir string) error {
	if err := os.Chdir(outputDir); err != nil {
		return fmt.Errorf("failed to change the working directory to %q: %w", outputDir, err)
	}
	return nil
}

// Initializes the project with Kubebuilder.
func kubebuilderInit(s store.Store) error {
	args := append([]string{"init"}, getInitArgs(s)...)
	if err := util.RunCmd("kubebuilder init", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run kubebuilder init command: %w", err)
	}

	return nil
}

// Edits the project to enable or disable multigroup layout.
func kubebuilderEdit(s store.Store) error {
	if s.Config().IsMultiGroup() {
		args := []string{"edit", "--multigroup"}
		if err := util.RunCmd("kubebuilder edit", "kubebuilder", args...); err != nil {
			return fmt.Errorf("failed to run kubebuilder edit command: %w", err)
		}
	}

	return nil
}

// Creates APIs and Webhooks for the project.
func kubebuilderCreate(s store.Store) error {
	resources, err := s.Config().GetResources()
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	// First, scaffold all APIs
	for _, r := range resources {
		if err = createAPI(r); err != nil {
			return fmt.Errorf("failed to create API for %s/%s/%s: %w", r.Group, r.Version, r.Kind, err)
		}
	}

	// Then, scaffold all webhooks
	// We cannot create a webhook for an API that does not exist
	for _, r := range resources {
		if err = createWebhook(r); err != nil {
			return fmt.Errorf("failed to create webhook for %s/%s/%s: %w", r.Group, r.Version, r.Kind, err)
		}
	}

	return nil
}

// Migrates the Grafana plugin.
func migrateGrafanaPlugin(s store.Store, src, des string) error {
	var grafanaPlugin struct{}
	err := s.Config().DecodePluginConfig(plugin.KeyFor(v1alpha.Plugin{}), grafanaPlugin)
	if errors.As(err, &config.PluginKeyNotFoundError{}) {
		log.Info("Grafana plugin not found, skipping migration")
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to decode grafana plugin config: %w", err)
	}

	if err = kubebuilderGrafanaEdit(); err != nil {
		return fmt.Errorf("error editing Grafana plugin: %w", err)
	}

	if err = grafanaConfigMigrate(src, des); err != nil {
		return fmt.Errorf("error migrating Grafana config: %w", err)
	}

	return kubebuilderGrafanaEdit()
}

// Migrates the Deploy Image plugin.
func migrateDeployImagePlugin(s store.Store) error {
	var deployImagePlugin v1alpha1.PluginConfig
	err := s.Config().DecodePluginConfig(plugin.KeyFor(v1alpha1.Plugin{}), &deployImagePlugin)
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
func createAPIWithDeployImage(resourceData v1alpha1.ResourceData) error {
	args := append([]string{"create", "api"}, getGVKFlagsFromDeployImage(resourceData)...)
	args = append(args, getDeployImageOptions(resourceData)...)
	if err := util.RunCmd("kubebuilder create api", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run kubebuilder create api command: %w", err)
	}

	return nil
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
		return "", fmt.Errorf("project path %q does not exist: %w", projectPath, err)
	}
	return inputPath, nil
}

// Helper function to get Init arguments for Kubebuilder.
func getInitArgs(s store.Store) []string {
	var args []string
	plugins := s.Config().GetPluginChain()

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
	if domain := s.Config().GetDomain(); domain != "" {
		args = append(args, "--domain", domain)
	}
	if repo := s.Config().GetRepository(); repo != "" {
		args = append(args, "--repo", repo)
	}
	return args
}

// Gets the GVK flags for a resource.
func getGVKFlags(res resource.Resource) []string {
	var args []string
	if res.Plural != "" {
		args = append(args, "--plural", res.Plural)
	}
	if res.Group != "" {
		args = append(args, "--group", res.Group)
	}
	if res.Version != "" {
		args = append(args, "--version", res.Version)
	}
	if res.Kind != "" {
		args = append(args, "--kind", res.Kind)
	}
	return args
}

// Gets the GVK flags for a Deploy Image resource.
func getGVKFlagsFromDeployImage(resourceData v1alpha1.ResourceData) []string {
	var args []string
	if resourceData.Group != "" {
		args = append(args, "--group", resourceData.Group)
	}
	if resourceData.Version != "" {
		args = append(args, "--version", resourceData.Version)
	}
	if resourceData.Kind != "" {
		args = append(args, "--kind", resourceData.Kind)
	}
	return args
}

// Gets the options for a Deploy Image resource.
func getDeployImageOptions(resourceData v1alpha1.ResourceData) []string {
	var args []string
	if resourceData.Options.Image != "" {
		args = append(args, fmt.Sprintf("--image=%s", resourceData.Options.Image))
	}
	if resourceData.Options.ContainerCommand != "" {
		args = append(args, fmt.Sprintf("--image-container-command=%s", resourceData.Options.ContainerCommand))
	}
	if resourceData.Options.ContainerPort != "" {
		args = append(args, fmt.Sprintf("--image-container-port=%s", resourceData.Options.ContainerPort))
	}
	if resourceData.Options.RunAsUser != "" {
		args = append(args, fmt.Sprintf("--run-as-user=%s", resourceData.Options.RunAsUser))
	}
	args = append(args, fmt.Sprintf("--plugins=%s", plugin.KeyFor(v1alpha1.Plugin{})))
	return args
}

// Creates an API resource.
func createAPI(res resource.Resource) error {
	args := append([]string{"create", "api"}, getGVKFlags(res)...)
	args = append(args, getAPIResourceFlags(res)...)

	// Add the external API path flag if the resource is external
	if res.IsExternal() {
		args = append(args, "--external-api-path", res.Path)
		args = append(args, "--external-api-domain", res.Domain)
	}

	if err := util.RunCmd("kubebuilder create api", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run kubebuilder create api command: %w", err)
	}

	return nil
}

// Gets flags for API resource creation.
func getAPIResourceFlags(res resource.Resource) []string {
	var args []string

	if res.API == nil || res.API.IsEmpty() {
		args = append(args, "--resource=false")
	} else {
		args = append(args, "--resource")
		if res.API.Namespaced {
			args = append(args, "--namespaced")
		} else {
			args = append(args, "--namespaced=false")
		}
	}
	if res.Controller {
		args = append(args, "--controller")
	} else {
		args = append(args, "--controller=false")
	}
	return args
}

// Creates a webhook resource.
func createWebhook(res resource.Resource) error {
	if res.Webhooks == nil || res.Webhooks.IsEmpty() {
		return nil
	}
	args := append([]string{"create", "webhook"}, getGVKFlags(res)...)
	args = append(args, getWebhookResourceFlags(res)...)

	if err := util.RunCmd("kubebuilder create webhook", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run kubebuilder create webhook command: %w", err)
	}

	return nil
}

// Gets flags for webhook creation.
func getWebhookResourceFlags(res resource.Resource) []string {
	var args []string
	if res.IsExternal() {
		args = append(args, "--external-api-path", res.Path)
		args = append(args, "--external-api-domain", res.Domain)
	}
	if res.HasValidationWebhook() {
		args = append(args, "--programmatic-validation")
	}
	if res.HasDefaultingWebhook() {
		args = append(args, "--defaulting")
	}
	if res.HasConversionWebhook() {
		args = append(args, "--conversion")
		if len(res.Webhooks.Spoke) > 0 {
			for _, spoke := range res.Webhooks.Spoke {
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
		return fmt.Errorf("source file path %q does not exist: %w", src, err)
	}
	if err = os.WriteFile(des, bytesRead, 0o755); err != nil {
		return fmt.Errorf("failed to write file %q: %w", des, err)
	}

	return nil
}

// Migrates Grafana configuration files.
func grafanaConfigMigrate(src, des string) error {
	grafanaConfig := fmt.Sprintf("%s/grafana/custom-metrics/config.yaml", src)
	if _, err := os.Stat(grafanaConfig); os.IsNotExist(err) {
		return fmt.Errorf("grafana config path %s does not exist: %w", grafanaConfig, err)
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
		log.Errorf("error decoding Helm plugin config: %v", err)
		return false
	}

	// Helm plugin is present
	return true
}
