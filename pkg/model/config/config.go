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

package config

import (
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

const (
	// Scaffolding versions
	Version1 = "1"
	Version2 = "2"
)

// Config is the unmarshalled representation of the configuration file
type Config struct {
	// Version is the project version, defaults to "1" (backwards compatibility)
	Version string `json:"version,omitempty"`

	// Domain is the domain associated with the project and used for API groups
	Domain string `json:"domain,omitempty"`

	// Repo is the go package name of the project root
	Repo string `json:"repo,omitempty"`

	// Resources tracks scaffolded resources in the project
	// This info is tracked only in project with version 2
	Resources []GVK `json:"resources,omitempty"`

	// Multigroup tracks if the project has more than one group
	MultiGroup bool `json:"multigroup,omitempty"`
}

// IsV1 returns true if it is a v1 project
func (config Config) IsV1() bool {
	return config.Version == Version1
}

// IsV2 returns true if it is a v2 project
func (config Config) IsV2() bool {
	return config.Version == Version2
}

// ResourceGroups returns unique groups of scaffolded resources in the project
func (config Config) ResourceGroups() []string {
	groupSet := map[string]struct{}{}
	for _, r := range config.Resources {
		groupSet[r.Group] = struct{}{}
	}

	groups := make([]string, 0, len(groupSet))
	for g := range groupSet {
		groups = append(groups, g)
	}

	return groups
}

// HasResource returns true if API resource is already tracked
// NOTE: this works only for v2, since in v1 resources are not tracked
func (config Config) HasResource(target *resource.Resource) bool {
	// Short-circuit v1
	if config.Version == Version1 {
		return false
	}

	// Return true if the target resource is found in the tracked resources
	for _, r := range config.Resources {
		if r.isEqualTo(target) {
			return true
		}
	}

	// Return false otherwise
	return false
}

// AddResource appends the provided resource to the tracked ones
// It returns if the configuration was modified
// NOTE: this works only for v2, since in v1 resources are not tracked
func (config *Config) AddResource(r *resource.Resource) bool {
	// Short-circuit v1
	if config.Version == Version1 {
		return false
	}

	// No-op if the resource was already tracked, return false
	if config.HasResource(r) {
		return false
	}

	// Append the resource to the tracked ones, return true
	config.Resources = append(config.Resources,
		GVK{Group: r.Group, Version: r.Version, Kind: r.Kind})
	return true
}

// GVK contains information about scaffolded resources
type GVK struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

// isEqualTo compares it with another resource
func (r GVK) isEqualTo(other *resource.Resource) bool {
	// Prevent panic if other is nil
	if other == nil {
		return r.Group == "" &&
			r.Version == "" &&
			r.Kind == ""
	}

	return r.Group == other.Group &&
		r.Version == other.Version &&
		r.Kind == other.Kind
}
