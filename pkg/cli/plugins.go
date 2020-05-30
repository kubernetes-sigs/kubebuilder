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

	"github.com/blang/semver"

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
// - Fully qualified key: "go.kubebuilder.io/v2.0"
// - Short key: "go/v2.0"
// - Fully qualified name: "go.kubebuilder.io"
// - Short name: "go"
// Some of these keys may conflict, ex. the fully-qualified and short names of
// "go.kubebuilder.io/v1.0" and "go.kubebuilder.io/v2.0" have ambiguous
// unversioned names "go.kubernetes.io" and "go". If pluginKey is ambiguous
// or does not match any known plugin's key, an error is returned.
//
// This function does not guarantee that the resolved set contains a plugin
// for each plugin type, i.e. an Init plugin might not be returned.
func resolvePluginsByKey(versionedPlugins []plugin.Base, pluginKey string) (resolved []plugin.Base, err error) {

	name, version := plugin.SplitKey(pluginKey)

	// Compare versions first to narrow the list of name comparisons.
	if version == "" {
		// Case: if plugin key has no version, check all plugin names.
		resolved = versionedPlugins
	} else {
		// Case: if plugin key has version, filter by version.
		resolved = findPluginsMatchingVersion(versionedPlugins, version)
	}

	if len(resolved) == 0 {
		return nil, errAmbiguousPlugin{pluginKey, "no versions match"}
	}

	// Compare names, taking into account whether name is fully-qualified or not.
	shortName := plugin.GetShortName(name)
	if name == shortName {
		// Case: if plugin name is short, find matching short names.
		resolved = findPluginsMatchingShortName(resolved, shortName)
	} else {
		// Case: if plugin name is fully-qualified, match only fully-qualified names.
		resolved = findPluginsMatchingName(resolved, name)
	}

	if len(resolved) == 0 {
		return nil, errAmbiguousPlugin{pluginKey, "no names match"}
	}

	// Since plugins has already been resolved by matching names and versions,
	// it should only contain one matching value for a versionless pluginKey if
	// it isn't ambiguous.
	if version == "" {
		if len(resolved) == 1 {
			return resolved, nil
		}
		return nil, errAmbiguousPlugin{pluginKey, fmt.Sprintf("possible keys: %+q", makePluginKeySlice(resolved...))}
	}

	rp, err := resolveToPlugin(resolved)
	if err != nil {
		return nil, errAmbiguousPlugin{pluginKey, err.Error()}
	}
	return []plugin.Base{rp}, nil
}

// findPluginsMatchingVersion returns a set of plugins with Version() matching
// version. The set will contain plugins with either major and minor versions
// matching exactly or major versions matching exactly and greater minor versions,
// but not a mix of the two match types.
func findPluginsMatchingVersion(plugins []plugin.Base, version string) []plugin.Base {
	// Assume versions have been validated already.
	v := must(semver.ParseTolerant(version))

	var equal, matchingMajor []plugin.Base
	for _, p := range plugins {
		pv := must(semver.ParseTolerant(p.Version()))
		if v.Major == pv.Major {
			if v.Minor == pv.Minor {
				equal = append(equal, p)
			} else if v.Minor < pv.Minor {
				matchingMajor = append(matchingMajor, p)
			}
		}
	}

	if len(equal) != 0 {
		return equal
	}
	return matchingMajor
}

// must wraps semver.Parse and panics if err is non-nil.
func must(v semver.Version, err error) semver.Version {
	if err != nil {
		panic(err)
	}
	return v
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

// resolveToPlugin returns a single plugin from plugins given the following
// conditions about plugins:
// 1. len(plugins) > 0.
// 2. No two plugin names are different.
// An error is returned if either condition is invalidated.
func resolveToPlugin(plugins []plugin.Base) (rp plugin.Base, err error) {
	// Versions are either an exact match or have greater minor versions, so
	// we choose the last in a sorted list of versions to get the correct one.
	versions := make([]semver.Version, len(plugins))
	for i, p := range plugins {
		versions[i] = must(semver.ParseTolerant(p.Version()))
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("possible versions: %+q", versions)
	}
	semver.Sort(versions)
	useVersion := versions[len(versions)-1]

	// If more than one name in plugins exists, the name portion of pluginKey
	// needs to be more specific.
	nameSet := make(map[string]struct{})
	for _, p := range plugins {
		nameSet[p.Name()] = struct{}{}
		// This condition will only be true once for an unambiguous plugin name,
		// since plugin keys have been checked for duplicates already.
		if must(semver.ParseTolerant(p.Version())).Equals(useVersion) {
			rp = p
		}
	}
	if len(nameSet) != 1 {
		return nil, fmt.Errorf("possible names: %+q", makeKeySlice(nameSet))
	}

	return rp, nil
}

// makeKeySlice returns a slice of all map keys in set.
func makeKeySlice(set map[string]struct{}) (keys []string) {
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return
}

// makePluginKeySlice returns a slice of all keys for each plugin in plugins.
func makePluginKeySlice(plugins ...plugin.Base) (keys []string) {
	for _, p := range plugins {
		keys = append(keys, plugin.KeyFor(p))
	}
	sort.Strings(keys)
	return
}
