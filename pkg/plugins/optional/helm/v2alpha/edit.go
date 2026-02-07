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
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds"
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
	delete        bool // Delete flag to remove Helm chart generation
}

//nolint:lll
func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Generate a Helm chart from your project's kustomize output.

This plugin dynamically generates Helm chart templates by parsing the output of 'make build-installer' 
(dist/install.yaml by default). The generated chart preserves all customizations made to your kustomize 
configuration including environment variables, labels, and annotations.

The chart structure mirrors your config/ directory organization for easy maintenance.`

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
Without --force, the plugin also preserves values.yaml, NOTES.txt, _helpers.tpl, .helmignore, and
.github/workflows/test-chart.yml. All template files in templates/ are always regenerated
to match your current kustomize output. Use --force to regenerate all files except Chart.yaml.

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
	fs.BoolVar(&p.delete, "delete", false, "delete Helm chart generation from the project")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	// Handle delete mode
	if p.delete {
		return p.deleteHelmChart(fs)
	}

	// Normal scaffold mode
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
	if err = p.config.DecodePluginConfig(key, &cfg); err != nil {
		switch {
		case errors.As(err, &config.UnsupportedFieldError{}):
			// Config version doesn't support plugin metadata
			return nil
		case errors.As(err, &config.PluginKeyNotFoundError{}):
			if key != canonicalKey {
				if err2 := p.config.DecodePluginConfig(canonicalKey, &cfg); err2 != nil {
					if errors.As(err2, &config.UnsupportedFieldError{}) {
						return nil
					}
					if !errors.As(err2, &config.PluginKeyNotFoundError{}) {
						return fmt.Errorf("error decoding plugin configuration: %w", err2)
					}
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

// deleteHelmChart removes Helm chart files and configuration (best effort)
func (p *editSubcommand) deleteHelmChart(fs machinery.Filesystem) error {
	slog.Info("Deleting Helm chart files...")

	// Get plugin config to find output directory
	key := plugin.GetPluginKeyForConfig(p.config.GetPluginChain(), Plugin{})
	canonicalKey := plugin.KeyFor(Plugin{})
	cfg := pluginConfig{}

	err := p.config.DecodePluginConfig(key, &cfg)
	if err != nil {
		if errors.As(err, &config.PluginKeyNotFoundError{}) && key != canonicalKey {
			_ = p.config.DecodePluginConfig(canonicalKey, &cfg)
		}
	}

	// Use configured output dir or default
	outputDir := p.outputDir
	if outputDir == "" {
		outputDir = cfg.OutputDir
	}
	if outputDir == "" {
		outputDir = DefaultOutputDir
	}

	deletedCount := 0
	warnCount := 0

	// Delete chart directory (best effort)
	chartDir := filepath.Join(outputDir, "chart")
	if exists, _ := afero.DirExists(fs.FS, chartDir); exists {
		if err := fs.FS.RemoveAll(chartDir); err != nil {
			slog.Warn("Failed to delete Helm chart directory", "path", chartDir, "error", err)
			warnCount++
		} else {
			slog.Info("Deleted Helm chart directory", "path", chartDir)
			deletedCount++
		}
	} else {
		slog.Warn("Helm chart directory not found", "path", chartDir)
		warnCount++
	}

	// Delete test workflow (best effort)
	testChartPath := filepath.Join(".github", "workflows", "test-chart.yml")
	if exists, _ := afero.Exists(fs.FS, testChartPath); exists {
		if err := fs.FS.Remove(testChartPath); err != nil {
			slog.Warn("Failed to delete test-chart.yml", "path", testChartPath, "error", err)
			warnCount++
		} else {
			slog.Info("Deleted test-chart workflow", "path", testChartPath)
			deletedCount++
		}
	} else {
		slog.Warn("Test chart workflow not found", "path", testChartPath)
		warnCount++
	}

	// Remove plugin config from PROJECT by encoding empty struct
	if encErr := p.config.EncodePluginConfig(key, struct{}{}); encErr != nil {
		// Try canonical key if different
		if key != canonicalKey {
			if encErr2 := p.config.EncodePluginConfig(canonicalKey, struct{}{}); encErr2 != nil {
				slog.Warn("Failed to remove plugin configuration from PROJECT file",
					"provided_key_error", encErr, "canonical_key_error", encErr2)
				warnCount++
			}
		} else {
			slog.Warn("Failed to remove plugin configuration from PROJECT file", "error", encErr)
			warnCount++
		}
	}

	fmt.Printf("\nSuccessfully completed Helm plugin deletion\n")
	if deletedCount > 0 {
		fmt.Printf("Deleted: %d item(s)\n", deletedCount)
	}
	if warnCount > 0 {
		fmt.Printf("Warnings: %d item(s) - some files may not exist or couldn't be deleted (see logs)\n", warnCount)
	}

	return nil
}
