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

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
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

**NOTE**: The plugin preserves customizations in values.yaml, Chart.yaml, _helpers.tpl, and .helmignore
unless --force is used. All template files are regenerated to match your current kustomize output.

The generated chart structure mirrors your config/ directory:
<output>/chart/
├── Chart.yaml
├── values.yaml
├── .helmignore
└── templates/
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
