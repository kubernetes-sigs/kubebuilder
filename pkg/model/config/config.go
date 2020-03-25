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
	"strings"
)

// Scaffolding versions
const (
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

	// Layout contains a key specifying which plugin created a project.
	Layout string `json:"layout,omitempty"`
}

// IsV1 returns true if it is a v1 project
func (config Config) IsV1() bool {
	return config.Version == Version1
}

// IsV2 returns true if it is a v2 project
func (config Config) IsV2() bool {
	return config.Version == Version2
}

// HasResource returns true if API resource is already tracked
func (config Config) HasResource(target GVK) bool {
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
// NOTE: in v1 resources are not tracked, so we return false
func (config *Config) AddResource(gvk GVK) bool {
	// Short-circuit v1
	if config.IsV1() {
		return false
	}

	// No-op if the resource was already tracked, return false
	if config.HasResource(gvk) {
		return false
	}

	// Append the resource to the tracked ones, return true
	config.Resources = append(config.Resources, gvk)
	return true
}

// HasGroup returns true if group is already tracked
func (config Config) HasGroup(group string) bool {
	// Return true if the target group is found in the tracked resources
	for _, r := range config.Resources {
		if strings.EqualFold(group, r.Group) {
			return true
		}
	}

	// Return false otherwise
	return false
}

// GVK contains information about scaffolded resources
type GVK struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

// isEqualTo compares it with another resource
func (r GVK) isEqualTo(other GVK) bool {
	return r.Group == other.Group &&
		r.Version == other.Version &&
		r.Kind == other.Kind
}
