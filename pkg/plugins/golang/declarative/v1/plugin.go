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

// Deprecated: The declarative plugin has been deprecated.
// The Declarative plugin is an implementation derived from the kubebuilder-declarative-pattern project.
// As the project maintainers possess the most comprehensive knowledge about its changes and Kubebuilder
// allows the creation of custom plugins using its library, it has been decided that this plugin will be
// better maintained within the kubebuilder-declarative-pattern project
// itself, which falls under its domain of responsibility. This decision aims to improve the maintainability
// of both the plugin and Kubebuilder, ultimately providing an enhanced user experience.
// To follow up on this work, please refer to the Issue #293:
// https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/issues/293.
package v1

import (
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"
)

const pluginName = "declarative." + golang.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 1}
	supportedProjectVersions = []config.Version{cfgv3.Version}
	pluginKey                = plugin.KeyFor(Plugin{})
)

var _ plugin.CreateAPI = Plugin{}

// Plugin implements the plugin.Full interface
type Plugin struct {
	initSubcommand
	createAPISubcommand
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

type pluginConfig struct {
	Resources []resource.GVK `json:"resources,omitempty"`
}

func (p Plugin) DeprecationWarning() string {
	return "The declarative plugin has been deprecated. \n" +
		"The Declarative plugin is an implementation derived from the kubebuilder-declarative-pattern project. " +
		"As the project maintainers possess the most comprehensive knowledge about its changes and Kubebuilder " +
		"allows the creation of custom plugins using its library, it has been decided that this plugin will be  " +
		"better maintained within the kubebuilder-declarative-pattern project " +
		"itself, which falls under its domain of responsibility. This decision aims to improve the maintainability " +
		"of both the plugin and Kubebuilder, ultimately providing an enhanced user experience." +
		"To follow up on this work, please refer to the Issue #293: " +
		"https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/issues/293."
}
