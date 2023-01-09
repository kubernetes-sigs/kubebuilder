/*
Copyright 2022 The Kubernetes Authors.

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
	"path"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
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
	// Plural is the resource's kind plural form.
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

// UpdateResource updates the provided resource with the options
func (opts Options) UpdateResource(res *resource.Resource, c config.Config) {
	if opts.Plural != "" {
		res.Plural = opts.Plural
	}

	if opts.DoAPI {
		//nolint:staticcheck
		if plugin.IsLegacyLayout(c) {
			res.Path = resource.APIPackagePathLegacy(c.GetRepository(), res.Group, res.Version, c.IsMultiGroup())
		} else {
			res.Path = resource.APIPackagePath(c.GetRepository(), res.Group, res.Version, c.IsMultiGroup())
		}

		res.API = &resource.API{
			CRDVersion: opts.CRDVersion,
			Namespaced: opts.Namespaced,
		}

	}

	if opts.DoController {
		res.Controller = true
	}

	if opts.DoDefaulting || opts.DoValidation || opts.DoConversion {
		// IsLegacyLayout is added to ensure backwards compatibility and should
		// be removed when we remove the go/v3 plugin
		//nolint:staticcheck
		if plugin.IsLegacyLayout(c) {
			res.Path = resource.APIPackagePathLegacy(c.GetRepository(), res.Group, res.Version, c.IsMultiGroup())
		} else {
			res.Path = resource.APIPackagePath(c.GetRepository(), res.Group, res.Version, c.IsMultiGroup())
		}
		res.Webhooks.WebhookVersion = opts.WebhookVersion
		if opts.DoDefaulting {
			res.Webhooks.Defaulting = true
		}
		if opts.DoValidation {
			res.Webhooks.Validation = true
		}
		if opts.DoConversion {
			res.Webhooks.Conversion = true
		}
	}

	// domain and path may need to be changed in case we are referring to a builtin core resource:
	//  - Check if we are scaffolding the resource now           => project resource
	//  - Check if we already scaffolded the resource            => project resource
	//  - Check if the resource group is a well-known core group => builtin core resource
	//  - In any other case, default to                          => project resource
	// TODO: need to support '--resource-pkg-path' flag for specifying resourcePath
	if !opts.DoAPI {
		var alreadyHasAPI bool
		if c.GetVersion().Compare(cfgv2.Version) == 0 {
			alreadyHasAPI = c.HasResource(res.GVK)
		} else {
			loadedRes, err := c.GetResource(res.GVK)
			alreadyHasAPI = err == nil && loadedRes.HasAPI()
		}
		if !alreadyHasAPI {
			if domain, found := coreGroups[res.Group]; found {
				res.Domain = domain
				res.Path = path.Join("k8s.io", "api", res.Group, res.Version)
			}
		}
	}
}
