/*
Copyright 2026 The Kubernetes Authors.

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

package v4

import (
	"errors"
	"fmt"
	log "log/slog"
	"slices"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

// resolveTrackedResource copies into the resource the values that the PROJECT file tracks for it,
// such as the domain and the path of an API defined outside the project, which the Group, Version
// and Kind flags cannot carry. Picking which resource they belong to is up to resource.Select.
func resolveTrackedResource(c config.Config, opts *goPlugin.Options, res *resource.Resource) error {
	opts.ExternalAPIDomain = strings.TrimSpace(opts.ExternalAPIDomain)
	opts.ExternalAPIPath = strings.TrimSpace(opts.ExternalAPIPath)

	tracked, err := c.GetResources()
	if err != nil {
		return fmt.Errorf("failed to load resources from project configuration: %w", err)
	}

	selected, err := resource.Select(tracked, res.GVK, opts.ExternalAPIDomain, opts.ExternalAPIPath)
	switch {
	case err != nil:
		return selectionError(err)
	case selected != nil:
		if selected.IsExternal() {
			log.Info("Using the external API tracked in the PROJECT file",
				"domain", selected.Domain, "path", selected.Path)
		}
		res.AdoptTracked(*selected)
	case opts.ExternalAPIDomain != "" && opts.ExternalAPIPath != "":
		// The resource is not tracked yet and the flags describe an API defined outside the project.
		if matches := resource.MatchGVK(tracked, res.GVK); len(matches) > 0 {
			log.Warn("Recording a new resource: the PROJECT file already tracks this Group, "+
				"Version and Kind", "domain", opts.ExternalAPIDomain,
				"tracked", selectionChoices(matches))
		}
		res.Domain = opts.ExternalAPIDomain
	}

	return nil
}

// selectionError says with which flags the user tells apart the resources that the Group, the
// Version and the Kind do not.
func selectionError(err error) error {
	var ambiguous resource.AmbiguousSelectionError
	if errors.As(err, &ambiguous) {
		return fmt.Errorf("the PROJECT file tracks %d resources for %s/%s, Kind %s. Select the "+
			"one you want with %s. To scaffold another one, pass '--external-api-domain' with a "+
			"new domain and '--external-api-path'",
			len(ambiguous.Matches), ambiguous.GVK.Group, ambiguous.GVK.Version, ambiguous.GVK.Kind,
			selectionChoices(ambiguous.Matches))
	}

	var notFound resource.DomainNotFoundError
	if errors.As(err, &notFound) {
		return fmt.Errorf("no resource found for %s/%s, Kind %s with the domain %q. The PROJECT "+
			"file tracks it with %s. Add '--external-api-path' to scaffold a new resource for the "+
			"domain %q",
			notFound.GVK.Group, notFound.GVK.Version, notFound.GVK.Kind, notFound.Domain,
			selectionChoices(notFound.Matches), notFound.Domain)
	}

	var mismatch resource.PathMismatchError
	if errors.As(err, &mismatch) {
		return fmt.Errorf("the PROJECT file already tracks %s/%s, Kind %s with the path %q, "+
			"reached with %s. Drop '--external-api-path' to work on it, or pass "+
			"'--external-api-domain' with another domain to scaffold a second resource for the "+
			"same Group, Version and Kind",
			mismatch.Tracked.Group, mismatch.Tracked.Version, mismatch.Tracked.Kind,
			mismatch.Tracked.Path, selectionChoices([]resource.Resource{mismatch.Tracked}))
	}

	return err
}

// selectionChoices lists how to reach each of the given resources, with the API group it builds.
// The API of the project needs no flag, and a core type of a group without a domain, such as apps,
// cannot be named yet. The list is sorted so that the errors do not depend on the order of the
// PROJECT file.
func selectionChoices(resources []resource.Resource) string {
	choices := make([]string, 0, len(resources))
	for _, res := range resources {
		switch {
		case res.IsDefinedInProject():
			choices = append(choices, fmt.Sprintf("no flag (%s, the API of your project)", res.QualifiedGroup()))
		case res.Domain == "":
			choices = append(choices, fmt.Sprintf("no flag yet (%s, a Kubernetes API without a domain)",
				res.QualifiedGroup()))
		default:
			choices = append(choices, fmt.Sprintf("'--external-api-domain %s' (%s)", res.Domain, res.QualifiedGroup()))
		}
	}
	slices.Sort(choices)

	return strings.Join(choices, ", ")
}
