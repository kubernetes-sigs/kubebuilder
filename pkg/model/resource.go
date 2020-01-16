/*
Copyright 2020 The Kubernetes Authors.

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

package model

// Resource describes the resource currently being generated
// TODO: unify with pkg/scaffold/resource.Resource
type Resource struct {
	// Namespaced is true if the resource is namespaced
	Namespaced bool `json:"namespaces,omitempty"`

	// Group is the API Group.  Does not contain the domain.
	Group string `json:"group,omitempty"`

	// Version is the API version - e.g. v1beta1
	Version string `json:"version,omitempty"`

	// Kind is the API Kind.
	Kind string `json:"kind,omitempty"`

	// Plural is the plural lowercase of Kind.
	Plural string `json:"plural,omitempty"`

	// Resource is the API Resource.
	Resource string `json:"resource,omitempty"`

	// ResourcePackage is the go package of the Resource
	GoPackage string `json:"goPackage,omitempty"`

	// GroupDomain is the Group + "." + Domain for the Resource
	GroupDomain string `json:"groupDomain,omitempty"`
}
