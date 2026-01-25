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

package scaffolds

import "fmt"

// validateNamespace validates that the namespace name follows Kubernetes naming conventions (DNS-1123 label).
//
// KUBERNETES NAMESPACE NAMING RULES:
// - Must be a valid DNS-1123 label
// - Contains only lowercase alphanumeric characters or '-'
// - Must start and end with an alphanumeric character
// - Maximum length: 63 characters
//
// This validation demonstrates how external plugins should validate user input
// before generating files. Invalid input should be caught early with clear error messages.
//
// See: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
func validateNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if len(namespace) > 63 {
		return fmt.Errorf("namespace must be 63 characters or less (got: %d)", len(namespace))
	}
	// Basic validation - alphanumeric and hyphens, must start/end with alphanumeric
	for i, r := range namespace {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("namespace must contain only lowercase alphanumeric characters or '-' (got: %c at position %d)", r, i)
		}
	}
	if namespace[0] == '-' || namespace[len(namespace)-1] == '-' {
		return fmt.Errorf("namespace must start and end with an alphanumeric character")
	}
	return nil
}
