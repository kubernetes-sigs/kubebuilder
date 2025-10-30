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

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/internal/validation"
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

// GetPluginKeyForConfig finds which key to use when saving plugin config.
// When a plugin is wrapped in a bundle, the bundle's key appears in the chain instead of the plugin's key.
// For example: "deploy-image.my-domain/v1-alpha" wraps "deploy-image.go.kubebuilder.io/v1-alpha".
// Returns the plugin's own key if nothing matches.
func GetPluginKeyForConfig(pluginChain []string, p Plugin) string {
	pluginKey := KeyFor(p)

	// Try exact match first
	for _, key := range pluginChain {
		if key == pluginKey {
			return pluginKey
		}
	}

	// No exact match. Try matching by base name + version to find bundled plugins.
	pluginName, _ := SplitKey(pluginKey)
	pluginVersion := p.Version().String()

	// Get base name (part before first dot): "deploy-image.go.kubebuilder.io" -> "deploy-image"
	baseName := pluginName
	if idx := strings.Index(pluginName, "."); idx != -1 {
		baseName = pluginName[:idx]
	}

	for _, key := range pluginChain {
		name, version := SplitKey(key)
		if version != pluginVersion {
			continue
		}

		// Check if this key matches the base name
		keyBaseName := name
		if idx := strings.Index(name, "."); idx != -1 {
			keyBaseName = name[:idx]
		}

		if keyBaseName == baseName {
			return key
		}
	}

	// Nothing matched, use plugin's own key
	return pluginKey
}

// Validate ensures a Plugin is valid.
func Validate(p Plugin) error {
	if err := validateName(p.Name()); err != nil {
		return fmt.Errorf("invalid plugin name %q: %w", p.Name(), err)
	}
	if err := p.Version().Validate(); err != nil {
		return fmt.Errorf("invalid plugin version %q: %w", p.Version(), err)
	}
	if len(p.SupportedProjectVersions()) == 0 {
		return fmt.Errorf("plugin %q must support at least one project version", KeyFor(p))
	}
	for _, projectVersion := range p.SupportedProjectVersions() {
		if err := projectVersion.Validate(); err != nil {
			return fmt.Errorf("plugin %q supports an invalid project version %q: %w", KeyFor(p), projectVersion, err)
		}
	}
	return nil
}

// ValidateKey ensures both plugin name and version are valid.
func ValidateKey(key string) error {
	name, version := SplitKey(key)
	if err := validateName(name); err != nil {
		return fmt.Errorf("invalid plugin name %q: %w", name, err)
	}
	// CLI-set plugins do not have to contain a version.
	if version != "" {
		var v Version
		if err := v.Parse(version); err != nil {
			return fmt.Errorf("invalid plugin version %q: %w", version, err)
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
