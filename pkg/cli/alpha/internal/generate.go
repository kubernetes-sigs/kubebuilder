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
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal/common"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	deployimagev1alpha1 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1"
	autoupdatev1alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/autoupdate/v1alpha"
	grafanav1alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/grafana/v1alpha"
	helmv1alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha"
	helmv2alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha"
)

// Generate store the required info for the command
type Generate struct {
	InputDir  string
	OutputDir string
}

// Generate handles the migration and scaffolding process.
func (opts *Generate) Generate() error {
	projectConfig, err := common.LoadProjectConfig(opts.InputDir)
	if err != nil {
		return fmt.Errorf("error loading project config: %v", err)
	}

	if opts.OutputDir == "" {
		cwd, getWdErr := os.Getwd()
		if getWdErr != nil {
			return fmt.Errorf("failed to get working directory: %w", getWdErr)
		}
		opts.OutputDir = cwd
		if _, err = os.Stat(opts.OutputDir); err == nil {
			slog.Warn("Using current working directory to re-scaffold the project")
			slog.Warn("This directory will be cleaned up and all files removed before the re-generation")

			// Ensure we clean the correct directory
			slog.Info("Cleaning directory", "dir", opts.OutputDir)

			// Use an absolute path to target files directly
			cleanupCmd := fmt.Sprintf("rm -rf %s/*", opts.OutputDir)
			err = util.RunCmd("Running cleanup", "sh", "-c", cleanupCmd)
			if err != nil {
				slog.Error("Cleanup failed", "error", err)
				return fmt.Errorf("cleanup failed: %w", err)
			}

			// Note that we should remove ALL files except the PROJECT file and .git directory
			cleanupCmd = fmt.Sprintf(
				`find %q -mindepth 1 -maxdepth 1 ! -name '.git' ! -name 'PROJECT' -exec rm -rf {} +`,
				opts.OutputDir,
			)
			err = util.RunCmd("Running cleanup", "sh", "-c", cleanupCmd)
			if err != nil {
				slog.Error("Cleanup failed", "error", err)
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

	if err = migrateAutoUpdatePlugin(projectConfig); err != nil {
		return fmt.Errorf("error migrating AutoUpdate plugin: %w", err)
	}

	if hasHelm, isV2Alpha := hasHelmPlugin(projectConfig); hasHelm && isV2Alpha {
		if err = kubebuilderHelmEditWithConfig(projectConfig); err != nil {
			return fmt.Errorf("error editing Helm plugin: %w", err)
		}
	}

	if err = migrateDeployImagePlugin(projectConfig); err != nil {
		return fmt.Errorf("error migrating deploy-image plugin: %w", err)
	}

	// Run make targets to ensure the project is properly set up.
	// These steps are performed on a best-effort basis: if any of the targets fail,
	// we slog a warning to inform the user, but we do not stop the process or return an error.
	// This is to avoid blocking the migration flow due to non-critical issues during setup.
	targets := []string{"fmt", "vet", "lint-fix"}
	for _, target := range targets {
		err := util.RunCmd(fmt.Sprintf("Running make %s", target), "make", target)
		if err != nil {
			slog.Warn("make target failed", "target", target, "error", err)
		}
	}

	return nil
}

// Validate ensures the options are valid and kubebuilder is installed.
func (opts *Generate) Validate() error {
	var err error
	opts.InputDir, err = common.GetInputPath(opts.InputDir)
	if err != nil {
		return fmt.Errorf("error getting input path %q: %w", opts.InputDir, err)
	}

	_, err = getExecutablePath()
	if err != nil {
		return err
	}

	return nil
}

// Helper function to get the PATH of binary.
func getExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("kubebuilder executable not found: %w", err)
	}

	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		slog.Warn("Unable to resolve symbolic link", "execPath", execPath, "error", err)
		// Fallback to execPath
		return execPath, nil
	}

	return realPath, nil
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
	execPath, err := getExecutablePath()
	if err != nil {
		return err
	}
	if err := util.RunCmd("kubebuilder init", execPath, args...); err != nil {
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
	err := s.Config().DecodePluginConfig(plugin.KeyFor(grafanav1alpha.Plugin{}), grafanaPlugin)
	if errors.As(err, &config.PluginKeyNotFoundError{}) {
		slog.Info("Grafana plugin not found, skipping migration")
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

func migrateAutoUpdatePlugin(s store.Store) error {
	var autoUpdatePlugin struct{}
	err := s.Config().DecodePluginConfig(plugin.KeyFor(autoupdatev1alpha.Plugin{}), autoUpdatePlugin)
	if errors.As(err, &config.PluginKeyNotFoundError{}) {
		slog.Info("Auto Update plugin not found, skipping migration")
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to decode autoupdate plugin config: %w", err)
	}

	args := []string{"edit", "--plugins", plugin.KeyFor(autoupdatev1alpha.Plugin{})}
	if err := util.RunCmd("kubebuilder edit", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run edit subcommand for Auto plugin: %w", err)
	}
	return nil
}

// Migrates the Deploy Image plugin.
func migrateDeployImagePlugin(s store.Store) error {
	var deployImagePlugin deployimagev1alpha1.PluginConfig
	err := s.Config().DecodePluginConfig(plugin.KeyFor(deployimagev1alpha1.Plugin{}), &deployImagePlugin)
	if errors.As(err, &config.PluginKeyNotFoundError{}) {
		slog.Info("Deploy-image plugin not found, skipping migration")
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
func createAPIWithDeployImage(resourceData deployimagev1alpha1.ResourceData) error {
	args := append([]string{"create", "api"}, getGVKFlagsFromDeployImage(resourceData)...)
	args = append(args, getDeployImageOptions(resourceData)...)
	if err := util.RunCmd("kubebuilder create api", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run kubebuilder create api command: %w", err)
	}

	return nil
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
			slog.Warn("We checked that your PROJECT file is configured with deprecated layout. "+
				"However, we will try our best to re-generate the project using new one",
				"deprecated_layout", plg,
				"new_layout", newPlugin)
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
func getGVKFlagsFromDeployImage(resourceData deployimagev1alpha1.ResourceData) []string {
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
func getDeployImageOptions(resourceData deployimagev1alpha1.ResourceData) []string {
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
	args = append(args, fmt.Sprintf("--plugins=%s", plugin.KeyFor(deployimagev1alpha1.Plugin{})))
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
		slog.Info("Grafana config file not found, skipping file migration", "path", grafanaConfig)
		return nil // Don't fail if config files don't exist
	}
	return copyFile(grafanaConfig, fmt.Sprintf("%s/grafana/custom-metrics/config.yaml", des))
}

// Edits the project to include the Grafana plugin.
func kubebuilderGrafanaEdit() error {
	args := []string{"edit", "--plugins", plugin.KeyFor(grafanav1alpha.Plugin{})}
	if err := util.RunCmd("kubebuilder edit", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run edit subcommand for Grafana plugin: %w", err)
	}
	return nil
}

// Edits the project to include the Helm plugin with tracked configuration.
func kubebuilderHelmEditWithConfig(s store.Store) error {
	var cfg struct {
		ManifestsFile string `json:"manifests,omitempty"`
		OutputDir     string `json:"output,omitempty"`
	}
	err := s.Config().DecodePluginConfig(plugin.KeyFor(helmv2alpha.Plugin{}), &cfg)
	if errors.As(err, &config.PluginKeyNotFoundError{}) {
		// No previous configuration, use defaults
		return kubebuilderHelmEdit(true)
	} else if err != nil {
		return fmt.Errorf("failed to decode helm plugin config: %w", err)
	}

	// Use tracked configuration values
	pluginKey := plugin.KeyFor(helmv2alpha.Plugin{})
	args := []string{"edit", "--plugins", pluginKey}
	if cfg.ManifestsFile != "" {
		args = append(args, "--manifests", cfg.ManifestsFile)
	}
	if cfg.OutputDir != "" {
		args = append(args, "--output-dir", cfg.OutputDir)
	}

	if err := util.RunCmd("kubebuilder edit", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run edit subcommand for Helm plugin: %w", err)
	}
	return nil
}

// Edits the project to include the Helm plugin.
func kubebuilderHelmEdit(isV2Alpha bool) error {
	var pluginKey string
	if isV2Alpha {
		pluginKey = plugin.KeyFor(helmv2alpha.Plugin{})
	} else {
		pluginKey = plugin.KeyFor(helmv1alpha.Plugin{})
	}

	args := []string{"edit", "--plugins", pluginKey}
	if err := util.RunCmd("kubebuilder edit", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run edit subcommand for Helm plugin: %w", err)
	}
	return nil
}

// hasHelmPlugin checks if any Helm plugin (v1alpha or v2alpha) is present by inspecting
// the plugin chain or configuration.
func hasHelmPlugin(cfg store.Store) (bool, bool) {
	var pluginConfig map[string]interface{}

	// Check for v2alpha first (preferred)
	err := cfg.Config().DecodePluginConfig(plugin.KeyFor(helmv2alpha.Plugin{}), &pluginConfig)
	if err == nil {
		return true, true // has helm plugin, is v2alpha
	}

	// Check for v1alpha
	err = cfg.Config().DecodePluginConfig(plugin.KeyFor(helmv1alpha.Plugin{}), &pluginConfig)
	if err != nil {
		// If neither Helm plugin is found, return false
		if errors.As(err, &config.PluginKeyNotFoundError{}) {
			return false, false
		}
		// slog other errors if needed
		slog.Error("error decoding Helm plugin config", "error", err)
		return false, false
	}

	// v1alpha Helm plugin is present
	return true, false // has helm plugin, is not v2alpha
}
