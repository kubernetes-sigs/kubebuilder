/*
Copyright 2024 The Kubernetes Authors.

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

package utils

import (
	"fmt"
	"os"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
)

// ImplementWebhooks will mock an webhook data
func ImplementWebhooks(filename, lowerKind string) error {
	// false positive
	// nolint:gosec
	bs, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	str := string(bs)

	str, err = util.EnsureExistAndReplace(
		str,
		"import (",
		`import (
	"errors"`)
	if err != nil {
		return err
	}

	// implement defaulting webhook logic
	replace := fmt.Sprintf(`if %s.Spec.Count == 0 {
		%s.Spec.Count = 5
	}`, lowerKind, lowerKind)
	str, err = util.EnsureExistAndReplace(
		str,
		"// TODO(user): fill in your defaulting logic.",
		replace,
	)
	if err != nil {
		return err
	}

	// implement validation webhook logic
	str, err = util.EnsureExistAndReplace(
		str,
		"// TODO(user): fill in your validation logic upon object creation.",
		fmt.Sprintf(`if %s.Spec.Count < 0 {
		return nil, errors.New(".spec.count must >= 0")
	}`, lowerKind))
	if err != nil {
		return err
	}
	str, err = util.EnsureExistAndReplace(
		str,
		"// TODO(user): fill in your validation logic upon object update.",
		fmt.Sprintf(`if %s.Spec.Count < 0 {
		return nil, errors.New(".spec.count must >= 0")
	}`, lowerKind))
	if err != nil {
		return err
	}
	// false positive
	// nolint:gosec
	return os.WriteFile(filename, []byte(str), 0644)
}
