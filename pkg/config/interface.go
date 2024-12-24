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

package config

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// Config defines the interface that project configuration types must follow.
type Config interface {
	/* Version */

	// GetVersion returns the current project version.
	GetVersion() Version

	/* String fields */

	// GetDomain returns the project domain.
	GetDomain() string
	// SetDomain sets the project domain.
	SetDomain(domain string) error

	// GetRepository returns the project repository.
	GetRepository() string
	// SetRepository sets the project repository.
	SetRepository(repository string) error

	// GetProjectName returns the project name.
	// This method was introduced in project version 3.
	GetProjectName() string
	// SetProjectName sets the project name.
	// This method was introduced in project version 3.
	SetProjectName(name string) error

	// GetPluginChain returns the plugin chain.
	// This method was introduced in project version 3.
	GetPluginChain() []string
	// SetPluginChain sets the plugin chain.
	// This method was introduced in project version 3.
	SetPluginChain(pluginChain []string) error

	/* Boolean fields */

	// IsMultiGroup checks if multi-group is enabled.
	IsMultiGroup() bool
	// SetMultiGroup enables multi-group.
	SetMultiGroup() error
	// ClearMultiGroup disables multi-group.
	ClearMultiGroup() error

	/* Resources */

	// ResourcesLength returns the number of tracked resources.
	ResourcesLength() int
	// HasResource checks if the provided GVK is stored in the Config.
	HasResource(gvk resource.GVK) bool
	// GetResource returns the stored resource matching the provided GVK.
	GetResource(gvk resource.GVK) (resource.Resource, error)
	// GetResources returns all the stored resources.
	GetResources() ([]resource.Resource, error)
	// AddResource adds the provided resource if it was not present, no-op if it was already present.
	AddResource(res resource.Resource) error
	// UpdateResource adds the provided resource if it was not present, modifies it if it was already present.
	UpdateResource(res resource.Resource) error

	// HasGroup checks if the provided group is the same as any of the tracked resources.
	HasGroup(group string) bool
	// ListCRDVersions returns a list of the CRD versions in use by the tracked resources.
	ListCRDVersions() []string
	// ListWebhookVersions returns a list of the webhook versions in use by the tracked resources.
	ListWebhookVersions() []string

	/* Plugins */

	// DecodePluginConfig decodes a plugin config stored in Config into configObj, which must be a pointer.
	// This method is intended to be used for custom configuration objects, which were introduced in project version 3.
	DecodePluginConfig(key string, configObj interface{}) error
	// EncodePluginConfig encodes a config object into Config by overwriting the existing object stored under key.
	// This method is intended to be used for custom configuration objects, which were introduced in project version 3.
	EncodePluginConfig(key string, configObj interface{}) error

	/* Persistence */

	// MarshalYAML Marshal returns the YAML representation of the Config.
	MarshalYAML() ([]byte, error)
	// UnmarshalYAML Unmarshal loads the Config fields from its YAML representation.
	UnmarshalYAML([]byte) error
}
