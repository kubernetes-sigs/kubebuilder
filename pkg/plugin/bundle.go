/*
Copyright 2021 The Kubernetes Authors.

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

package plugin

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
)

type bundle struct {
	name    string
	version Version
	plugins []Plugin

	supportedProjectVersions []config.Version
}

// NewBundle creates a new Bundle with the provided name and version, and that wraps the provided plugins.
// The list of supported project versions is computed from the provided plugins.
func NewBundle(name string, version Version, plugins ...Plugin) (Bundle, error) {
	supportedProjectVersions := CommonSupportedProjectVersions(plugins...)
	if len(supportedProjectVersions) == 0 {
		return nil, fmt.Errorf("in order to bundle plugins, they must all support at least one common project version")
	}

	// Plugins may be bundles themselves, so unbundle here
	// NOTE(Adirio): unbundling here ensures that Bundle.Plugin always returns a flat list of Plugins instead of also
	//               including Bundles, and therefore we don't have to use a recursive algorithm when resolving.
	allPlugins := make([]Plugin, 0, len(plugins))
	for _, plugin := range plugins {
		if pluginBundle, isBundle := plugin.(Bundle); isBundle {
			allPlugins = append(allPlugins, pluginBundle.Plugins()...)
		} else {
			allPlugins = append(allPlugins, plugin)
		}
	}

	return bundle{
		name:                     name,
		version:                  version,
		plugins:                  allPlugins,
		supportedProjectVersions: supportedProjectVersions,
	}, nil
}

// Name implements Plugin
func (b bundle) Name() string {
	return b.name
}

// Version implements Plugin
func (b bundle) Version() Version {
	return b.version
}

// SupportedProjectVersions implements Plugin
func (b bundle) SupportedProjectVersions() []config.Version {
	return b.supportedProjectVersions
}

// Plugins implements Bundle
func (b bundle) Plugins() []Plugin {
	return b.plugins
}
