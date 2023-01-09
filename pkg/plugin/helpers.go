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

package plugin

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/internal/validation"
)

// KeyFor returns a Plugin's unique identifying string.
func KeyFor(p Plugin) string {
	return path.Join(p.Name(), p.Version().String())
}

// SplitKey returns a name and version for a plugin key.
func SplitKey(key string) (string, string) {
	if !strings.Contains(key, "/") {
		return key, ""
	}
	keyParts := strings.SplitN(key, "/", 2)
	return keyParts[0], keyParts[1]
}

// GetShortName returns plugin's short name (name before domain) if name
// is fully qualified (has a domain suffix), otherwise GetShortName returns name.
// Deprecated
func GetShortName(name string) string {
	return strings.SplitN(name, ".", 2)[0]
}

// Deprecated: it was added to ensure backwards compatibility and should
// be removed when we remove the go/v3 plugin
// IsLegacyLayout returns true when is possible to identify that the project
// was scaffolded with the previous layout
func IsLegacyLayout(config config.Config) bool {
	for _, pluginKey := range config.GetPluginChain() {
		if strings.Contains(pluginKey, "go.kubebuilder.io/v3") || strings.Contains(pluginKey, "go.kubebuilder.io/v2") {
			return true
		}
	}
	return false
}

// Validate ensures a Plugin is valid.
func Validate(p Plugin) error {
	if err := validateName(p.Name()); err != nil {
		return fmt.Errorf("invalid plugin name %q: %v", p.Name(), err)
	}
	if err := p.Version().Validate(); err != nil {
		return fmt.Errorf("invalid plugin version %q: %v", p.Version(), err)
	}
	if len(p.SupportedProjectVersions()) == 0 {
		return fmt.Errorf("plugin %q must support at least one project version", KeyFor(p))
	}
	for _, projectVersion := range p.SupportedProjectVersions() {
		if err := projectVersion.Validate(); err != nil {
			return fmt.Errorf("plugin %q supports an invalid project version %q: %v", KeyFor(p), projectVersion, err)
		}
	}
	return nil
}

// ValidateKey ensures both plugin name and version are valid.
func ValidateKey(key string) error {
	name, version := SplitKey(key)
	if err := validateName(name); err != nil {
		return fmt.Errorf("invalid plugin name %q: %v", name, err)
	}
	// CLI-set plugins do not have to contain a version.
	if version != "" {
		var v Version
		if err := v.Parse(version); err != nil {
			return fmt.Errorf("invalid plugin version %q: %v", version, err)
		}
	}
	return nil
}

// validateName ensures name is a valid DNS 1123 subdomain.
func validateName(name string) error {
	if errs := validation.IsDNS1123Subdomain(name); len(errs) != 0 {
		return fmt.Errorf("invalid plugin name %q: %v", name, errs)
	}
	return nil
}

// SupportsVersion checks if a plugin supports a project version.
func SupportsVersion(p Plugin, projectVersion config.Version) bool {
	for _, version := range p.SupportedProjectVersions() {
		if projectVersion.Compare(version) == 0 {
			return true
		}
	}
	return false
}

// CommonSupportedProjectVersions returns the projects versions that are supported by all the provided Plugins
func CommonSupportedProjectVersions(plugins ...Plugin) []config.Version {
	// Count how many times each supported project version appears
	supportedProjectVersionCounter := make(map[config.Version]int)
	for _, plugin := range plugins {
		for _, supportedProjectVersion := range plugin.SupportedProjectVersions() {
			if _, exists := supportedProjectVersionCounter[supportedProjectVersion]; !exists {
				supportedProjectVersionCounter[supportedProjectVersion] = 1
			} else {
				supportedProjectVersionCounter[supportedProjectVersion]++
			}
		}
	}

	// Check which versions are present the expected number of times
	supportedProjectVersions := make([]config.Version, 0, len(supportedProjectVersionCounter))
	expectedTimes := len(plugins)
	for supportedProjectVersion, times := range supportedProjectVersionCounter {
		if times == expectedTimes {
			supportedProjectVersions = append(supportedProjectVersions, supportedProjectVersion)
		}
	}

	// Sort the output to guarantee consistency
	sort.Slice(supportedProjectVersions, func(i int, j int) bool {
		return supportedProjectVersions[i].Compare(supportedProjectVersions[j]) == -1
	})

	return supportedProjectVersions
}
