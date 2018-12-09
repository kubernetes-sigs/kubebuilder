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

	"github.com/markbates/inflect"
)

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

	// Pattern is set if we are generating a extended pattern of resource (e.g. addon)
	Pattern Pattern
}

// Pattern is the enumerated type for patterns
type Pattern string

const (
	// PatternNone is the pattern constant for standard generation
	PatternNone = ""

	// PatternAddon is the pattern constant for addons
	PatternAddon = "addon"
)

// PatternAllValues is the list of valid pattern values
var PatternAllValues = []Pattern{PatternNone, PatternAddon}

// Get is part of the implementation of flag.Getter
func (p *Pattern) Get() interface{} {
	return *p
}

// String is part of the implementation of flag.Value
func (p *Pattern) String() string {
	return string(*p)
}

// Set is part of the implentation of flag.Value
func (p *Pattern) Set(s string) error {
	*p = Pattern(s)
	return nil
}

// Type is part of the implentation of pflag.Value
func (p *Pattern) Type() string {
	return "pattern"
}

// PatternAddon is true if the resource is using the addon pattern
func (r *Resource) PatternAddon() bool {
	return r.Pattern == PatternAddon
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

	rs := inflect.NewDefaultRuleset()
	if len(r.Resource) == 0 {
		r.Resource = rs.Pluralize(strings.ToLower(r.Kind))
	}

	groupMatch := regexp.MustCompile("^[a-z]+$")
	if !groupMatch.MatchString(r.Group) {
		return fmt.Errorf("group must match ^[a-z]+$ (was %s)", r.Group)
	}

	versionMatch := regexp.MustCompile("^v\\d+(alpha\\d+|beta\\d+)?$")
	if !versionMatch.MatchString(r.Version) {
		return fmt.Errorf(
			"version must match ^v\\d+(alpha\\d+|beta\\d+)?$ (was %s)", r.Version)
	}

	if r.Kind != inflect.Camelize(r.Kind) {
		return fmt.Errorf("Kind must be camelcase (expected %s was %s)", inflect.Camelize(r.Kind), r.Kind)
	}

	{
		found := false
		for _, f := range PatternAllValues {
			if f == r.Pattern {
				found = true
				continue
			}
		}
		if !found {
			var patternStrings []string
			for _, f := range PatternAllValues {
				patternStrings = append(patternStrings, string(f))
			}
			return fmt.Errorf("Pattern %q is not recognized, must be one of %s", strings.Join(patternStrings, ","))
		}
	}

	return nil
}
