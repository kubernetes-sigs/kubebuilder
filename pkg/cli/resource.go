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

package cli

import (
	"errors"
	"strings"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

const (
	groupPresent   = "group flag present but empty"
	versionPresent = "version flag present but empty"
	kindPresent    = "kind flag present but empty"
)

// resourceOptions contains the information required to build a new resource.Resource.
type resourceOptions struct {
	resource.GVK

	// Webhook holds the webhook configuration for multi-GVK webhooks
	// (mutually exclusive with GVK fields).
	Webhook resource.Webhook
}

func bindResourceFlags(fs *pflag.FlagSet) *resourceOptions {
	options := &resourceOptions{}

	fs.StringVar(&options.Group, "group", "", "Resource Group (e.g., batch, apps)")
	fs.StringVar(&options.Version, "version", "", "Resource Version (e.g., v1, v1beta1)")
	fs.StringVar(&options.Kind, "kind", "", "Resource Kind (e.g., CronJob, Deployment)")

	// Multi-GVK webhook flags (for webhooks that intercept multiple resource types)
	fs.StringVar(&options.Webhook.Name, "name", "",
		"Name for a webhook that intercepts multiple resource types. "+
			"Use with --groups, --kinds, and --versions instead of --group, --version, --kind")
	fs.StringSliceVar(&options.Webhook.Groups, "groups", nil,
		"Comma-separated API groups the webhook intercepts (e.g., 'apps,batch'). Use \"\" for the core group")
	fs.StringSliceVar(&options.Webhook.Kinds, "kinds", nil,
		"Comma-separated resource kinds the webhook intercepts (e.g., 'Pod,Deployment')")
	fs.StringSliceVar(&options.Webhook.Versions, "versions", nil,
		"Comma-separated API versions the webhook intercepts, or '*' for all (e.g., 'v1,v1beta1')")

	return options
}

// isMultiGVKWebhook returns true if multi-GVK webhook flags were provided.
func (opts resourceOptions) isMultiGVKWebhook() bool {
	return opts.Webhook.Name != ""
}

// validate verifies that all the fields have valid values.
func (opts resourceOptions) validate() error {
	// In multi-GVK webhook mode, GVK flags are not required.
	if opts.isMultiGVKWebhook() {
		if len(opts.Webhook.Groups) == 0 {
			return errors.New("--groups is required with --name")
		}
		if len(opts.Webhook.Kinds) == 0 {
			return errors.New("--kinds is required with --name")
		}
		if len(opts.Webhook.Versions) == 0 {
			return errors.New("--versions is required with --name (use '*' for all)")
		}
		// Reject GVK flags when using multi-GVK webhook mode
		if opts.Version != "" || opts.Kind != "" {
			return errors.New("--version and --kind cannot be used with --name; " +
				"use --groups, --kinds, and --versions instead")
		}
		return nil
	}

	// Check that the required flags did not get a flag as their value.
	// We can safely look for a '-' as the first char as none of the fields accepts it.
	// NOTE: We must do this for all the required flags first or we may output the wrong
	// error as flags may seem to be missing because Cobra assigned them to another flag.
	if strings.HasPrefix(opts.Group, "-") {
		return errors.New(groupPresent)
	}
	if strings.HasPrefix(opts.Version, "-") {
		return errors.New(versionPresent)
	}
	if strings.HasPrefix(opts.Kind, "-") {
		return errors.New(kindPresent)
	}

	// Check that required GVK flags are present (only in non-multi-GVK webhook mode).
	if opts.Version == "" {
		return errors.New(versionPresent)
	}
	if opts.Kind == "" {
		return errors.New(kindPresent)
	}

	return nil
}

// newResource creates a new resource from the options. Always returns a non-nil
// *resource.Resource. When the options represent a multi-GVK webhook (no GVK),
// the returned Resource carries the Webhook field set instead.
func (opts resourceOptions) newResource() *resource.Resource {
	res := &resource.Resource{
		GVK: resource.GVK{ // Remove whitespaces to prevent values like " " pass validation
			Group:   strings.TrimSpace(opts.Group),
			Domain:  strings.TrimSpace(opts.Domain),
			Version: strings.TrimSpace(opts.Version),
			Kind:    strings.TrimSpace(opts.Kind),
		},
		Plural:   resource.RegularPlural(opts.Kind),
		API:      &resource.API{},
		Webhooks: &resource.Webhooks{},
	}

	if opts.isMultiGVKWebhook() {
		wh := resource.Webhook{
			Name:           opts.Webhook.Name,
			WebhookVersion: "v1",
			Groups:         trimSlice(opts.Webhook.Groups),
			Kinds:          trimSlice(opts.Webhook.Kinds),
			Versions:       trimSlice(opts.Webhook.Versions),
		}
		res.Webhook = &wh
	}

	return res
}

func trimSlice(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = strings.TrimSpace(v)
	}
	return out
}
