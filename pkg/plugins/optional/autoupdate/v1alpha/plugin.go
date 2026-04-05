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

package v1alpha

import (
	"errors"
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
)

//nolint:lll
const metaDataDescription = `This plugin scaffolds a GitHub Action that helps you keep your project aligned with the latest Kubebuilder improvements. With a tiny amount of setup, you'll receive **automatic notifications** whenever a new Kubebuilder release is available, with both a GitHub Issue and Pull Request created by default.

Under the hood, the workflow runs 'kubebuilder alpha update' using a **3-way merge strategy** to refresh your scaffold while preserving your code. It creates and pushes an update branch, then opens both a GitHub **Issue** (for notification) and a **Pull Request** (for review and merge).

### How to set it up

1) **Add the plugin**: Use the Kubebuilder CLI to scaffold the automation into your repo.
2) **Review the workflow**: The file '.github/workflows/auto_update.yml' runs on a schedule to check for updates.
3) **Permissions required** (via the built-in 'GITHUB_TOKEN'):
   - **contents: write** — needed to create and push the update branch.
   - **issues: write** — needed to create the notification Issue (can be disabled with --open-gh-issue=false).
   - **pull-requests: write** — needed to create the Pull Request (can be disabled with --open-gh-pr=false).
   - **models: read** (optional) — only required if using --use-gh-models flag for AI-generated summaries.
4) **Protect your branches**: Enable **branch protection rules** so automated changes **cannot** be pushed directly. All updates must go through a Pull Request for review.

### Optional: GitHub Models AI Summary

By default, the workflow does NOT use GitHub Models. To enable AI-generated summaries in Pull Requests:
  - Ensure your repository/organization has permissions to use GitHub Models.
  - Re-run: kubebuilder edit --plugins="autoupdate/v1-alpha" --use-gh-models

Without this flag, the workflow will still work but won't include AI summaries (avoiding 403 Forbidden errors).`

const pluginName = "autoupdate." + plugins.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 1, Stage: stage.Alpha}
	supportedProjectVersions = []config.Version{cfgv3.Version}
)

// Plugin implements the plugin.Full interface
type Plugin struct {
	editSubcommand
}

var _ plugin.Edit = Plugin{}

// PluginConfig defines the structure that will be used to track the data
type PluginConfig struct {
	UseGHModels bool `json:"useGHModels,omitempty"`
	OpenGHIssue bool `json:"openGHIssue,omitempty"`
	OpenGHPR    bool `json:"openGHPR,omitempty"`
}

// Name returns the name of the plugin
func (Plugin) Name() string { return pluginName }

// Version returns the version of the Helm plugin
func (Plugin) Version() plugin.Version { return pluginVersion }

// SupportedProjectVersions returns an array with all project versions supported by the plugin
func (Plugin) SupportedProjectVersions() []config.Version { return supportedProjectVersions }

// GetEditSubcommand will return the subcommand which is responsible for adding and/or edit a autoupdate
func (p Plugin) GetEditSubcommand() plugin.EditSubcommand { return &p.editSubcommand }

// Description returns a short description of the plugin
func (Plugin) Description() string {
	return "Proposes Kubebuilder scaffold updates via GitHub Actions"
}

// DeprecationWarning define the deprecation message or return empty when plugin is not deprecated
func (p Plugin) DeprecationWarning() string {
	return ""
}

// insertPluginMetaToConfig will insert the metadata to the plugin configuration
func insertPluginMetaToConfig(target config.Config, cfg PluginConfig) error {
	key := plugin.GetPluginKeyForConfig(target.GetPluginChain(), Plugin{})
	canonicalKey := plugin.KeyFor(Plugin{})

	// Decode existing config for validation, but don't overwrite flag values
	var existing PluginConfig
	if err := target.DecodePluginConfig(key, &existing); err != nil {
		switch {
		case errors.As(err, &config.UnsupportedFieldError{}):
			return nil
		case errors.As(err, &config.PluginKeyNotFoundError{}):
			if key != canonicalKey {
				if err2 := target.DecodePluginConfig(canonicalKey, &existing); err2 != nil {
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

	// Always encode the new config from flags (cfg parameter), not the existing one
	if err := target.EncodePluginConfig(key, cfg); err != nil {
		return fmt.Errorf("error encoding plugin configuration: %w", err)
	}

	return nil
}
