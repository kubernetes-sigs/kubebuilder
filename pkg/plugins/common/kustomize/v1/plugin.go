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

//go:deprecated This package has been deprecated in favor of v2
package v1

import (
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
)

// KustomizeVersion is the kubernetes-sigs/kustomize version to be used in the project
const KustomizeVersion = "v3.8.7"

const pluginName = "kustomize.common." + plugins.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 1}
	supportedProjectVersions = []config.Version{cfgv3.Version}
)

var (
	_ plugin.Init          = Plugin{}
	_ plugin.CreateAPI     = Plugin{}
	_ plugin.CreateWebhook = Plugin{}
)

// Plugin implements the plugin.Full interface
type Plugin struct {
	initSubcommand
	createAPISubcommand
	createWebhookSubcommand
}

// Name returns the name of the plugin
func (Plugin) Name() string { return pluginName }

// Version returns the version of the plugin
func (Plugin) Version() plugin.Version { return pluginVersion }

// SupportedProjectVersions returns an array with all project versions supported by the plugin
func (Plugin) SupportedProjectVersions() []config.Version { return supportedProjectVersions }

// GetInitSubcommand will return the subcommand which is responsible for scaffolding init project
func (p Plugin) GetInitSubcommand() plugin.InitSubcommand { return &p.initSubcommand }

// GetCreateAPISubcommand will return the subcommand which is responsible for scaffolding apis
func (p Plugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand { return &p.createAPISubcommand }

// GetCreateWebhookSubcommand will return the subcommand which is responsible for scaffolding webhooks
func (p Plugin) GetCreateWebhookSubcommand() plugin.CreateWebhookSubcommand {
	return &p.createWebhookSubcommand
}

func (p Plugin) DeprecationWarning() string {
	return "This version is deprecated.The kustomize/v1 plugin used within go/v3 projects uses an old version " +
		"of kustomize (v3). It is recommended that you upgrade your project to use the go/v4 and kustomize/v2 plugins."
}
