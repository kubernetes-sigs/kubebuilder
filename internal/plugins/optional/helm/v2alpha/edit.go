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

package v2alpha

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"go.yaml.in/yaml/v3"

	"sigs.k8s.io/kubebuilder/v4/internal/plugins/optional/helm/v2alpha/scaffolds"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
)

const (
	// DefaultManifestsFile is the default path for kustomize output manifests
	DefaultManifestsFile = "dist/install.yaml"
	// DefaultOutputDir is the default output directory for Helm charts
	DefaultOutputDir = "dist"
	// v1AlphaPluginKey is the deprecated v1-alpha plugin key
	v1AlphaPluginKey = "helm.kubebuilder.io/v1-alpha"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config        config.Config
	force         bool
	manifestsFile string
	outputDir     string
}

//nolint:lll
func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Generate a Helm chart from your project's kustomize output.

Parses 'make build-installer' output (dist/install.yaml) and generates chart to allow easy
distribution of your project. When enabled, adds Helm helpers targets to Makefile`

	subcmdMeta.Examples = fmt.Sprintf(`# Generate Helm chart from default manifests (dist/install.yaml) to default output (dist/)
  %[1]s edit --plugins=%[2]s

# Generate Helm chart and overwrite existing files (useful for updates)
  %[1]s edit --plugins=%[2]s --force

# Generate Helm chart from a custom manifests file
  %[1]s edit --plugins=%[2]s --manifests=path/to/custom-install.yaml

# Generate Helm chart to a custom output directory
  %[1]s edit --plugins=%[2]s --output-dir=charts

# Generate from custom manifests to custom output directory
  %[1]s edit --plugins=%[2]s --manifests=manifests/install.yaml --output-dir=helm-charts

# Typical workflow:
  make build-installer  # Generate dist/install.yaml with latest changes
  %[1]s edit --plugins=%[2]s  # Generate/update Helm chart in dist/chart/

**NOTE**: Chart.yaml is never overwritten (contains user-managed version info).
Without --force, the plugin also preserves values.yaml, NOTES.txt, _helpers.tpl, .helmignore,
and .github/workflows/test-chart.yml.
All other template files in templates/ are always regenerated to match your current
kustomize output. Use --force to regenerate all files except Chart.yaml.

The generated chart structure mirrors your config/ directory:
<output>/chart/
├── Chart.yaml
├── values.yaml
├── .helmignore
└── templates/
    ├── NOTES.txt
    ├── _helpers.tpl
    ├── rbac/
    ├── manager/
    ├── webhook/
    └── ...
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.force, "force", false, "if true, regenerates all the files")
	fs.StringVar(&p.manifestsFile, "manifests", DefaultManifestsFile,
		"path to the YAML file containing Kubernetes manifests from kustomize output")
	fs.StringVar(&p.outputDir, "output-dir", DefaultOutputDir, "output directory for the generated Helm chart")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	// If using default manifests file, ensure it exists by running make build-installer
	if p.manifestsFile == DefaultManifestsFile {
		if err := p.ensureManifestsExist(); err != nil {
			slog.Warn("Failed to generate default manifests file", "error", err, "file", p.manifestsFile)
		}
	}

	scaffolder := scaffolds.NewKustomizeHelmScaffolder(p.config, p.force, p.manifestsFile, p.outputDir)
	scaffolder.InjectFS(fs)
	err := scaffolder.Scaffold()
	if err != nil {
		return fmt.Errorf("error scaffolding Helm chart: %w", err)
	}

	// Remove deprecated v1-alpha plugin entry from PROJECT file
	// This must happen in Scaffold (before config is saved) to be persisted
	p.removeV1AlphaPluginEntry()

	// Save plugin config to PROJECT file
	key := plugin.GetPluginKeyForConfig(p.config.GetPluginChain(), Plugin{})
	canonicalKey := plugin.KeyFor(Plugin{})
	cfg := pluginConfig{}
	isFirstRun := false
	if err = p.config.DecodePluginConfig(key, &cfg); err != nil {
		switch {
		case errors.As(err, &config.UnsupportedFieldError{}):
			// Config version doesn't support plugin metadata
			return nil
		case errors.As(err, &config.PluginKeyNotFoundError{}):
			// This is the first time the plugin is run
			isFirstRun = true
			if key != canonicalKey {
				if err2 := p.config.DecodePluginConfig(canonicalKey, &cfg); err2 != nil {
					if errors.As(err2, &config.UnsupportedFieldError{}) {
						return nil
					}
					if !errors.As(err2, &config.PluginKeyNotFoundError{}) {
						return fmt.Errorf("error decoding plugin configuration: %w", err2)
					}
				} else {
					// Found config under canonical key, not first run
					isFirstRun = false
				}
			}
		default:
			return fmt.Errorf("error decoding plugin configuration: %w", err)
		}
	}

	// Update configuration with current parameters
	cfg.ManifestsFile = p.manifestsFile
	cfg.OutputDir = p.outputDir

	if err = p.config.EncodePluginConfig(key, cfg); err != nil {
		return fmt.Errorf("error encoding plugin configuration: %w", err)
	}

	// Add Helm deployment targets to Makefile only on first run
	if isFirstRun {
		slog.Info("adding Helm deployment targets to Makefile...")
		// Extract namespace from manifests for accurate Makefile generation
		namespace := p.extractNamespaceFromManifests()
		if err := p.addHelmMakefileTargets(namespace); err != nil {
			slog.Warn("failed to add Helm targets to Makefile", "error", err)
		}
	}

	return nil
}

// ensureManifestsExist runs make build-installer to generate the default manifests file
func (p *editSubcommand) ensureManifestsExist() error {
	slog.Info("Generating default manifests file", "file", p.manifestsFile)

	// Run the required make targets to generate the manifests file
	targets := []string{"manifests", "generate", "build-installer"}
	for _, target := range targets {
		if err := util.RunCmd(fmt.Sprintf("Running make %s", target), "make", target); err != nil {
			return fmt.Errorf("make %s failed: %w", target, err)
		}
	}

	// Verify the file was created
	if _, err := os.Stat(p.manifestsFile); err != nil {
		return fmt.Errorf("manifests file %s was not created: %w", p.manifestsFile, err)
	}

	slog.Info("Successfully generated manifests file", "file", p.manifestsFile)
	return nil
}

// PostScaffold automatically uncomments cert-manager installation when webhooks are present
func (p *editSubcommand) PostScaffold() error {
	hasWebhooks := hasWebhooksWith(p.config)

	if hasWebhooks {
		workflowFile := filepath.Join(".github", "workflows", "test-chart.yml")
		if _, err := os.Stat(workflowFile); err != nil {
			slog.Info(
				"Workflow file not found, unable to uncomment cert-manager installation",
				"error", err,
				"file", workflowFile,
			)
			return nil
		}
		target := `
#      - name: Install cert-manager via Helm (wait for readiness)
#        run: |
#          helm repo add jetstack https://charts.jetstack.io
#          helm repo update
#          helm install cert-manager jetstack/cert-manager \
#            --namespace cert-manager \
#            --create-namespace \
#            --set crds.enabled=true \
#            --wait \
#            --timeout 300s`
		if err := util.UncommentCode(workflowFile, target, "#"); err != nil {
			hasUncommented, errCheck := util.HasFileContentWith(workflowFile, "- name: Install cert-manager via Helm")
			if !hasUncommented || errCheck != nil {
				slog.Warn("Failed to uncomment cert-manager installation in workflow file", "error", err, "file", workflowFile)
			}
		} else {
			target = `# TODO: Uncomment if cert-manager is enabled`
			_ = util.ReplaceInFile(workflowFile, target, "")
		}
	}
	return nil
}

// addHelmMakefileTargets appends Helm deployment targets to the Makefile if they don't already exist
func (p *editSubcommand) addHelmMakefileTargets(namespace string) error {
	makefilePath := "Makefile"
	if _, err := os.Stat(makefilePath); os.IsNotExist(err) {
		return fmt.Errorf("makefile not found")
	}

	// Get the Helm Makefile targets
	helmTargets := getHelmMakefileTargets(p.config.GetProjectName(), namespace, p.outputDir)

	// Append the targets if they don't already exist
	if err := util.AppendCodeIfNotExist(makefilePath, helmTargets); err != nil {
		return fmt.Errorf("failed to append Helm targets to Makefile: %w", err)
	}

	slog.Info("added Helm deployment targets to Makefile",
		"targets", "helm-deploy, helm-uninstall, helm-status, helm-history, helm-rollback")
	return nil
}

// extractNamespaceFromManifests parses the manifests file to extract the manager namespace.
// Returns projectName-system if manifests don't exist or namespace not found.
func (p *editSubcommand) extractNamespaceFromManifests() string {
	// Default to project-name-system pattern
	defaultNamespace := p.config.GetProjectName() + "-system"

	// If manifests file doesn't exist, use default
	if _, err := os.Stat(p.manifestsFile); os.IsNotExist(err) {
		return defaultNamespace
	}

	// Parse the manifests to get the namespace
	file, err := os.Open(p.manifestsFile)
	if err != nil {
		return defaultNamespace
	}
	defer func() {
		_ = file.Close()
	}()

	// Parse YAML documents looking for the manager Deployment
	decoder := yaml.NewDecoder(file)
	for {
		var doc map[string]any
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		// Check if this is a Deployment (manager)
		if kind, ok := doc["kind"].(string); ok && kind == "Deployment" {
			if metadata, ok := doc["metadata"].(map[string]any); ok {
				// Check if it's the manager deployment
				if name, ok := metadata["name"].(string); ok && strings.Contains(name, "controller-manager") {
					// Extract namespace from the manager Deployment
					if namespace, ok := metadata["namespace"].(string); ok && namespace != "" {
						return namespace
					}
				}
			}
		}
	}

	// Fallback to default if manager Deployment not found
	return defaultNamespace
}

// getHelmMakefileTargets returns the Helm Makefile targets as a string
// following the same patterns as the existing Makefile deployment section
func getHelmMakefileTargets(projectName, namespace, outputDir string) string {
	if outputDir == "" {
		outputDir = "dist"
	}

	// Use the project name as default for release name
	release := projectName

	return helmMakefileTemplate(namespace, release, outputDir)
}

// helmMakefileTemplate returns the Helm deployment section template
// This follows the same pattern as the Kustomize deployment section in the Go plugin
const helmMakefileTemplateFormat = `
##@ Helm Deployment

## Helm binary to use for deploying the chart
HELM ?= helm
## Namespace to deploy the Helm release
HELM_NAMESPACE ?= %s
## Name of the Helm release
HELM_RELEASE ?= %s
## Path to the Helm chart directory
HELM_CHART_DIR ?= %s/chart
## Additional arguments to pass to helm commands
HELM_EXTRA_ARGS ?=

.PHONY: install-helm
install-helm: ## Install the latest version of Helm.
	@command -v $(HELM) >/dev/null 2>&1 || { \
		echo "Installing Helm..." && \
		curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-4 | bash; \
	}

.PHONY: helm-deploy
helm-deploy: install-helm ## Deploy manager to the K8s cluster via Helm. Specify an image with IMG.
	$(HELM) upgrade --install $(HELM_RELEASE) $(HELM_CHART_DIR) \
		--namespace $(HELM_NAMESPACE) \
		--create-namespace \
		--set manager.image.repository=$${IMG%%:*} \
		--set manager.image.tag=$${IMG##*:} \
		--wait \
		--timeout 5m \
		$(HELM_EXTRA_ARGS)

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall the Helm release from the K8s cluster.
	$(HELM) uninstall $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)

.PHONY: helm-status
helm-status: ## Show Helm release status.
	$(HELM) status $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)

.PHONY: helm-history
helm-history: ## Show Helm release history.
	$(HELM) history $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)

.PHONY: helm-rollback
helm-rollback: ## Rollback to previous Helm release.
	$(HELM) rollback $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)
`

func helmMakefileTemplate(namespace, release, outputDir string) string {
	return fmt.Sprintf(helmMakefileTemplateFormat, namespace, release, outputDir)
}

func hasWebhooksWith(c config.Config) bool {
	resources, err := c.GetResources()
	if err != nil {
		return false
	}

	for _, res := range resources {
		if res.HasDefaultingWebhook() || res.HasValidationWebhook() || res.HasConversionWebhook() {
			return true
		}
	}

	return false
}

// removeV1AlphaPluginEntry removes the deprecated helm.kubebuilder.io/v1-alpha plugin entry.
// This must be called from Scaffold (before config is saved) for changes to be persisted.
func (p *editSubcommand) removeV1AlphaPluginEntry() {
	// Only attempt to remove if using v3 config (which supports plugin configs)
	cfg, ok := p.config.(*cfgv3.Cfg)
	if !ok {
		return
	}

	// Check if v1-alpha plugin entry exists
	if cfg.Plugins == nil {
		return
	}

	if _, exists := cfg.Plugins[v1AlphaPluginKey]; exists {
		delete(cfg.Plugins, v1AlphaPluginKey)
		slog.Info("removed deprecated v1-alpha plugin entry")
	}
}
