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
	log "log/slog"
	"path"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var coreGroups = map[string]string{
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

// Options contains the information required to build a new resource.Resource.
type Options struct {
	// Plural is the resource's kind plural form.
	Plural string

	// ExternalAPIPath allows to inform a path for APIs not defined in the project
	ExternalAPIPath string

	// ExternalAPIDomain allows to inform the resource domain to build the Qualified Group
	// to generate the RBAC markers
	ExternalAPIDomain string

	// ExternalAPIModule specifies the Go module path for the external API with optional version.
	// Example: github.com/cert-manager/cert-manager@v1.18.2
	ExternalAPIModule string

	// Namespaced is true if the resource should be namespaced.
	Namespaced bool

	// SSA is true if Server-Side Apply should be enabled for the API.
	SSA bool

	// Flags that define which parts should be scaffolded
	DoAPI        bool
	DoController bool
	DoDefaulting bool
	DoValidation bool
	DoConversion bool

	// ControllerName is the name of the controller to scaffold.
	// This is used when creating multiple controllers for the same resource (GVK).
	// If not provided, a default name based on the resource kind will be used.
	ControllerName string

	// Spoke versions for conversion webhook
	Spoke []string

	// DefaultingPath is the custom path for the defaulting/mutating webhook
	DefaultingPath string

	// ValidationPath is the custom path for the validation webhook
	ValidationPath string
}

// UpdateResource updates the provided resource with the options
func (opts Options) UpdateResource(res *resource.Resource, c config.Config) {
	if opts.Plural != "" {
		res.Plural = opts.Plural
	}

	if opts.DoAPI {
		res.Path = resource.APIPackagePath(c.GetRepository(), res.Group, res.Version, c.IsMultiGroup())

		res.API = &resource.API{
			CRDVersion: "v1",
			Namespaced: opts.Namespaced,
			SSA:        opts.SSA,
		}
	}

	if opts.DoController {
		opts.updateControllers(res)
	}

	if opts.DoDefaulting || opts.DoValidation || opts.DoConversion {
		res.Path = resource.APIPackagePath(c.GetRepository(), res.Group, res.Version, c.IsMultiGroup())

		res.Webhooks.WebhookVersion = "v1"
		if opts.DoDefaulting {
			res.Webhooks.Defaulting = true
			if opts.DefaultingPath != "" {
				res.Webhooks.DefaultingPath = opts.DefaultingPath
			}
		}
		if opts.DoValidation {
			res.Webhooks.Validation = true
			if opts.ValidationPath != "" {
				res.Webhooks.ValidationPath = opts.ValidationPath
			}
		}
		if opts.DoConversion {
			res.Webhooks.Conversion = true
			res.Webhooks.Spoke = opts.Spoke
		}
	}

	if len(opts.ExternalAPIPath) > 0 {
		res.External = true
		res.Path = opts.ExternalAPIPath
		if len(opts.ExternalAPIDomain) > 0 {
			res.Domain = opts.ExternalAPIDomain
		}
		// Store module path if provided
		if len(opts.ExternalAPIModule) > 0 {
			res.Module = opts.ExternalAPIModule
		}
	}

	// domain and path may need to be changed in case we are referring to a builtin core resource:
	//  - Check if we are scaffolding the resource now           => project resource
	//  - Check if we already scaffolded the resource            => project resource
	//  - Check if the resource group is a well-known core group => builtin core resource
	//  - In any other case, default to                          => project resource
	if !opts.DoAPI {
		var alreadyHasAPI bool
		loadedRes, err := c.GetResource(res.GVK)
		alreadyHasAPI = err == nil && loadedRes.HasAPI()
		if !alreadyHasAPI {
			if res.External {
				res.Path = opts.ExternalAPIPath
				res.Domain = opts.ExternalAPIDomain
			} else {
				// Handle core types
				if domain, found := coreGroups[res.Group]; found {
					res.Core = true
					res.Domain = domain
					res.Path = path.Join("k8s.io", "api", res.Group, res.Version)
				}
			}
		}
	}
}

// updateControllers applies controller-related options to the resource.
// It handles both legacy (--controller) and new (--controller-name) controller creation.
func (opts Options) updateControllers(res *resource.Resource) {
	if opts.ControllerName == "" {
		// No controller name specified: use legacy mode
		if res.Controllers == nil || res.Controllers.IsEmpty() {
			res.Controller = true
		} else {
			// Warn when trying to use legacy mode on a resource with named controllers
			log.Warn("resource already has named controllers; use --controller-name to add another controller")
		}
		return
	}

	// Controller name specified: migrate from legacy format if needed
	if res.Controller {
		if res.Controllers == nil {
			res.Controllers = &resource.Controllers{}
		}
		// Convert the legacy controller: true to a named controller
		defaultName := strings.ToLower(res.Kind)
		_ = res.Controllers.AddController(defaultName)
		res.Controller = false
	}

	// Initialize controllers array if not yet created
	if res.Controllers == nil {
		res.Controllers = &resource.Controllers{}
	}

	// Add the new named controller (AddController validates and checks for duplicates)
	_ = res.Controllers.AddController(opts.ControllerName)
}
