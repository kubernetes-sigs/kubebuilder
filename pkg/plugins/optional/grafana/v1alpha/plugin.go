/*
Copyright 2022 The Kubernetes Authors.

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
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/grafana/v1alpha/scaffolds"
)

const pluginName = "grafana." + plugins.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 1, Stage: stage.Alpha}
	supportedProjectVersions = []config.Version{cfgv3.Version}
	pluginKey                = plugin.KeyFor(&Plugin{})
)

// Plugin implements the plugin.Full interface
type Plugin struct {
	initSubcommand *subcommand
	editSubcommand *subcommand
}

var _ plugin.Init = &Plugin{}

// Name returns the name of the plugin
func (p *Plugin) Name() string { return pluginName }

// Version returns the version of the grafana plugin
func (p *Plugin) Version() plugin.Version { return pluginVersion }

// SupportedProjectVersions returns an array with all project versions supported by the plugin
func (p *Plugin) SupportedProjectVersions() []config.Version { return supportedProjectVersions }

// GetInitSubcommand will return the subcommand which is responsible for initializing and scaffolding grafana manifests
func (p *Plugin) GetInitSubcommand() plugin.InitSubcommand {
	if p.initSubcommand == nil {
		p.initSubcommand = &subcommand{
			cmd:                "init",
			exampleDescription: "# Initialize a common project with this plugin",
			scaffolder:         scaffolds.NewInitScaffolder(),
		}
	}

	return p.initSubcommand
}

// GetEditSubcommand will return the subcommand which is responsible for adding grafana manifests
func (p *Plugin) GetEditSubcommand() plugin.EditSubcommand {
	if p.editSubcommand == nil {
		p.editSubcommand = &subcommand{
			cmd:                "edit",
			exampleDescription: "# Edit a common project with this plugin",
			scaffolder:         scaffolds.NewEditScaffolder(),
		}
	}

	return p.editSubcommand
}

type pluginConfig struct{}

// DeprecationWarning define the deprecation message or return empty when plugin is not deprecated
func (p *Plugin) DeprecationWarning() string {
	return ""
}
