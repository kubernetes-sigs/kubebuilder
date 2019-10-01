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

// Resource contains the information required to scaffold files for a resource.
type Resource struct {
	// Namespaced is true if the resource is namespaced
	Namespaced bool

	// Group is the API Group.  Does not contain the domain.
	Group string

	// GroupImportSafe is the API Group.  Does not contain the domain and it the "-"
	// It is used to do safe imports.
	GroupImportSafe string

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

	if err := IsDNS1123Subdomain(r.Group); err != nil {
		return fmt.Errorf("group name is invalid: (%v)", err)
	}

	r.GroupImportSafe = strings.Replace(r.Group, "-", "", -1)

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

// The following code came from "k8s.io/apimachinery/pkg/util/validation"
// If be required the usage of more funcs from this then please replace it for the import
// ---------------------------------------
const (
	dns1123LabelFmt          string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123SubdomainFmt      string = dns1123LabelFmt + "(\\." + dns1123LabelFmt + ")*"
	dns1123SubdomainErrorMsg string = "a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character"

	// dns1123SubdomainMaxLength is a subdomain's max length in DNS (RFC 1123)
	dns1123SubdomainMaxLength int = 253
)

var dns1123SubdomainRegexp = regexp.MustCompile("^" + dns1123SubdomainFmt + "$")

// IsDNS1123Subdomain tests for a string that conforms to the definition of a
// subdomain in DNS (RFC 1123).
func IsDNS1123Subdomain(value string) []string {
	var errs []string
	if len(value) > dns1123SubdomainMaxLength {
		errs = append(errs, maxLenError(dns1123SubdomainMaxLength))
	}
	if !dns1123SubdomainRegexp.MatchString(value) {
		errs = append(errs, regexError(dns1123SubdomainErrorMsg, dns1123SubdomainFmt, "example.com"))
	}
	return errs
}

// MaxLenError returns a string explanation of a "string too long" validation
// failure.
func maxLenError(length int) string {
	return fmt.Sprintf("must be no more than %d characters", length)
}

// RegexError returns a string explanation of a regex validation failure.
func regexError(msg string, fmt string, examples ...string) string {
	if len(examples) == 0 {
		return msg + " (regex used for validation is '" + fmt + "')"
	}
	msg += " (e.g. "
	for i := range examples {
		if i > 0 {
			msg += " or "
		}
		msg += "'" + examples[i] + "', "
	}
	msg += "regex used for validation is '" + fmt + "')"
	return msg
}

// ---------------------------------------
