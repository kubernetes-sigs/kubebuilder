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

package cli

import (
	"fmt"
	"sort"

	"sigs.k8s.io/kubebuilder/pkg/internal/validation"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

// errAmbiguousPlugin should be returned when an ambiguous plugin key is
// found.
type errAmbiguousPlugin struct {
	key, msg string
}

func (e errAmbiguousPlugin) Error() string {
	return fmt.Sprintf("ambiguous plugin %q: %s", e.key, e.msg)
}

// resolvePluginsByKey resolves versionedPlugins to a subset of plugins by
// matching keys to some form of pluginKey. Those forms can be a:
// - Fully qualified key: "go.kubebuilder.io/v2"
// - Short key: "go/v2"
// - Fully qualified name: "go.kubebuilder.io"
// - Short name: "go"
// Some of these keys may conflict, ex. the fully-qualified and short names of
// "go.kubebuilder.io/v1" and "go.kubebuilder.io/v2" have ambiguous
// unversioned names "go.kubernetes.io" and "go". If pluginKey is ambiguous
// or does not match any known plugin's key, an error is returned.
//
// This function does not guarantee that the resolved set contains a plugin
// for each plugin type, i.e. an Init plugin might not be returned.
func resolvePluginsByKey(versionedPlugins []plugin.Base, pluginKey string) (resolved []plugin.Base, err error) {

	name, version := plugin.SplitKey(pluginKey)

	// Compare names, taking into account whether name is fully-qualified or not.
	shortName := plugin.GetShortName(name)
	if name == shortName {
		// Case: if plugin name is short, find matching short names.
		resolved = findPluginsMatchingShortName(versionedPlugins, shortName)
	} else {
		// Case: if plugin name is fully-qualified, match only fully-qualified names.
		resolved = findPluginsMatchingName(versionedPlugins, name)
	}

	if len(resolved) == 0 {
		return nil, errAmbiguousPlugin{
			key: pluginKey,
			msg: fmt.Sprintf("no names match, possible plugins: %+q", makePluginKeySlice(versionedPlugins...)),
		}
	}

	if version != "" {
		// Case: if plugin key has version, filter by version.
		v, err := plugin.ParseVersion(version)
		if err != nil {
			return nil, err
		}
		keys := makePluginKeySlice(resolved...)
		for i := 0; i < len(resolved); i++ {
			if v.Compare(resolved[i].Version()) != 0 {
				resolved = append(resolved[:i], resolved[i+1:]...)
				i--
			}
		}
		if len(resolved) == 0 {
			return nil, errAmbiguousPlugin{
				key: pluginKey,
				msg: fmt.Sprintf("no versions match, possible plugins: %+q", keys),
			}
		}
	}

	// Since plugins has already been resolved by matching names and versions,
	// it should only contain one matching value if it isn't ambiguous.
	if len(resolved) != 1 {
		return nil, errAmbiguousPlugin{
			key: pluginKey,
			msg: fmt.Sprintf("matching plugins: %+q", makePluginKeySlice(resolved...)),
		}
	}
	return resolved, nil
}

// findPluginsMatchingName returns a set of plugins with Name() exactly
// matching name.
func findPluginsMatchingName(plugins []plugin.Base, name string) (equal []plugin.Base) {
	for _, p := range plugins {
		if p.Name() == name {
			equal = append(equal, p)
		}
	}
	return equal
}

// findPluginsMatchingShortName returns a set of plugins with
// GetShortName(Name()) exactly matching shortName.
func findPluginsMatchingShortName(plugins []plugin.Base, shortName string) (equal []plugin.Base) {
	for _, p := range plugins {
		if plugin.GetShortName(p.Name()) == shortName {
			equal = append(equal, p)
		}
	}
	return equal
}

// makePluginKeySlice returns a slice of all keys for each plugin in plugins.
func makePluginKeySlice(plugins ...plugin.Base) (keys []string) {
	for _, p := range plugins {
		keys = append(keys, plugin.KeyFor(p))
	}
	sort.Strings(keys)
	return
}

// validatePlugins validates the name and versions of a list of plugins.
func validatePlugins(plugins ...plugin.Base) error {
	pluginKeySet := make(map[string]struct{}, len(plugins))
	for _, p := range plugins {
		if err := validatePlugin(p); err != nil {
			return err
		}
		// Check for duplicate plugin keys.
		pluginKey := plugin.KeyFor(p)
		if _, seen := pluginKeySet[pluginKey]; seen {
			return fmt.Errorf("two plugins have the same key: %q", pluginKey)
		}
		pluginKeySet[pluginKey] = struct{}{}
	}
	return nil
}

// validatePlugin validates the name and versions of a plugin.
func validatePlugin(p plugin.Base) error {
	pluginName := p.Name()
	if err := plugin.ValidateName(pluginName); err != nil {
		return fmt.Errorf("invalid plugin name %q: %v", pluginName, err)
	}
	if err := p.Version().Validate(); err != nil {
		return fmt.Errorf("invalid plugin version %q: %v", p.Version(), err)
	}
	for _, projectVersion := range p.SupportedProjectVersions() {
		if err := validation.ValidateProjectVersion(projectVersion); err != nil {
			return fmt.Errorf("invalid project version %q: %v", projectVersion, err)
		}
	}
	return nil
}
