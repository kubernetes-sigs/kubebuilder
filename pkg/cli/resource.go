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
	"fmt"
	log "log/slog"
	"slices"
	"strings"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

const (
	groupPresent   = "group flag present but empty"
	versionPresent = "version flag present but empty"
	kindPresent    = "kind flag present but empty"
	domainPresent  = "domain flag present but empty"
)

// resourceOptions contains the information required to build a new resource.Resource.
type resourceOptions struct {
	resource.GVK
}

func bindResourceFlags(fs *pflag.FlagSet) *resourceOptions {
	options := &resourceOptions{}

	fs.StringVar(&options.Group, "group", "", "Resource Group (e.g., batch, apps)")
	fs.StringVar(&options.Version, "version", "", "Resource Version (e.g., v1, v1beta1)")
	fs.StringVar(&options.Kind, "kind", "", "Resource Kind (e.g., CronJob, Deployment)")
	fs.StringVar(&options.Domain, "domain", "",
		"Resource Domain (e.g., cert-manager.io). Overrides the project domain for this resource; "+
			"required to disambiguate when multiple existing resources share the same Group/Version/Kind")

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
	if strings.HasPrefix(opts.Domain, "-") {
		return errors.New(domainPresent)
	}

	// We do not check here if the GVK values are empty because that would
	// make them mandatory and some plugins may want to set default values.
	// Instead, this is checked by resource.GVK.Validate()

	return nil
}

// trackedDomains returns the Domain of every tracked resource sharing the
// Group/Version/Kind in opts, in project file order.
func (opts resourceOptions) trackedDomains(c config.Config) []string {
	resources, _ := c.GetResources()
	var domains []string
	for _, r := range resources {
		if r.Group == opts.Group && r.Version == opts.Version && r.Kind == opts.Kind {
			domains = append(domains, r.Domain)
		}
	}
	return domains
}

// resolveDomain returns the Domain to build the resource with. An explicit --domain wins.
// Otherwise a single tracked Group/Version/Kind match lends its own Domain, which is how
// an external resource (e.g. cert-manager.io) is found when only G/V/K is on the CLI, and
// with no match at all the project domain applies.
//
// Several matches resolve to nothing rather than to one of them, since picking would
// depend on the order the project file lists them in. A plugin may still name one from
// its own flags while injecting the resource, so the verdict waits for checkDomain.
func (opts resourceOptions) resolveDomain(tracked []string, projectDomain string) string {
	if opts.Domain != "" {
		return opts.Domain
	}
	switch len(tracked) {
	case 0:
		return projectDomain
	case 1:
		return tracked[0]
	default:
		return ""
	}
}

// checkDomain judges the Domain the resource ended up with, once every plugin has had a
// chance to name one.
//
// A domain matching none of the tracked resources was asked for, so it stands: it records
// a new resource beside them, which is allowed but easy to do by accident, so it is said
// out loud. Nothing at all means the ambiguity went unresolved, and the command is refused
// rather than working on a resource the user never named.
func (opts resourceOptions) checkDomain(tracked []string, resolved string) error {
	if len(tracked) == 0 || slices.Contains(tracked, resolved) {
		return nil
	}

	groups := strings.Join(qualifiedGroups(opts.Group, tracked), ", ")
	if resolved == "" {
		return fmt.Errorf(
			"group %q, version %q and kind %q match more than one resource (%s): "+
				"pass --domain to choose the one to work on",
			opts.Group, opts.Version, opts.Kind, groups,
		)
	}

	log.Warn("Recording a new resource; the project already tracks this group, version and kind",
		"recording", resource.GVK{Group: opts.Group, Domain: resolved}.QualifiedGroup(),
		"tracked", groups,
	)
	return nil
}

// qualifiedGroups renders each domain as the qualified group it produces, so the user
// reads the same strings the project file and the RBAC markers show.
func qualifiedGroups(group string, domains []string) []string {
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		out = append(out, resource.GVK{Group: group, Domain: d}.QualifiedGroup())
	}
	return out
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
