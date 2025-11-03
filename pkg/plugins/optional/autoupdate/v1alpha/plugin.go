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
const metaDataDescription = `This plugin scaffolds a GitHub Action that helps you keep your project aligned with the latest Kubebuilder improvements. With a tiny amount of setup, you’ll receive **automatic issue notifications** whenever a new Kubebuilder release is available. Each issue includes a **compare link** so you can open a Pull Request with one click and review the changes safely.

Under the hood, the workflow runs 'kubebuilder alpha update' using a **3-way merge strategy** to refresh your scaffold while preserving your code. It creates and pushes an update branch, then opens a GitHub **Issue** containing the PR URL you can use to review and merge.

### How to set it up

1) **Add the plugin**: Use the Kubebuilder CLI to scaffold the automation into your repo.
2) **Review the workflow**: The file '.github/workflows/auto_update.yml' runs on a schedule to check for updates.
3) **Permissions required** (via the built-in 'GITHUB_TOKEN'):
   - **contents: write** — needed to create and push the update branch.
   - **issues: write** — needed to create the tracking Issue with the PR link.
4) **Protect your branches**: Enable **branch protection rules** so automated changes **cannot** be pushed directly. All updates must go through a Pull Request for review.`

const pluginName = "autoupdate." + plugins.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 1, Stage: stage.Alpha}
	supportedProjectVersions = []config.Version{cfgv3.Version}
)

// Plugin implements the plugin.Full interface
type Plugin struct {
	editSubcommand
	initSubcommand
}

var _ plugin.Init = Plugin{}

type pluginConfig struct{}

// Name returns the name of the plugin
func (Plugin) Name() string { return pluginName }

// Version returns the version of the Helm plugin
func (Plugin) Version() plugin.Version { return pluginVersion }

// SupportedProjectVersions returns an array with all project versions supported by the plugin
func (Plugin) SupportedProjectVersions() []config.Version { return supportedProjectVersions }

// GetEditSubcommand will return the subcommand which is responsible for adding and/or edit a autoupdate
func (p Plugin) GetEditSubcommand() plugin.EditSubcommand { return &p.editSubcommand }

// GetInitSubcommand will return the subcommand which is responsible for init autoupdate plugin
func (p Plugin) GetInitSubcommand() plugin.InitSubcommand { return &p.initSubcommand }

// DeprecationWarning define the deprecation message or return empty when plugin is not deprecated
func (p Plugin) DeprecationWarning() string {
	return ""
}

// insertPluginMetaToConfig will insert the metadata to the plugin configuration
func insertPluginMetaToConfig(target config.Config, cfg pluginConfig) error {
	key := plugin.GetPluginKeyForConfig(target.GetPluginChain(), Plugin{})
	canonicalKey := plugin.KeyFor(Plugin{})

	if err := target.DecodePluginConfig(key, &cfg); err != nil {
		switch {
		case errors.As(err, &config.UnsupportedFieldError{}):
			return nil
		case errors.As(err, &config.PluginKeyNotFoundError{}):
			if key != canonicalKey {
				if err2 := target.DecodePluginConfig(canonicalKey, &cfg); err2 != nil {
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

	if err := target.EncodePluginConfig(key, cfg); err != nil {
		return fmt.Errorf("error encoding plugin configuration: %w", err)
	}

	return nil
}
