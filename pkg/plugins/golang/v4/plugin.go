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

package v4

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

const pluginName = "base." + golang.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 4, Stage: stage.Stable}
	supportedProjectVersions = []config.Version{cfgv3.Version}
)

var _ plugin.Full = Plugin{}

// Plugin implements the plugin.Full interface
type Plugin struct {
	initSubcommand
	createAPISubcommand
	createWebhookSubcommand
	editSubcommand
}

// Name returns the name of the plugin
func (Plugin) Name() string { return pluginName }

// Version returns the version of the plugin
func (Plugin) Version() plugin.Version { return pluginVersion }

// SupportedProjectVersions returns an array with all project versions supported by the plugin
func (Plugin) SupportedProjectVersions() []config.Version { return supportedProjectVersions }

// GetInitSubcommand will return the subcommand which is responsible for initializing and common scaffolding
func (p Plugin) GetInitSubcommand() plugin.InitSubcommand { return &p.initSubcommand }

// GetCreateAPISubcommand will return the subcommand which is responsible for scaffolding apis
func (p Plugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand { return &p.createAPISubcommand }

// GetCreateWebhookSubcommand will return the subcommand which is responsible for scaffolding webhooks
func (p Plugin) GetCreateWebhookSubcommand() plugin.CreateWebhookSubcommand {
	return &p.createWebhookSubcommand
}

// GetEditSubcommand will return the subcommand which is responsible for editing the scaffold of the project
func (p Plugin) GetEditSubcommand() plugin.EditSubcommand { return &p.editSubcommand }

func (p Plugin) DeprecationWarning() string {
	return ""
}
