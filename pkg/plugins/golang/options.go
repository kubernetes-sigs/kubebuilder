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
	"fmt"
	log "log/slog"
	"path"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

const k8sIODomainSuffix = "k8s.io"

var coreGroups = map[string]string{
	"admission":             k8sIODomainSuffix,
	"admissionregistration": k8sIODomainSuffix,
	"apps":                  "",
	"auditregistration":     k8sIODomainSuffix,
	"apiextensions":         k8sIODomainSuffix,
	"authentication":        k8sIODomainSuffix,
	"authorization":         k8sIODomainSuffix,
	"autoscaling":           "",
	"batch":                 "",
	"certificates":          k8sIODomainSuffix,
	"coordination":          k8sIODomainSuffix,
	"core":                  "",
	"events":                k8sIODomainSuffix,
	"extensions":            "",
	"imagepolicy":           k8sIODomainSuffix,
	"networking":            k8sIODomainSuffix,
	"node":                  k8sIODomainSuffix,
	"metrics":               k8sIODomainSuffix,
	"policy":                "",
	"rbac.authorization":    k8sIODomainSuffix,
	"scheduling":            k8sIODomainSuffix,
	"setting":               k8sIODomainSuffix,
	"storage":               k8sIODomainSuffix,
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
	//
	// Alpha: part of the Server-Side Apply (--ssa) alpha feature and may change in future releases.
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
		if !res.External {
			res.Path = resource.APIPackagePath(c.GetRepository(), res.Group, res.Version, c.IsMultiGroup())
		}

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
				if len(opts.ExternalAPIPath) > 0 {
					res.Path = opts.ExternalAPIPath
				}
				if len(opts.ExternalAPIDomain) > 0 {
					res.Domain = opts.ExternalAPIDomain
				}
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

// updateControllers records the controller in the Controllers list.
// UpdateResource has no error return, so failures are logged.
func (opts Options) updateControllers(res *resource.Resource) {
	if err := res.Migrate(); err != nil {
		log.Error("failed to migrate resource", "kind", res.Kind, "error", err)
		return
	}

	if res.Controllers == nil {
		res.Controllers = &resource.Controllers{}
	}

	name := opts.ControllerName
	if name == "" {
		name = resource.DefaultControllerName(res.Kind)
	}
	if !res.Controllers.HasController(name) {
		if err := res.Controllers.AddController(name); err != nil {
			log.Error("failed to record controller", "name", name, "kind", res.Kind, "error", err)
		}
	}
}

// ValidateControllerName rejects the controller these options request when the resource records
// another controller that generates the same reconciler, or when its reconciler name or file
// name collides with a controller of another resource in the same package, including default
// names. A controller that versions of a kind share is not a collision when the requested
// resource records it too.
func (opts Options) ValidateControllerName(res *resource.Resource, c config.Config) error {
	name := opts.ControllerName
	if name == "" {
		name = resource.DefaultControllerName(res.Kind)
	}

	if err := checkOwnControllers(c, res, name); err != nil {
		return err
	}

	return checkOtherControllers(c, res, name)
}

// checkOwnControllers rejects a name that generates the reconciler of another controller the
// resource already records.
func checkOwnControllers(c config.Config, res *resource.Resource, name string) error {
	stored, err := c.GetResource(res.GVK)
	if err != nil {
		// The resource is not recorded yet, so it has no controller to collide with.
		return nil
	}

	reconciler := resource.NormalizeReconcilerName(name, res.Kind)
	for _, recorded := range stored.GetControllerNames() {
		if recorded != name && resource.NormalizeReconcilerName(recorded, res.Kind) == reconciler {
			return fmt.Errorf("controller name %q conflicts with existing controller %q: both generate %s",
				name, recorded, reconciler)
		}
	}

	return nil
}

// checkOtherControllers rejects a name that generates the file or the reconciler of a controller
// of another resource in the same package.
func checkOtherControllers(c config.Config, res *resource.Resource, name string) error {
	resources, err := c.GetResources()
	if err != nil {
		return fmt.Errorf("failed to get resources to check controller names: %w", err)
	}

	reconciler := resource.NormalizeReconcilerName(name, res.Kind)
	// Versions of a kind may share 1 controller that each of them records. Scaffolding it again
	// for a version that records it does not clash with the other versions.
	stored, err := c.GetResource(res.GVK)
	recordsIt := err == nil && stored.HasControllerWithName(name)

	for _, other := range resources {
		// Each group has its own controller package in a multigroup project.
		if other.IsEqualTo(res.GVK) || (c.IsMultiGroup() && other.Group != res.Group) {
			continue
		}
		sameKind := other.Group == res.Group && other.Domain == res.Domain && other.Kind == res.Kind

		for _, recorded := range other.GetControllerNames() {
			if recordsIt && sameKind && recorded == name {
				continue
			}
			if resource.NormalizeReconcilerName(recorded, other.Kind) != reconciler {
				if recorded == name {
					return fmt.Errorf("controller name %q conflicts with controller of %s/%s/%s: "+
						"both generate %s_controller.go in the same package; use '--controller-name' to pick another name",
						name, other.Group, other.Version, other.Kind, resource.NormalizeFileName(name))
				}
				continue
			}

			return fmt.Errorf(
				"controller %q conflicts with controller %q of %s/%s/%s: "+
					"both generate %s in the same package; use '--controller-name' to pick "+
					"another name, or '--controller=false' if that controller already reconciles this resource",
				name, recorded, other.Group, other.Version, other.Kind, reconciler,
			)
		}
	}

	return nil
}
