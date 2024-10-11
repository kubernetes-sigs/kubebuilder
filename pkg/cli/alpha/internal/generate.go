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
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1"
)

type Generate struct {
	InputDir  string
	OutputDir string
}

const (
	defaultOutputDir     = "output-dir"
	grafanaPluginKey     = "grafana.kubebuilder.io/v1-alpha"
	deployImagePluginKey = "deploy-image.go.kubebuilder.io/v1-alpha"
)

// Generate handles the migration and scaffolding process.
func (opts *Generate) Generate() error {
	config, err := loadProjectConfig(opts.InputDir)
	if err != nil {
		return err
	}

	if err := createDirectory(opts.OutputDir); err != nil {
		return err
	}

	if err := changeWorkingDirectory(opts.OutputDir); err != nil {
		return err
	}

	if err := kubebuilderInit(config); err != nil {
		return err
	}

	if err := kubebuilderEdit(config); err != nil {
		return err
	}

	if err := kubebuilderCreate(config); err != nil {
		return err
	}

	if err := migrateGrafanaPlugin(config, opts.InputDir, opts.OutputDir); err != nil {
		return err
	}

	if err := migrateDeployImagePlugin(config); err != nil {
		return err
	}

	return nil
}

// Validate ensures the options are valid and kubebuilder is installed.
func (opts *Generate) Validate() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	opts.InputDir, err = getInputPath(cwd, opts.InputDir)
	if err != nil {
		return err
	}

	opts.OutputDir, err = getOutputPath(cwd, opts.OutputDir)
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
	config := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := config.LoadFrom(fmt.Sprintf("%s/%s", inputDir, yaml.DefaultPath)); err != nil {
		return nil, fmt.Errorf("failed to load PROJECT file: %w", err)
	}
	return config, nil
}

// Helper function to create the output directory.
func createDirectory(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
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

	for _, r := range resources {
		if err := createAPI(r); err != nil {
			return fmt.Errorf("failed to create API: %w", err)
		}
		if err := createWebhook(r); err != nil {
			return fmt.Errorf("failed to create webhook: %w", err)
		}
	}

	return nil
}

// Migrates the Grafana plugin.
func migrateGrafanaPlugin(store store.Store, src, des string) error {
	var grafanaPlugin struct{}
	err := store.Config().DecodePluginConfig(grafanaPluginKey, grafanaPlugin)
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
	err := store.Config().DecodePluginConfig(deployImagePluginKey, &deployImagePlugin)
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
func getInputPath(currentWorkingDirectory, inputPath string) (string, error) {
	if inputPath == "" {
		inputPath = currentWorkingDirectory
	}
	projectPath := fmt.Sprintf("%s/%s", inputPath, yaml.DefaultPath)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return "", fmt.Errorf("project path %s does not exist: %w", projectPath, err)
	}
	return inputPath, nil
}

// Helper function to get output path.
func getOutputPath(currentWorkingDirectory, outputPath string) (string, error) {
	if outputPath == "" {
		outputPath = fmt.Sprintf("%s/%s", currentWorkingDirectory, defaultOutputDir)
	}
	if _, err := os.Stat(outputPath); err == nil {
		return "", fmt.Errorf("output path %s already exists", outputPath)
	}
	return outputPath, nil
}

// Helper function to get Init arguments for Kubebuilder.
func getInitArgs(store store.Store) []string {
	var args []string
	plugins := store.Config().GetPluginChain()
	if len(plugins) > 0 {
		args = append(args, "--plugins", strings.Join(plugins, ","))
	}
	if domain := store.Config().GetDomain(); domain != "" {
		args = append(args, "--domain", domain)
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
	args = append(args, fmt.Sprintf("--plugins=%s", "deploy-image/v1-alpha"))
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
	if resource.HasConversionWebhook() {
		args = append(args, "--conversion")
	}
	if resource.HasValidationWebhook() {
		args = append(args, "--programmatic-validation")
	}
	if resource.HasDefaultingWebhook() {
		args = append(args, "--defaulting")
	}
	return args
}

// Copies files from source to destination.
func copyFile(src, des string) error {
	bytesRead, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("source file path %s does not exist: %w", src, err)
	}
	return os.WriteFile(des, bytesRead, 0755)
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
	args := []string{"edit", "--plugins", grafanaPluginKey}
	if err := util.RunCmd("kubebuilder edit", "kubebuilder", args...); err != nil {
		return fmt.Errorf("failed to run edit subcommand for Grafana plugin: %w", err)
	}
	return nil
}
