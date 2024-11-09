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
}

func bindResourceFlags(fs *pflag.FlagSet) *resourceOptions {
	options := &resourceOptions{}

	fs.StringVar(&options.Group, "group", "", "resource Group")
	fs.StringVar(&options.Version, "version", "", "resource Version")
	fs.StringVar(&options.Kind, "kind", "", "resource Kind")

	return options
}

// validate verifies that all the fields have valid values.
func (opts resourceOptions) validate() error {
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

	// We do not check here if the GVK values are empty because that would
	// make them mandatory and some plugins may want to set default values.
	// Instead, this is checked by resource.GVK.Validate()

	return nil
}

// newResource creates a new resource from the options
func (opts resourceOptions) newResource() *resource.Resource {
	return &resource.Resource{
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
}
