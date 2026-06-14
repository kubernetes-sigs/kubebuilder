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

package v4

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

func validateExternalAPIPathFlag(externalAPIPath string) error {
	if externalAPIPath == "" {
		return nil
	}
	if err := resource.ValidateExternalAPIPath(
		externalAPIPath,
		"Use '--external-api-module' for module@version dependencies",
	); err != nil {
		return fmt.Errorf("invalid '--external-api-path': %w", err)
	}

	return nil
}
