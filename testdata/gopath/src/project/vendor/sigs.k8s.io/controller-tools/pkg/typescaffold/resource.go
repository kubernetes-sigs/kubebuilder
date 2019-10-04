/*
Copyright 2018 The Kubernetes Authors.

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

package typescaffold

import (
	"fmt"
	"strings"

	"github.com/markbates/inflect"
)

// Resource contains the information required to scaffold files for a resource.
type Resource struct {
	// Namespaced is true if the resource is namespaced
	Namespaced bool

	// Kind is the API Kind.
	Kind string

	// Resource is the API Resource.
	Resource string
}

// Validate checks the Resource values to make sure they are valid.
func (r *Resource) Validate() error {
	if len(r.Kind) == 0 {
		return fmt.Errorf("kind cannot be empty")
	}

	rs := inflect.NewDefaultRuleset()
	if len(r.Resource) == 0 {
		r.Resource = rs.Pluralize(strings.ToLower(r.Kind))
	}

	if r.Kind != inflect.Camelize(r.Kind) {
		return fmt.Errorf("Kind must be camelcase (expected %s was %s)", inflect.Camelize(r.Kind), r.Kind)
	}

	return nil
}
