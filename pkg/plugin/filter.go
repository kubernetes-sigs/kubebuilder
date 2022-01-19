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

package plugin

import (
	"strings"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
)

// FilterPluginsByKey returns the set of plugins that match the provided key (may be not-fully qualified)
func FilterPluginsByKey(plugins []Plugin, key string) ([]Plugin, error) {
	name, ver := SplitKey(key)
	hasVersion := ver != ""
	var version Version
	if hasVersion {
		if err := version.Parse(ver); err != nil {
			return nil, err
		}
	}

	filtered := make([]Plugin, 0, len(plugins))
	for _, plugin := range plugins {
		if !strings.HasPrefix(plugin.Name(), name) {
			continue
		}
		if hasVersion && plugin.Version().Compare(version) != 0 {
			continue
		}
		filtered = append(filtered, plugin)
	}
	return filtered, nil
}

// FilterPluginsByProjectVersion returns the set of plugins that support the provided project version
func FilterPluginsByProjectVersion(plugins []Plugin, projectVersion config.Version) []Plugin {
	filtered := make([]Plugin, 0, len(plugins))
	for _, plugin := range plugins {
		for _, supportedVersion := range plugin.SupportedProjectVersions() {
			if supportedVersion.Compare(projectVersion) == 0 {
				filtered = append(filtered, plugin)
				break
			}
		}
	}
	return filtered
}
