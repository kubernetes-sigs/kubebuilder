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

package resource

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gobuffalo/flect"
)

// GroupMatchRegex is the regex used to validate groupNames
const GroupMatchRegex = "^[a-zA-Z0-9]+[a-zA-Z0-9-_]*[a-zA-Z0-9]+$"

// Resource contains the information required to scaffold files for a resource.
type Resource struct {
	// Namespaced is true if the resource is namespaced
	Namespaced bool

	// Group is the API Group.  Does not contain the domain.
	Group string

	// Version is the API version - e.g. v1beta1
	Version string

	// Kind is the API Kind.
	Kind string

	// Resource is the API Resource.
	Resource string

	// ShortNames is the list of resource shortnames.
	ShortNames []string

	// CreateExampleReconcileBody will create a Deployment in the Reconcile example
	CreateExampleReconcileBody bool
}

// Validate checks the Resource values to make sure they are valid.
func (r *Resource) Validate() error {
	if len(r.Group) == 0 {
		return fmt.Errorf("group cannot be empty")
	}
	if len(r.Version) == 0 {
		return fmt.Errorf("version cannot be empty")
	}
	if len(r.Kind) == 0 {
		return fmt.Errorf("kind cannot be empty")
	}

	if len(r.Resource) == 0 {
		r.Resource = flect.Pluralize(strings.ToLower(r.Kind))
	}

	groupMatch := regexp.MustCompile(GroupMatchRegex)
	if !groupMatch.MatchString(r.Group) {
		return fmt.Errorf("group must match %s (was %s)", GroupMatchRegex, r.Group)
	}

	versionMatch := regexp.MustCompile("^v\\d+(alpha\\d+|beta\\d+)?$")
	if !versionMatch.MatchString(r.Version) {
		return fmt.Errorf(
			"version must match ^v\\d+(alpha\\d+|beta\\d+)?$ (was %s)", r.Version)
	}
	if r.Kind != flect.Pascalize(r.Kind) {
		return fmt.Errorf("kind must be camelcase (expected %s was %s)", flect.Pascalize(r.Kind), r.Kind)
	}

	return nil
}
