/*
Copyright 2021 The Kubernetes Authors.

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

package golang

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	newconfig "sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/internal/validation"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

const (
	versionPattern  = "^v\\d+(?:alpha\\d+|beta\\d+)?$"
	groupPresent    = "group flag present but empty"
	versionPresent  = "version flag present but empty"
	kindPresent     = "kind flag present but empty"
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

// Options contains the information required to build a new resource.Resource.
type Options struct {
	// Group is the resource's group. Does not contain the domain.
	Group string
	// Domain is the resource's domain.
	Domain string
	// Version is the resource's version.
	Version string
	// Kind is the resource's kind.
	Kind string

	// Plural is the resource's kind plural form.
	// Optional
	Plural string

	// CRDVersion is the CustomResourceDefinition API version that will be used for the resource.
	CRDVersion string
	// WebhookVersion is the {Validating,Mutating}WebhookConfiguration API version that will be used for the resource.
	WebhookVersion string

	// Namespaced is true if the resource should be namespaced.
	Namespaced bool

	// Flags that define which parts should be scaffolded
	DoAPI        bool
	DoController bool
	DoDefaulting bool
	DoValidation bool
	DoConversion bool
}

// Validate verifies that all the fields have valid values
func (opts Options) Validate() error {
	// Check that the required flags did not get a flag as their value
	// We can safely look for a '-' as the first char as none of the fields accepts it
	// NOTE: We must do this for all the required flags first or we may output the wrong
	// error as flags may seem to be missing because Cobra assigned them to another flag.
	if strings.HasPrefix(opts.Group, "-") {
		return fmt.Errorf(groupPresent)
	}
	if strings.HasPrefix(opts.Version, "-") {
		return fmt.Errorf(versionPresent)
	}
	if strings.HasPrefix(opts.Kind, "-") {
		return fmt.Errorf(kindPresent)
	}
	// Now we can check that all the required flags are not empty
	if len(opts.Version) == 0 {
		return fmt.Errorf(versionRequired)
	}
	if len(opts.Kind) == 0 {
		return fmt.Errorf(kindRequired)
	}

	// Check if the qualified group has a valid DNS1123 subdomain value
	if err := validation.IsDNS1123Subdomain(opts.QualifiedGroup()); err != nil {
		return fmt.Errorf("either group or domain is invalid: (%v)", err)
	}

	// Check if the version follows the valid pattern
	if !versionRegex.MatchString(opts.Version) {
		return fmt.Errorf("version must match %s (was %s)", versionPattern, opts.Version)
	}

	validationErrors := []string{}

	// Require kind to start with an uppercase character
	if string(opts.Kind[0]) == strings.ToLower(string(opts.Kind[0])) {
		validationErrors = append(validationErrors, "kind must start with an uppercase character")
	}

	validationErrors = append(validationErrors, validation.IsDNS1035Label(strings.ToLower(opts.Kind))...)

	if len(validationErrors) != 0 {
		return fmt.Errorf("invalid Kind: %#v", validationErrors)
	}

	// TODO: validate plural strings if provided

	// Ensure apiVersions for k8s types are empty or valid.
	for typ, apiVersion := range map[string]string{
		"CRD":     opts.CRDVersion,
		"Webhook": opts.WebhookVersion,
	} {
		switch apiVersion {
		case "", "v1", "v1beta1":
		default:
			return fmt.Errorf("%s version must be one of: v1, v1beta1", typ)
		}
	}

	return nil
}

// QualifiedGroup returns the fully qualified group name with the available information.
func (opts Options) QualifiedGroup() string {
	switch "" {
	case opts.Domain:
		return opts.Group
	case opts.Group:
		return opts.Domain
	default:
		return fmt.Sprintf("%s.%s", opts.Group, opts.Domain)
	}
}

// GVK returns the GVK identifier of a resource.
func (opts Options) GVK() resource.GVK {
	return resource.GVK{
		Group:   opts.Group,
		Domain:  opts.Domain,
		Version: opts.Version,
		Kind:    opts.Kind,
	}
}

// NewResource creates a new resource from the options
func (opts Options) NewResource(c newconfig.Config) resource.Resource {
	res := resource.Resource{
		GVK:        opts.GVK(),
		Path:       resource.APIPackagePath(c.GetRepository(), opts.Group, opts.Version, c.IsMultiGroup()),
		Controller: opts.DoController,
	}

	if opts.Plural != "" {
		res.Plural = opts.Plural
	} else {
		// If not provided, compute a plural for Kind
		res.Plural = resource.RegularPlural(opts.Kind)
	}

	if opts.DoAPI {
		res.API = &resource.API{
			CRDVersion: opts.CRDVersion,
			Namespaced: opts.Namespaced,
		}
	} else {
		// Make sure that the pointer is not nil to prevent pointer dereference errors
		res.API = &resource.API{}
	}

	if opts.DoDefaulting || opts.DoValidation || opts.DoConversion {
		res.Webhooks = &resource.Webhooks{
			WebhookVersion: opts.WebhookVersion,
			Defaulting:     opts.DoDefaulting,
			Validation:     opts.DoValidation,
			Conversion:     opts.DoConversion,
		}
	} else {
		// Make sure that the pointer is not nil to prevent pointer dereference errors
		res.Webhooks = &resource.Webhooks{}
	}

	// domain and path may need to be changed in case we are referring to a builtin core resource:
	//  - Check if we are scaffolding the resource now           => project resource
	//  - Check if we already scaffolded the resource            => project resource
	//  - Check if the resource group is a well-known core group => builtin core resource
	//  - In any other case, default to                          => project resource
	// TODO: need to support '--resource-pkg-path' flag for specifying resourcePath
	if !opts.DoAPI {
		if !c.HasResource(opts.GVK()) {
			if domain, found := coreGroups[opts.Group]; found {
				res.Domain = domain
				res.Path = path.Join("k8s.io", "api", opts.Group, opts.Version)
			}
		}
	}

	return res
}
