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

package resource

import "fmt"

// AmbiguousSelectionError is returned when the Group, Version and Kind match more than one
// resource and neither the domain nor the path tells them apart.
type AmbiguousSelectionError struct {
	GVK     GVK
	Matches []Resource
}

// Error implements error interface
func (e AmbiguousSelectionError) Error() string {
	return fmt.Sprintf("%d resources match %s/%s, Kind %s", len(e.Matches), e.GVK.Group, e.GVK.Version, e.GVK.Kind)
}

// DomainNotFoundError is returned when no resource of the matching Group, Version and Kind has
// the given domain.
type DomainNotFoundError struct {
	GVK     GVK
	Domain  string
	Matches []Resource
}

// Error implements error interface
func (e DomainNotFoundError) Error() string {
	return fmt.Sprintf("no resource matches %s/%s, Kind %s with the domain %q",
		e.GVK.Group, e.GVK.Version, e.GVK.Kind, e.Domain)
}

// PathMismatchError is returned when the resource that the Group, Version and Kind identify has
// another path than the given one.
type PathMismatchError struct {
	Tracked Resource
	Path    string
}

// Error implements error interface
func (e PathMismatchError) Error() string {
	return fmt.Sprintf("resource %s/%s, Kind %s has the path %q, not %q",
		e.Tracked.Group, e.Tracked.Version, e.Tracked.Kind, e.Tracked.Path, e.Path)
}

// Select returns the resource of the list that the given GVK identifies, or nil when the list
// tracks none.
//
// A resource is identified by Group, Domain, Version and Kind, but the domain of an API defined
// outside the project is not always known, so it is left out of the match. More than one resource
// can then match, for example an Issuer from cert-manager.io and an Issuer from another vendor.
// The domain and the path tell them apart, the API of the project is taken when neither is given,
// and an error says why the rest cannot be told apart.
func Select(resources []Resource, gvk GVK, domain, apiPath string) (*Resource, error) {
	matches := MatchGVK(resources, gvk)
	if len(matches) == 0 {
		return nil, nil
	}

	if domain != "" {
		tracked := FindByDomain(matches, domain)
		if tracked == nil {
			// The domain and the path give a full identity that no resource has yet.
			if apiPath != "" {
				return nil, nil
			}

			return nil, DomainNotFoundError{GVK: gvk, Domain: domain, Matches: matches}
		}
		if apiPath != "" && tracked.Path != "" && tracked.Path != apiPath {
			return nil, PathMismatchError{Tracked: *tracked, Path: apiPath}
		}

		return tracked, nil
	}

	if tracked := FindByPath(matches, apiPath); tracked != nil {
		return tracked, nil
	}

	if len(matches) == 1 {
		if apiPath != "" && matches[0].Path != "" {
			return nil, PathMismatchError{Tracked: matches[0], Path: apiPath}
		}

		return &matches[0], nil
	}

	// The domain and the path name an API defined outside the project, so without them the
	// command works on the API of the project.
	if tracked := FindDefinedInProject(matches); tracked != nil {
		return tracked, nil
	}

	return nil, AmbiguousSelectionError{GVK: gvk, Matches: matches}
}

// MatchGVK returns the resources with the same Group, Version and Kind as the given GVK. The
// domain is left out because the caller does not always know it.
func MatchGVK(resources []Resource, gvk GVK) []Resource {
	var matches []Resource
	for _, res := range resources {
		if res.Group == gvk.Group && res.Version == gvk.Version && res.Kind == gvk.Kind {
			matches = append(matches, res)
		}
	}

	return matches
}

// FindByDomain returns the resource with the given domain, or nil when no resource has it.
func FindByDomain(resources []Resource, domain string) *Resource {
	for i := range resources {
		if resources[i].Domain == domain {
			return &resources[i]
		}
	}

	return nil
}

// FindByPath returns the only resource with the given path, or nil when the path is empty or does
// not single out one resource.
func FindByPath(resources []Resource, apiPath string) *Resource {
	if apiPath == "" {
		return nil
	}

	var found *Resource
	for i := range resources {
		if resources[i].Path != apiPath {
			continue
		}
		if found != nil {
			return nil
		}
		found = &resources[i]
	}

	return found
}

// FindDefinedInProject returns the only resource that the project defines itself, or nil when the
// resources are all defined outside it, or when more than one belongs to the project.
func FindDefinedInProject(resources []Resource) *Resource {
	var found *Resource
	for i := range resources {
		if !resources[i].IsDefinedInProject() {
			continue
		}
		if found != nil {
			return nil
		}
		found = &resources[i]
	}

	return found
}

// AdoptTracked copies the values that identify the tracked resource, such as the domain and the
// path of an external API, which the caller cannot infer from the Group, the Version and the Kind.
func (r *Resource) AdoptTracked(tracked Resource) {
	r.Domain = tracked.Domain
	r.Path = tracked.Path
	r.Plural = tracked.Plural
	r.External = tracked.External
	r.Core = tracked.Core
	r.Module = tracked.Module
}
