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
	"errors"
	"fmt"

	"github.com/blang/semver"

	"sigs.k8s.io/kubebuilder/pkg/internal/validation"
)

// ValidateVersion ensures version adheres to the plugin version format,
// which is tolerant semver.
func ValidateVersion(version string) error {
	if version == "" {
		return errors.New("plugin version is empty")
	}
	// ParseTolerant allows versions with a "v" prefix or shortened versions,
	// ex. "3" or "v3.0".
	if _, err := semver.ParseTolerant(version); err != nil {
		return fmt.Errorf("failed to validate plugin version %q: %v", version, err)
	}
	return nil
}

// ValidateName ensures name is a valid DNS 1123 subdomain.
func ValidateName(name string) error {
	if errs := validation.IsDNS1123Subdomain(name); len(errs) != 0 {
		return fmt.Errorf("plugin name %q is invalid: %v", name, errs)
	}
	return nil
}
