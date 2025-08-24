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

package v1alpha

import (
	"fmt"
	log "log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config config.Config
	force  bool
}

//nolint:lll
func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Initialize or update a Helm chart to distribute the project under the dist/ directory.

**NOTE** Before running the edit command, ensure you first execute 'make manifests' to regenerate
the latest Helm chart with your most recent changes.`

	subcmdMeta.Examples = fmt.Sprintf(`# Initialize or update a Helm chart to distribute the project under the dist/ directory
  %[1]s edit --plugins=%[2]s

# Update the Helm chart under the dist/ directory and overwrite all files
  %[1]s edit --plugins=%[2]s --force

**IMPORTANT**: If the "--force" flag is not used, the following files will not be updated to preserve your customizations:
dist/chart/
├── values.yaml
└── templates/
    └── manager/
        └── manager.yaml

The following files are never updated after their initial creation:
  - chart/Chart.yaml
  - chart/templates/_helpers.tpl
  - chart/.helmignore

All other files are updated without the usage of the '--force=true' flag
when the edit option is used to ensure that the
manifests in the chart align with the latest changes.
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.force, "force", false, "if true, regenerates all the files")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewHelmScaffolder(p.config, p.force)
	scaffolder.InjectFS(fs)
	err := scaffolder.Scaffold()
	if err != nil {
		return fmt.Errorf("error scaffolding Helm chart: %w", err)
	}

	// Track the resources following a declarative approach
	return insertPluginMetaToConfig(p.config, pluginConfig{})
}

// PostScaffold automatically uncomments cert-manager installation when webhooks are present
func (p *editSubcommand) PostScaffold() error {
	hasWebhooks := hasWebhooksWith(p.config)

	if hasWebhooks {
		workflowFile := filepath.Join(".github", "workflows", "test-chart.yml")
		if _, err := os.Stat(workflowFile); err != nil {
			log.Info(
				"Workflow file not found, unable to uncomment cert-manager installation",
				"error", err,
				"file", workflowFile,
			)
			return nil
		}
		//nolint:lll
		target := `
#      - name: Install cert-manager via Helm
#        run: |
#          helm repo add jetstack https://charts.jetstack.io
#          helm repo update
#          helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set installCRDs=true
#
#      - name: Wait for cert-manager to be ready
#        run: |
#          kubectl wait --namespace cert-manager --for=condition=available --timeout=300s deployment/cert-manager
#          kubectl wait --namespace cert-manager --for=condition=available --timeout=300s deployment/cert-manager-cainjector
#          kubectl wait --namespace cert-manager --for=condition=available --timeout=300s deployment/cert-manager-webhook`
		if err := util.UncommentCode(workflowFile, target, "#"); err != nil {
			hasUncommented, errCheck := util.HasFileContentWith(workflowFile, "- name: Install cert-manager via Helm")
			if !hasUncommented || errCheck != nil {
				log.Warn("Failed to uncomment cert-manager installation in workflow file", "error", err, "file", workflowFile)
			}
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
