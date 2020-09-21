/*
Copyright 2020 The Kubernetes Authors.

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

package v2

import (
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

const pluginName = "go" + plugin.DefaultNameQualifier

var (
	supportedProjectVersions = []string{config.Version2, config.Version3Alpha}
	pluginVersion            = plugin.Version{Number: 2}
)

var (
	_ plugin.Base                      = Plugin{}
	_ plugin.InitPluginGetter          = Plugin{}
	_ plugin.CreateAPIPluginGetter     = Plugin{}
	_ plugin.CreateWebhookPluginGetter = Plugin{}
)

// Plugin defines the plugins operations for the v2 plugin version.
type Plugin struct {
	initPlugin
	createAPIPlugin
	createWebhookPlugin
}

// Name returns the name of the plugin for the v2 which is in this case `go.kubebuilder.io`
func (Plugin) Name() string { return pluginName }

// Version returns the version of the plugin which in this case is 2
func (Plugin) Version() plugin.Version { return pluginVersion }

// SupportedProjectVersions returns an array with all versions project versions are supported by the plugin
// E.g a plugin can be used with projects that were built with the PROJECT version 3-alpha but not in the Project
// version 2. See that the PROJECT version is defined in the attribute version of the PROJECT file.
func (Plugin) SupportedProjectVersions() []string { return supportedProjectVersions }

// GetInitPlugin will return the plugin versions for v2 which is responsible for initialized and scaffold the project
func (p Plugin) GetInitPlugin() plugin.Init { return &p.initPlugin }

// GetCreateAPIPlugin will return the plugin for v2 which is responsible for scaffold apis
func (p Plugin) GetCreateAPIPlugin() plugin.CreateAPI { return &p.createAPIPlugin }

// GetCreateWebhookPlugin will return the plugin for v2 which is responsible for scaffold webhooks for the project
func (p Plugin) GetCreateWebhookPlugin() plugin.CreateWebhook { return &p.createWebhookPlugin }
