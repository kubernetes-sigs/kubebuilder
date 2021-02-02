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
	"strings"

	newconfig "sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

const (
	groupPresent    = "group flag present but empty"
	versionPresent  = "version flag present but empty"
	kindPresent     = "kind flag present but empty"
	groupRequired   = "group cannot be empty if the domain is empty"
	versionRequired = "version cannot be empty"
	kindRequired    = "kind cannot be empty"
)

var (
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
	if len(opts.Group) == 0 && len(opts.Domain) == 0 {
		return fmt.Errorf(groupRequired)
	}
	if len(opts.Version) == 0 {
		return fmt.Errorf(versionRequired)
	}
	if len(opts.Kind) == 0 {
		return fmt.Errorf(kindRequired)
	}

	return nil
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
	options := make([]resource.Option, 0)

	if opts.Plural != "" {
		options = append(options, resource.WithPlural(opts.Plural))
	}

	options = append(options, resource.WithLocalPath(c.GetRepository(), c.IsMultiGroup()))

	if opts.DoAPI {
		options = append(options, resource.ScaffoldAPI(opts.CRDVersion), resource.WithScope(opts.Namespaced))
	}

	if opts.DoController {
		options = append(options, resource.ScaffoldController())
	}

	if opts.DoDefaulting {
		options = append(options, resource.ScaffoldDefaultingWebhook(opts.WebhookVersion))
	}

	if opts.DoValidation {
		options = append(options, resource.ScaffoldValidationWebhook(opts.WebhookVersion))
	}

	if opts.DoConversion {
		options = append(options, resource.ScaffoldConversionWebhook(opts.WebhookVersion))
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
				opts.Domain = domain
				options = append(options, resource.WithBuiltInPath())
			}
		}
	}

	res, _ := resource.New(opts.GVK(), options...)

	return res
}
