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

	"github.com/blang/semver"

	"sigs.k8s.io/kubebuilder/pkg/internal/validation"
)

// errInvalidVersion should be returned when a plugin's version is invalid.
type errInvalidVersion struct {
	version, msg string
}

func (e errInvalidVersion) Error() string {
	if e.version == "" {
		return fmt.Sprintf("plugin version is empty")
	}
	return fmt.Sprintf("invalid plugin version %q: %s", e.version, e.msg)
}

// ValidateVersion ensures version adheres to the plugin version format,
// which is tolerant semver.
func ValidateVersion(version string) error {
	if version == "" {
		return errInvalidVersion{}
	}
	// ParseTolerant allows versions with a "v" prefix or shortened versions,
	// ex. "3" or "v3.0".
	v, err := semver.ParseTolerant(version)
	if err != nil {
		return errInvalidVersion{version, err.Error()}
	}
	cv := semver.Version{Major: v.Major, Minor: v.Minor}
	if !v.Equals(cv) {
		return errInvalidVersion{version, "must contain major and minor version only"}
	}

	return nil
}

// errInvalidName should be returned when a plugin's name is invalid.
type errInvalidName struct {
	name, msg string
}

func (e errInvalidName) Error() string {
	if e.name == "" {
		return fmt.Sprintf("plugin name is empty")
	}
	return fmt.Sprintf("invalid plugin name %q: %s", e.name, e.msg)
}

// ValidateName ensures name is a valid DNS 1123 subdomain.
func ValidateName(name string) error {
	if name == "" {
		return errInvalidName{}
	}
	if errs := validation.IsDNS1123Subdomain(name); len(errs) != 0 {
		return errInvalidName{name, fmt.Sprintf("%s", errs)}
	}
	return nil
}
