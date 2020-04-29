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
	"path"
	"regexp"
	"strings"

	"github.com/gobuffalo/flect"

	"sigs.k8s.io/kubebuilder/pkg/internal/validation"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
)

const (
	versionPattern = "^v\\d+(alpha\\d+|beta\\d+)?$"

	groupRequired   = "group cannot be empty"
	versionRequired = "version cannot be empty"
	kindRequired    = "kind cannot be empty"
)

var (
	versionRegex = regexp.MustCompile(versionPattern)

	coreGroups = map[string]string{
		"admission":             "k8s.io",
		"admissionregistration": "k8s.io",
		"apps":                  "",
		"auditregistration":     "k8s.io",
		"apiextensions":         "k8s.io",
		"authentication":        "k8s.io",
		"authorization":         "k8s.io",
		"autoscaling":           "",
		"batch":                 "",
		"certificates":          "k8s.io",
		"coordination":          "k8s.io",
		"core":                  "",
		"events":                "k8s.io",
		"extensions":            "",
		"imagepolicy":           "k8s.io",
		"networking":            "k8s.io",
		"node":                  "k8s.io",
		"metrics":               "k8s.io",
		"policy":                "",
		"rbac.authorization":    "k8s.io",
		"scheduling":            "k8s.io",
		"setting":               "k8s.io",
		"storage":               "k8s.io",
	}
)

// Options contains the information required to build a new Resource
type Options struct {
	// Group is the API Group. Does not contain the domain.
	Group string

	// Version is the API version.
	Version string

	// Kind is the API Kind.
	Kind string

	// Plural is the API Kind plural form.
	// Optional
	Plural string

	// Namespaced is true if the resource is namespaced.
	Namespaced bool
}

// Validate verifies that all the fields have valid values
func (opts *Options) Validate() error {
	// Check that the required flags did not get a flag as their value
	// We can safely look for a '-' as the first char as none of the fields accepts it
	// NOTE: We must do this for all the required flags first or we may output the wrong
	// error as flags may seem to be missing because Cobra assigned them to another flag.
	if strings.HasPrefix(opts.Group, "-") {
		return fmt.Errorf(groupRequired)
	}
	if strings.HasPrefix(opts.Version, "-") {
		return fmt.Errorf(versionRequired)
	}
	if strings.HasPrefix(opts.Kind, "-") {
		return fmt.Errorf(kindRequired)
	}
	// Now we can check that all the required flags are not empty
	if len(opts.Group) == 0 {
		return fmt.Errorf(groupRequired)
	}
	if len(opts.Version) == 0 {
		return fmt.Errorf(versionRequired)
	}
	if len(opts.Kind) == 0 {
		return fmt.Errorf(kindRequired)
	}

	// Check if the Group has a valid DNS1123 subdomain value
	if err := validation.IsDNS1123Subdomain(opts.Group); err != nil {
		return fmt.Errorf("group name is invalid: (%v)", err)
	}

	// Check if the version follows the valid pattern
	if !versionRegex.MatchString(opts.Version) {
		return fmt.Errorf("version must match %s (was %s)", versionPattern, opts.Version)
	}

	validationErrors := []string{}

	// require Kind to start with an uppercase character
	if string(opts.Kind[0]) == strings.ToLower(string(opts.Kind[0])) {
		validationErrors = append(validationErrors, "kind must start with an uppercase character")
	}

	validationErrors = append(validationErrors, validation.IsDNS1035Label(strings.ToLower(opts.Kind))...)

	if len(validationErrors) != 0 {
		return fmt.Errorf("invalid Kind: %#v", validationErrors)
	}

	// TODO: validate plural strings if provided

	return nil
}

// GVK returns the group-version-kind information to check against tracked resources in the configuration file
func (opts *Options) GVK() config.GVK {
	return config.GVK{
		Group:   opts.Group,
		Version: opts.Version,
		Kind:    opts.Kind,
	}
}

// safeImport returns a cleaned version of the provided string that can be used for imports
func (opts *Options) safeImport(unsafe string) string {
	safe := unsafe

	// Remove dashes and dots
	safe = strings.Replace(safe, "-", "", -1)
	safe = strings.Replace(safe, ".", "", -1)

	return safe
}

// NewResource creates a new resource from the options
func (opts *Options) NewResource(c *config.Config, doResource bool) *Resource {
	res := opts.newResource()

	replacer := res.Replacer()

	pkg := replacer.Replace(path.Join(c.Repo, "api", "%[version]"))
	if c.MultiGroup {
		pkg = replacer.Replace(path.Join(c.Repo, "apis", "%[group]", "%[version]"))
	}
	domain := c.Domain

	// pkg and domain may need to be changed in case we are referring to a builtin core resource:
	//  - Check if we are scaffolding the resource now           => project resource
	//  - Check if we already scaffolded the resource            => project resource
	//  - Check if the resource group is a well-known core group => builtin core resource
	//  - In any other case, default to                          => project resource
	// TODO: need to support '--resource-pkg-path' flag for specifying resourcePath
	if !doResource {
		if !c.HasResource(opts.GVK()) {
			if coreDomain, found := coreGroups[opts.Group]; found {
				pkg = replacer.Replace(path.Join("k8s.io", "api", "%[group]", "%[version]"))
				domain = coreDomain
			}
		}
	}

	res.Package = pkg
	res.Domain = opts.Group
	if domain != "" {
		res.Domain += "." + domain
	}

	return res
}

func (opts *Options) newResource() *Resource {
	// If not provided, compute a plural for for Kind
	plural := opts.Plural
	if plural == "" {
		plural = flect.Pluralize(strings.ToLower(opts.Kind))
	}

	return &Resource{
		Namespaced:       opts.Namespaced,
		Group:            opts.Group,
		GroupPackageName: opts.safeImport(opts.Group),
		Version:          opts.Version,
		Kind:             opts.Kind,
		Plural:           plural,
		ImportAlias:      opts.safeImport(opts.Group + opts.Version),
	}
}
