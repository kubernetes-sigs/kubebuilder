/*
Copyright 2026 The Kubernetes Authors.

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

package v1alpha

import (
	"fmt"
	"slices"
	"strings"
)

// validateProvider returns an error when p is not one of the recognised provider names.
func validateProvider(p string) error {
	if slices.Contains(validProviders, p) {
		return nil
	}
	return fmt.Errorf("unknown provider %q; valid values: %s", p, strings.Join(validProviders, ", "))
}
