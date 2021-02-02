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

package resource

import (
	"fmt"
	"path"
)

// Option defines the type that .
type Option func(*Resource) error

// New creates a new Resource.
func New(gvk GVK, options ...Option) (Resource, error) {
	resource := Resource{
		GVK:      gvk,
		Plural:   RegularPlural(gvk.Kind),
		API:      &API{},      // Make sure that the pointer is not nil to prevent pointer dereference errors
		Webhooks: &Webhooks{}, // Make sure that the pointer is not nil to prevent pointer dereference errors
	}

	for _, option := range options {
		if err := option(&resource); err != nil {
			return resource, err
		}
	}

	return resource, nil
}

// WithPlural sets the resource plural form.
func WithPlural(plural string) Option {
	return func(resource *Resource) error {
		resource.Plural = plural
		return nil
	}
}

// WithPath sets the resource path.
func WithPath(path string) Option {
	return func(resource *Resource) error {
		resource.Path = path
		return nil
	}
}

// WithLocalPath sets the resource path to a local location.
func WithLocalPath(repo string, isMultiGroup bool) Option {
	return func(resource *Resource) error {
		resource.Path = APIPackagePath(repo, resource.Group, resource.Version, isMultiGroup)
		return nil
	}
}

// WithBuiltInPath sets the resource path to a core resource location.
func WithBuiltInPath() Option {
	return func(resource *Resource) error {
		resource.Path = path.Join("k8s.io", "api", resource.Group, resource.Version)
		return nil
	}
}

// ScaffoldAPI specifies if a Resource should scaffold the API types.
func ScaffoldAPI(version string) Option {
	return func(resource *Resource) error {
		resource.API.CRDVersion = version
		return nil
	}
}

// WithScope allows to define the scope of the Resource: namespaced or cluster-scoped.
func WithScope(namespaced bool) Option {
	return func(resource *Resource) error {
		resource.API.Namespaced = namespaced
		return nil
	}
}

// ScaffoldController specifies if a Resource should scaffold the controller.
func ScaffoldController() Option {
	return func(resource *Resource) error {
		resource.Controller = true
		return nil
	}
}

var errWebhookVersion = fmt.Errorf("unable to scaffold several webhooks with different versions")

// ScaffoldDefaultingWebhook specifies if a Resource should scaffold a defaulting webhook.
func ScaffoldDefaultingWebhook(version string) Option {
	return func(resource *Resource) error {
		if resource.Webhooks.WebhookVersion == "" {
			resource.Webhooks.WebhookVersion = version
		} else if resource.Webhooks.WebhookVersion != version {
			return errWebhookVersion
		}

		resource.Webhooks.Defaulting = true
		return nil
	}
}

// ScaffoldValidationWebhook specifies if a Resource should scaffold a validation webhook.
func ScaffoldValidationWebhook(version string) Option {
	return func(resource *Resource) error {
		if resource.Webhooks.WebhookVersion == "" {
			resource.Webhooks.WebhookVersion = version
		} else if resource.Webhooks.WebhookVersion != version {
			return errWebhookVersion
		}

		resource.Webhooks.Validation = true
		return nil
	}
}

// ScaffoldConversionWebhook specifies if a Resource should scaffold a conversion webhook.
func ScaffoldConversionWebhook(version string) Option {
	return func(resource *Resource) error {
		if resource.Webhooks.WebhookVersion == "" {
			resource.Webhooks.WebhookVersion = version
		} else if resource.Webhooks.WebhookVersion != version {
			return errWebhookVersion
		}

		resource.Webhooks.Conversion = true
		return nil
	}
}
