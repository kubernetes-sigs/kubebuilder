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
	"strings"

	"sigs.k8s.io/kubebuilder/v2/pkg/internal/validation"
)

// Key returns a unique identifying string for a plugin's name and version.
func Key(name, version string) string {
	if version == "" {
		return name
	}
	return path.Join(name, "v"+strings.TrimLeft(version, "v"))
}

// KeyFor returns a Plugin's unique identifying string.
func KeyFor(p Plugin) string {
	return Key(p.Name(), p.Version().String())
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
func GetShortName(name string) string {
	return strings.SplitN(name, ".", 2)[0]
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
		if err := validation.ValidateProjectVersion(projectVersion); err != nil {
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
		if _, err := ParseVersion(version); err != nil {
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

// SupportsVersion checks if a plugins supports a project version.
func SupportsVersion(p Plugin, projectVersion string) bool {
	for _, version := range p.SupportedProjectVersions() {
		if version == projectVersion {
			return true
		}
	}
	return false
}
