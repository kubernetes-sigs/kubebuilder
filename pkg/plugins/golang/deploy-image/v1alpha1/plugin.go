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

package v1alpha1

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

const pluginName = "deploy-image." + golang.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 1, Stage: stage.Alpha}
	supportedProjectVersions = []config.Version{cfgv3.Version}
	pluginKey                = plugin.KeyFor(Plugin{})
)

var _ plugin.CreateAPI = Plugin{}

// Plugin implements the plugin.Full interface
type Plugin struct {
	createAPISubcommand
}

// Name returns the name of the plugin
func (Plugin) Name() string { return pluginName }

// Version returns the version of the plugin
func (Plugin) Version() plugin.Version { return pluginVersion }

// SupportedProjectVersions returns an array with all project versions supported by the plugin
func (Plugin) SupportedProjectVersions() []config.Version { return supportedProjectVersions }

// GetCreateAPISubcommand will return the subcommand which is responsible for scaffolding apis
func (p Plugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand { return &p.createAPISubcommand }

// PluginConfig defines the structure that will be used to track the data
type PluginConfig struct {
	Resources []ResourceData `json:"resources,omitempty"`
}

// ResourceData store the resource data used in the plugin
type ResourceData struct {
	Group   string  `json:"group,omitempty"`
	Domain  string  `json:"domain,omitempty"`
	Version string  `json:"version"`
	Kind    string  `json:"kind"`
	Options options `json:"options,omitempty"`
}

type options struct {
	Image            string `json:"image,omitempty"`
	ContainerCommand string `json:"containerCommand,omitempty"`
	ContainerPort    string `json:"containerPort,omitempty"`
	RunAsUser        string `json:"runAsUser,omitempty"`
}

// DeprecationWarning define the deprecation message or return empty when plugin is not deprecated
func (p Plugin) DeprecationWarning() string {
	return ""
}
